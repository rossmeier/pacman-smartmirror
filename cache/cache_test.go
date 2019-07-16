package cache

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"path"
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

	dir := path.Join(os.TempDir(), "smartmirror-test")
	assert.NoError(t, os.RemoveAll(dir))
	assert.NoError(t, os.MkdirAll(dir, 0700))
	defer func() {
		assert.NoError(t, os.RemoveAll(dir))
	}()

	c, err := New(dir, mirrorlist.Mirrorlist{mirrorlist.Mirror(s.URL)})
	assert.NoError(t, err)
	p, err := packet.FromFilename(_filename)
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
			log.Println(b.String())
		}()
		if i == 25 {
			time.Sleep(2 * time.Millisecond)
		}
	}
	wg.Wait()
}
