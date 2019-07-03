package mirrorlist

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
		"http://mirrors.arnoldthebat.co.uk/archlinux/community/os/any/"+p.Filename(),
		string(m[1].PacketURL(p, "community")),
	)
}

func TestMirrorlistBad(t *testing.T) {
	m, err := FromReader(strings.NewReader(mirrorlistBad))
	assert.Error(t, err)
	assert.Nil(t, m)
}
