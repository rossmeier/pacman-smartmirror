package pacman

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"

	"github.com/veecue/pacman-smartmirror/impl"
)

// ParseDB reads a pacman .db file and will call cb for each packet
func (i *pacmanImpl) ParseDB(r io.Reader, cb impl.PacketCallback) error {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("error gunzipping file: %w", err)
	}
	defer zr.Close()

	return i.parseDBGUnzipped(zr, cb)
}

// parseDBGUnzipped reads a pacman .db file and will call cb for each packet
func (i *pacmanImpl) parseDBGUnzipped(r io.Reader, cb impl.PacketCallback) error {

	buf := &bytes.Buffer{}
	reader := tar.NewReader(r)
	for {
		pkg, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("error while reading tar: %w", err)
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
		filename = filename[:len(filename)-1]
		p, err := i.PacketFromFilename(filename)
		if err != nil {
			return (err)
		}

		cb(p)

		buf.Reset()
	}
}

func (i *pacmanImpl) GetDB(repopath string) string {
	return path.Join(repopath, i.reponame+".db")
}
