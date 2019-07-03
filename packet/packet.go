package packet

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

var (
	filenameRegex = regexp.MustCompile(`(.+)-(.+)-(.+)-(.+)\.pkg\.tar\.xz`)
)

// Packet represents a pacman Packet
type Packet struct {
	Name    string
	Version string
	Rel     int
	Arch    string
}

// Filename returns the corresponding filename the packet is saved as
func (p *Packet) Filename() string {
	return fmt.Sprintf("%s-%s-%d-%s.pkg.tar.xz",
		p.Name,
		p.Version,
		p.Rel,
		p.Arch,
	)
}

// FromFilename parses a packet's filename and returns the parsed information
func FromFilename(filename string) (*Packet, error) {
	matches := filenameRegex.FindStringSubmatch(filename)
	if len(matches) != 5 {
		return nil, errors.New("Invalid filename")
	}

	rel, err := strconv.Atoi(matches[3])
	if err != nil {
		return nil, errors.New("Packet rel not an integer")
	}

	return &Packet{
		Name:    matches[1],
		Version: matches[2],
		Rel:     rel,
		Arch:    matches[4],
	}, nil
}
