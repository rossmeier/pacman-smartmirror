package packet

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	filenameRegex = regexp.MustCompile(`(.+)-(.+-.+)-(.+)\.pkg\.tar\.xz`)
)

// Packet represents a pacman Packet
type Packet struct {
	Name    string
	Version string
	Arch    string
}

// Filename returns the corresponding filename the packet is saved as
func (p *Packet) Filename() string {
	return fmt.Sprintf("%s-%s-%s.pkg.tar.xz",
		p.Name,
		p.Version,
		p.Arch,
	)
}

// FromFilename parses a packet's filename and returns the parsed information
func FromFilename(filename string) (*Packet, error) {
	matches := filenameRegex.FindStringSubmatch(filename)
	if len(matches) != 4 {
		return nil, errors.New("Invalid filename")
	}

	return &Packet{
		Name:    matches[1],
		Version: matches[2],
		Arch:    matches[3],
	}, nil
}
