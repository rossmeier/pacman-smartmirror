package packet

import (
	"strconv"
	"strings"
	"unicode"
)

type version struct {
	epoch int
	v     string
}

func getVersion(s string) (v version, err error) {
	sp := strings.SplitN(s, ":", 2)
	if len(sp) == 2 {
		v.epoch, err = strconv.Atoi(sp[0])
		if err != nil {
			return
		}
		sp[0] = sp[1]
	} else {
		v.epoch = 0
	}
	v.v = sp[0]
	return
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func cmp(a, b int) int {
	if a < b {
		return -1
	}

	if a > b {
		return 1
	}

	return 0
}

// ported from libalpm/version.c
func rpmvercmp(a, b string) (ret int) {
	if a == b {
		return 0
	}
	var startI, startJ int
	var i, j int
	for i < len(a) && j < len(b) {
		for i < len(a) && !unicode.IsDigit(rune(a[i])) && !unicode.IsLetter(rune(a[i])) {
			i++
		}

		for j < len(b) && !unicode.IsDigit(rune(b[j])) && !unicode.IsLetter(rune(b[j])) {
			j++
		}

		if i == len(a) || j == len(b) {
			break
		}

		ret = cmp(i-startI, j-startJ)
		if ret != 0 {
			return
		}

		startI = i
		startJ = j

		var isnum bool
		if unicode.IsDigit(rune(a[startI])) {
			for startI < len(a) && unicode.IsDigit(rune(a[startI])) {
				startI++
			}
			for startJ < len(b) && unicode.IsDigit(rune(b[startJ])) {
				startJ++
			}
			isnum = true
		} else {
			for startI < len(a) && unicode.IsLetter(rune(a[startI])) {
				startI++
			}
			for startJ < len(b) && unicode.IsLetter(rune(b[startJ])) {
				startJ++
			}
			isnum = false
		}

		if i == startI {
			panic("Aparrently we failed somewhere")
		}

		if j == startJ {
			if isnum {
				return 1
			}
			return -1
		}

		if isnum {
			for i < len(a) && a[i] == '0' {
				i++
			}
			for j < len(b) && b[j] == '0' {
				j++
			}
			ret = cmp(startI-i, startJ-j)
			if ret != 0 {
				return ret
			}
		}

		ret = strings.Compare(a[i:startI], b[j:startJ])
		if ret != 0 {
			return ret
		}

		i = startI
		j = startJ
	}

	if i == len(a) && j == len(b) {
		return 0
	}

	if (i == len(a) && !unicode.IsLetter(rune(b[j])) && !unicode.IsDigit(rune(b[j]))) ||
		i < len(a) && (unicode.IsLetter(rune(a[i])) || unicode.IsDigit(rune(a[i]))) {
		return -1
	}
	return 1
}

// CompareVersions compares the two given packet versions
func CompareVersions(v1, v2 string) (r int, err error) {
	a, err := getVersion(v1)
	if err != nil {
		return
	}

	b, err := getVersion(v2)
	if err != nil {
		return
	}

	r = cmp(a.epoch, b.epoch)
	if r != 0 {
		return r, nil
	}

	return rpmvercmp(a.v, b.v), nil
}
