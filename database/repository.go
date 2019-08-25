package database

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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

// ParseDBFromFile reads a pacman .db file and call cb for each packet directly from File
func ParseDBFromFile(filename string, cb func(*packet.Packet)) error {
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
func ParseDB(r io.Reader, cb func(*packet.Packet)) error {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return errors.Wrap(err, "Error gunzipping file")
	}
	defer zr.Close()

	return ParseDBgunzipped(r, cb)
}

// ParseDBSlice reads a pacman .db file and creates a []packet.Packet
func ParseDBSlice(r io.Reader) ([]packet.Packet, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "Error gunzipping file")
	}
	defer zr.Close()

	return ParseDBgunzippedSlice(zr)
}

// ParseDBgunzippedSlice reads a pacman .db file and creates a []packet.Packet
func ParseDBgunzippedSlice(r io.Reader) ([]packet.Packet, error) {

	readDb := make([]packet.Packet, 0)

	ParseDBgunzipped(r, func(p *packet.Packet) {
		readDb = append(readDb, *p)
	})

	return readDb, nil
}

// ParseDBgunzipped reads a pacman .db file and will call cb for each packet
func ParseDBgunzipped(r io.Reader, cb func(*packet.Packet)) error {

	buf := &bytes.Buffer{}
	reader := tar.NewReader(r)
	for {
		pkg, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.New("Error while reading tar (DbScratch)")
		}
		if pkg.FileInfo().IsDir() {
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
		p, err := packet.FromFilename(filename)
		if err != nil {
			return (err)
		}

		cb(p)

		buf.Reset()
	}
}
