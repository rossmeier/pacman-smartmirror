package pacman

import (
	"errors"

	"github.com/veecue/pacman-smartmirror/impl"
)

type pacmanImpl struct {
	reponame string
}

func newPacmanImpl(args map[string]string) impl.Impl {
	reponame, ok := args["reponame"]
	if !ok {
		panic(errors.New("missing reponame"))
	}
	return &pacmanImpl{
		reponame: reponame,
	}
}

func init() {
	impl.Register("pacman", newPacmanImpl)
}
