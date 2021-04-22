package cache

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/veecue/pacman-smartmirror/database"
	"github.com/veecue/pacman-smartmirror/mirrorlist"
	"github.com/veecue/pacman-smartmirror/packet"
	"github.com/veecue/pacman-smartmirror/test"

	_ "github.com/veecue/pacman-smartmirror/impl/pacman"
)

const (
	_filename = "linux-5.2.arch2-1-x86_64.pkg.tar.xz"
	_repo     = "core"
	_arch     = "x86_64"
)

var (
	_content = strings.Repeat("heyoo", 100)
)

func getSize(t *testing.T, s io.Seeker) int {
	n, err := s.Seek(0, io.SeekEnd)
	assert.NoError(t, err)
	_, err = s.Seek(0, io.SeekStart)
	assert.NoError(t, err)
	return int(n)
}

func TestSimple(t *testing.T) {
	var calls int32
	s := test.NewServer(t, func(w http.ResponseWriter, filename string, repo string, arch string) {
		if atomic.AddInt32(&calls, 1) > 1 {
			assert.Fail(t, "Server called to often")
		}
		assert.Equal(t, _filename, filename)
		assert.Equal(t, _repo, repo)
		assert.Equal(t, _arch, arch)
		http.ServeContent(w, &http.Request{}, "a.tar.xz", time.Time{}, strings.NewReader(_content))
	})
	defer s.StopServer(t)

	dir, err := ioutil.TempDir("", "smartmirror-test")
	assert.NoError(t, err)
	assert.NoError(t, os.RemoveAll(dir))
	assert.NoError(t, os.MkdirAll(dir, 0700))
	defer func() {
		assert.NoError(t, os.RemoveAll(dir))
	}()

	// Place some data in the cache dir
	const existing = "xorg-xinit-1.4.1-1-x86_64.pkg.tar.xz"
	const part = "zbar-0.23-1-x86_64.pkg.tar.xz.part"
	assert.NoError(t, os.MkdirAll(filepath.Join(dir, _arch, _repo), 0755))
	for _, f := range []string{part, existing} {
		assert.NoError(t, ioutil.WriteFile(filepath.Join(dir, _arch, _repo, f), []byte("nothing here"), 0644))
	}

	c, err := New(dir, mirrorlist.Mirrorlist{mirrorlist.Mirror(s.URL)})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(c.packets), "Wrong packet number (packets: %v)", c.packets)
	repo := database.Repository{
		Name: _repo,
		Arch: _arch,
	}
	assert.Contains(t, c.packets, repo)
	for _, p := range c.packets[repo] {
		assert.Equal(t, existing, p.Filename())
	}
	_, err = os.Stat(filepath.Join(dir, part))
	assert.True(t, os.IsNotExist(err))

	p, err := packet.FromFilename("pacman", _filename)
	assert.NoError(t, err)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := c.GetPacket(p, &database.Repository{
				Name: _repo,
				Arch: _arch,
			})
			assert.NoError(t, err)
			if err != nil {
				t.FailNow()
			}
			assert.Equal(t, len(_content), getSize(t, r))
			var b bytes.Buffer
			_, err = io.CopyN(&b, r, int64(len(_content)))
			assert.NoError(t, err)
			assert.Equal(t, b.String(), _content)
		}()
		if i == 25 {
			time.Sleep(2 * time.Millisecond)
		}
	}
	wg.Wait()
}
