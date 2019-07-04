package packet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testCompareVersions(t *testing.T, s []string) {
	// compare all permutations of test list
	for i := 0; i < len(s); i++ {
		for j := 0; j < len(s); j++ {
			r, err := CompareVersions(s[i], s[j])
			//r := vc.CompareSimple(s[i], s[j])
			assert.NoError(t, err)
			if i < j {
				assert.Equal(t, -1, r, "%s <! %s", s[i], s[j])
			} else if i > j {
				assert.Equal(t, 1, r, "%s >! %s", s[i], s[j])
			} else {
				assert.Equal(t, 0, r, "%s =! %s", s[i], s[j])
			}
		}
	}
}

func TestCompareVersions(t *testing.T) {
	testCompareVersions(t,
		[]string{"1.0a", "1.0b", "1.0beta", "1.0p", "1.0pre",
			"1.0rc", "1.0", "1.0.a", "1.0.1"},
	)
	testCompareVersions(t,
		[]string{"1", "1.0", "1.1", "1.1.1", "1.2", "2.0", "3.0.0"},
	)
	testCompareVersions(t,
		[]string{"17.3.4a", "1:0.0.1", "1:2.0", "2:1"},
	)
}
