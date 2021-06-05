package pacman

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/veecue/pacman-smartmirror/packet"
)

var i = newPacmanImpl(map[string]string{"reponame": "core"})

func TestFilename(t *testing.T) {
	for _, filename := range []string{
		"xorg-util-macros-1.19.2-1-any.pkg.tar.xz",
		"xorg-util-macros-1.21.2-1-any.pkg.tar.zst",
	} {
		packet, err := i.PacketFromFilename(filename)
		assert.NoError(t, err, "Error while parsing filename: %v", err)
		assert.Equal(t, filename, packet.Filename())
	}
}

func TestInvalidFilename(t *testing.T) {
	for _, filename := range []string{
		"linux.pkg.tar.xz",
		"xorg-util-macros-1.21.2-1-any.pkg.tar.foo",
		"xorg-util-macros-1.21.2-1-any.pkg.tar.zst.sig",
	} {
		_, err := i.PacketFromFilename(filename)
		assert.Error(t, err)
	}
}

func TestSet(t *testing.T) {
	packets := []*pkg{
		{
			name:    "a",
			version: "0.1",
			arch:    "any",
		}, {
			name:    "a",
			version: "0.2",
			arch:    "any",
		}, {
			name:    "aa",
			version: "0.1",
			arch:    "any",
		},
	}
	s := make(packet.Set)
	for _, p := range packets {
		s.Insert(p)
	}
	assert.Equal(t, len(packets), len(s))
	s.Insert(packets[0])
	assert.Equal(t, len(packets), len(s))
	assert.Equal(t, *packets[0], *(s.ByFilename(packets[0].Filename()).(*pkg)))
	assert.Nil(t, s.ByFilename("nonexistant"))
	assert.Equal(t, 2, len(s.ByName("a")))
	assert.Equal(t, 2, len(s.FindOtherVersions(packets[0])))
	s.Delete(packets[0].Filename())
	assert.Equal(t, len(packets)-1, len(s))
}
