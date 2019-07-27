package packet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testCompareVersions(t *testing.T, s []string) {
	// compare all permutations of test list
	for i := 0; i < len(s); i++ {
		for j := 0; j < len(s); j++ {
			r := CompareVersions(s[i], s[j])
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
	// Ascending list from pacman doc (alphanumeric)
	testCompareVersions(t,
		[]string{"1.0a", "1.0b", "1.0beta", "1.0p", "1.0pre",
			"1.0rc", "1.0", "1.0.a", "1.0.1"},
	)

	// Ascending list from pacman doc (numeric)
	testCompareVersions(t,
		[]string{"1", "1.0", "1.1", "1.1.1", "1.2", "2.0", "3.0.0"},
	)

	// With epoch
	testCompareVersions(t,
		[]string{"17.3.4a", "1:0.0.1", "1:2.0", "2:1"},
	)

	// With rel
	testCompareVersions(t,
		[]string{"1.0-1", "1.0-2", "1.0-03", "1.0-17"},
	)
}
