package pacman

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/veecue/pacman-smartmirror/database"
	"github.com/veecue/pacman-smartmirror/packet"
)

// ParseDB reads a pacman .db file and will call cb for each packet
func ParseDB(r io.Reader, cb database.PacketCallback) error {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return errors.Wrap(err, "Error gunzipping file")
	}
	defer zr.Close()

	return parseDBGUnzipped(zr, cb)
}

// parseDBGUnzipped reads a pacman .db file and will call cb for each packet
func parseDBGUnzipped(r io.Reader, cb database.PacketCallback) error {

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

func init() {
	database.RegisterImpl("pacman", ParseDB)
}
