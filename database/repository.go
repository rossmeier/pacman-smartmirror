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

// ParseDBFromFile reads a pacman .db file and creats a []packet.Packet directly from File
func ParseDBFromFile(filename string) ([]packet.Packet, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading file")
	}
	defer file.Close()

	return ParseDB(file)
}

// ParseDB reads a pacman .db file and creats a []packet.Packet
func ParseDB(r io.Reader) ([]packet.Packet, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "Error gunzipping file")
	}
	defer zr.Close()

	return ParseDBgunzipped(zr)
}

// ParseDBCBFromFile provides a callback for ParseDBFromFile
func ParseDBCBFromFile(filename string, callback func([]packet.Packet)) {
	packets, err := ParseDBFromFile(filename)
	if err != nil {
		callback(packets)
	} else {
		callback(nil)
	}
}

// ParseDBCB provides a callback for ParseDB
func ParseDBCB(r io.Reader, callback func([]packet.Packet)) {
	packets, err := ParseDB(r)
	if err != nil {
		callback(packets)
	} else {
		callback(nil)
	}
}

// ParseDBgunzipped reads a pacman .db file and creats a []DatabaseEntry
func ParseDBgunzipped(r io.Reader) ([]packet.Packet, error) {

	readDb := make([]packet.Packet, 0)

	buf := &bytes.Buffer{}
	reader := tar.NewReader(r)
	for {
		pkg, err := reader.Next()
		if err == io.EOF {
			return readDb, nil
		}
		if err != nil {
			return nil, errors.New("Error while reading tar (DbScratch)")
		}
		if pkg.FileInfo().IsDir() {
			continue
		}
		io.Copy(buf, reader)
		str, err := buf.ReadString('\n')
		if err != nil {
			panic(err)
		}
		if str != "%FILENAME%\n" {
			panic("Invalid filename designator: " + str)
		}
		filename, err := buf.ReadString('\n')
		if err != nil {
			panic(err)
		}
		p, err := packet.FromFilename(filename)
		if err != nil {
			panic(err)
		}

		readDb = append(readDb, *p)

		buf.Reset()
	}

	return nil, errors.New("unreachable code executed @ ParseDB")
}
