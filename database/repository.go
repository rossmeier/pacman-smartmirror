package database

import (
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/veecue/pacman-smartmirror/packet"
)

// Repository describes a Repo as found on an upstream server
type Repository struct {
	Name string
	Arch string
}

func (r Repository) String() string {
	return r.Arch + "/" + r.Name
}

type ParseDBFunc func(io.Reader, PacketCallback) error

// PacketCallback is a callback for packets that will receive the packet parsed from
// the filename and a reader containing the rest of the packages "desc" file with
// further information
type PacketCallback func(packet.Packet, io.Reader)

// ParseDBFromFile reads a pacman .db file and call cb for each packet directly from File
func ParseDBFromFile(impl string, filename string, cb PacketCallback) error {
	file, err := os.Open(filename)
	if err != nil {
		return errors.Wrap(err, "Error reading file")
	}
	defer file.Close()

	return ParseDB(impl, file, cb)
}

var parseDBfuncs = make(map[string]ParseDBFunc)

func ParseDB(impl string, reader io.Reader, cb PacketCallback) error {
	fn, ok := parseDBfuncs[impl]
	if !ok {
		return fmt.Errorf("unknown repo implementation: %s", impl)
	}
	return fn(reader, cb)
}

func RegisterImpl(name string, fn ParseDBFunc) {
	if _, ok := parseDBfuncs[name]; ok {
		panic("Given impl already registered")
	}
	parseDBfuncs[name] = fn
}
