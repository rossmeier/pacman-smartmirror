package impl

import (
	"errors"
	"fmt"
	"io"

	"github.com/veecue/pacman-smartmirror/packet"
)

// PacketCallback is a callback for packets that will receive the packet parsed from
// the filename
type PacketCallback func(packet.Packet)

var (
	// ErrInvalidFilename indicates that a packet's filename could not be parsed by the
	// specific implementation
	ErrInvalidFilename = errors.New("invalid filename")
)

// Factory is a function that creates an Implf from the given string arguments.
// Should panic in case of error.
type Factory func(args map[string]string) Impl

var impls = make(map[string]Factory)

// Impl is a specific implementation for interfacing the data structures of a unix package manager
type Impl interface {
	GetDB(repopath string) string
	PacketFromFilename(name string) (packet.Packet, error)
	ParseDB(reader io.Reader, cb PacketCallback) error
	CompareVersions(v1, v2 string) int
}

// Get returns the requested registered impl by its name and string arguments
func Get(impl string, args map[string]string) Impl {
	fn, ok := impls[impl]
	if !ok {
		panic(fmt.Errorf("unknown implementation: %s", impl))
	}
	return fn(args)
}

// Register is used to register an impl within this package so it can be found by Get
func Register(name string, fn Factory) {
	if _, ok := impls[name]; ok {
		panic("Given impl already registered")
	}
	impls[name] = fn
}
