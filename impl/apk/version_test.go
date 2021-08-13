package apk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionCompare(t *testing.T) {
	x := &apkImpl{}
	assert.Less(t, x.CompareVersions("1.0", "1.1"), 0)
	assert.Less(t, x.CompareVersions("3.3.3-r2", "3.3.3p1-r2"), 0)
}
