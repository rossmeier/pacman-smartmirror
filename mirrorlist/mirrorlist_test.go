package mirrorlist

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/veecue/pacman-smartmirror/database"
	"github.com/veecue/pacman-smartmirror/packet"
)

const mirrorlistGood = `## Germany
Server = http://mirror.pseudoform.org/$repo/os/$arch
## France
Server = http://mirrors.arnoldthebat.co.uk/archlinux/$repo/os/$arch
Internet = no_internet
#
asdf
`

const mirrorlistBad = `Server = asdf\-:
`

func TestMirrorlistGood(t *testing.T) {
	m, err := FromReader(strings.NewReader(mirrorlistGood))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(m))
	assert.Equal(t, "http://mirror.pseudoform.org/$repo/os/$arch", string(m[0]))
	assert.Equal(t, "http://mirrors.arnoldthebat.co.uk/archlinux/$repo/os/$arch", string(m[1]))

	p, err := packet.FromFilename("youtube-dl-2019.07.02-1-any.pkg.tar.xz")
	assert.NoError(t, err)
	assert.Equal(t,
		"http://mirrors.arnoldthebat.co.uk/archlinux/community/os/x86_64/"+p.Filename(),
		string(m[1].PacketURL(p, &database.Repository{
			Name: "community",
			Arch: "x86_64",
		})),
	)
	assert.Equal(t,
		"http://mirrors.arnoldthebat.co.uk/archlinux/community/os/x86_64/community.db",
		m[1].RepoURL(&database.Repository{
			Name: "community",
			Arch: "x86_64",
		}),
	)
}

type eReader struct{}

func (eReader) Read(b []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestMirrorlistBad(t *testing.T) {
	m, err := FromReader(strings.NewReader(mirrorlistBad))
	assert.Error(t, err)
	assert.Nil(t, m)

	_, err = FromReader(eReader{})
	assert.Error(t, err)
}

func TestMirrorlistFile(t *testing.T) {
	_, err := FromFile("/path/to/non/existant/file")
	assert.Error(t, err)
	f, err := ioutil.TempFile("", "pacman-smartmirror-mirrorlist")
	defer func() {
		f.Close()
		assert.NoError(t, os.Remove(f.Name()))
	}()
	assert.NoError(t, err)
	_, err = io.WriteString(f, mirrorlistGood)
	assert.NoError(t, err)
	m, err := FromFile(f.Name())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(m))
}
