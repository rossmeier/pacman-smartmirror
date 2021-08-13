package pacman

import (
	"strings"

	"github.com/veecue/pacman-smartmirror/packet"
)

type version struct {
	epoch string
	v     string
	rel   string
}

func getVersion(s string) (v version) {
	sp := strings.SplitN(s, ":", 2)
	if len(sp) == 2 {
		v.epoch = sp[0]
		sp[0] = sp[1]
	} else {
		v.epoch = "0"
	}
	sp = strings.SplitN(sp[0], "-", 2)
	if len(sp) >= 2 {
		v.v = sp[0]
		v.rel = sp[1]
	} else {
		v.v = sp[0]
	}
	return
}

func (*pacmanImpl) CompareVersions(v1, v2 string) (r int) {
	a := getVersion(v1)
	b := getVersion(v2)

	r = packet.RPMVerCmp(a.epoch, b.epoch)
	if r != 0 {
		return
	}

	r = packet.RPMVerCmp(a.v, b.v)
	if r != 0 {
		return
	}

	return packet.RPMVerCmp(a.rel, b.rel)
}
