package packet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilename(t *testing.T) {
	for _, filename := range []string{
		"xorg-util-macros-1.19.2-1-any.pkg.tar.xz",
		"xorg-util-macros-1.21.2-1-any.pkg.tar.zst",
	} {
		packet, err := FromFilename(filename)
		assert.NoError(t, err, "Error while parsing filename: %v", err)
		assert.Equal(t, filename, packet.Filename())
	}
}

func TestInvalidFilename(t *testing.T) {
	for _, filename := range []string{
		"linux.pkg.tar.xz",
		"xorg-util-macros-1.21.2-1-any.pkg.tar.foo",
	} {
		_, err := FromFilename(filename)
		assert.Error(t, err)
	}
}
