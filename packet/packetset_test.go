package packet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {
	packets := []*Packet{
		{
			Name:    "a",
			Version: "0.1",
			Arch:    "any",
		}, {
			Name:    "a",
			Version: "0.2",
			Arch:    "any",
		}, {
			Name:    "aa",
			Version: "0.1",
			Arch:    "any",
		}, {
			Name:    "a",
			Version: "0.2",
			Arch:    "x86_64",
		},
	}
	s := make(Set)
	for _, p := range packets {
		s.Insert(p)
	}
	assert.Equal(t, len(packets), len(s))
	s.Insert(packets[0])
	assert.Equal(t, len(packets), len(s))
	assert.Equal(t, *packets[0], *s.ByFilename(packets[0].Filename()))
	assert.Nil(t, s.ByFilename("nonexistant"))
	assert.Equal(t, 3, len(s.ByName("a")))
	assert.Equal(t, 2, len(s.FindOtherVersions(packets[0])))
	s.Delete(packets[0].Filename())
	assert.Equal(t, len(packets)-1, len(s))
}
