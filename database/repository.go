package database

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

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

// PacketCallback is a callback for packets that will receive the packet parsed from
// the filename and a reader containing the rest of the packages "desc" file with
// further information
type PacketCallback func(packet.Packet, io.Reader)

// ParseDBFromFile reads a pacman .db file and call cb for each packet directly from File
func ParseDBFromFile(filename string, cb PacketCallback) error {
	file, err := os.Open(filename)
	if err != nil {
		return errors.Wrap(err, "Error reading file")
	}
	defer file.Close()

	return ParseDB(file, cb)
}

// ParseDBFromFileSlice reads a pacman .db file and creates a []packet.Packet directly from File
func ParseDBFromFileSlice(filename string) ([]packet.Packet, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading file")
	}
	defer file.Close()

	return ParseDBSlice(file)
}

// ParseDB reads a pacman .db file and will call cb for each packet
func ParseDB(r io.Reader, cb PacketCallback) error {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return errors.Wrap(err, "Error gunzipping file")
	}
	defer zr.Close()

	return ParseDBGUnzipped(zr, cb)
}

// ParseDBSlice reads a pacman .db file and creates a []packet.Packet
func ParseDBSlice(r io.Reader) ([]packet.Packet, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "Error gunzipping file")
	}
	defer zr.Close()

	return ParseDBGUnzippedSlice(zr)
}

// ParseDBGUnzippedSlice reads a pacman .db file and creates a []packet.Packet
func ParseDBGUnzippedSlice(r io.Reader) ([]packet.Packet, error) {
	readDb := make([]packet.Packet, 0)

	err := ParseDBGUnzipped(r, func(p packet.Packet, _ io.Reader) {
		readDb = append(readDb, p)
	})
	if err != nil {
		return nil, err
	}

	return readDb, nil
}

// ParseDBGUnzipped reads a pacman .db file and will call cb for each packet
func ParseDBGUnzipped(r io.Reader, cb PacketCallback) error {

	buf := &bytes.Buffer{}
	reader := tar.NewReader(r)
	for {
		pkg, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.Wrap(err, "Error while reading tar")
		}
		if pkg.FileInfo().IsDir() {
			continue
		}
		if _, name := filepath.Split(pkg.Name); name != "desc" {
			continue
		}
		io.Copy(buf, reader)
		str, err := buf.ReadString('\n')
		if err != nil {
			return (err)
		}
		if str != "%FILENAME%\n" {
			return errors.New("Invalid filename designator: " + str)
		}
		filename, err := buf.ReadString('\n')
		if err != nil {
			return (err)
		}
		p, err := packet.FromFilename("pacman", filename)
		if err != nil {
			return (err)
		}

		cb(p, buf)

		buf.Reset()
	}
}
