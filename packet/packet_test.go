package packet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilename(t *testing.T) {
	const filename = "xorg-util-macros-1.19.2-1-any.pkg.tar.xz"
	packet, err := FromFilename(filename)
	assert.NoError(t, err, "Error while parsing filename: %v", err)
	assert.Equal(t, filename, packet.Filename())
}
