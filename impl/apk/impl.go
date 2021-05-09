package apk

import (
	"github.com/veecue/pacman-smartmirror/impl"
)

type apkImpl struct{}

func newAPKImpl(args map[string]string) impl.Impl {
	return &apkImpl{}
}

func init() {
	impl.Register("apk", newAPKImpl)
}
