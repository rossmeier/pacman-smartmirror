package apk

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/veecue/pacman-smartmirror/impl"
)

// ParseDB reads an apk APKINDEX.tar.gz file and will call cb for each packet
func (i *apkImpl) ParseDB(r io.Reader, cb impl.PacketCallback) error {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("error gunzipping file: %w", err)
	}
	defer zr.Close()

	return i.parseDBGUnzipped(zr, cb)
}

func parsePkgInfo(r *bufio.Reader) (*pkg, error) {
	name := ""
	version := ""
	for {
		str, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		str = str[:len(str)-1]
		if strings.HasPrefix(str, "P:") {
			name = str[2:]
		}
		if strings.HasPrefix(str, "V:") {
			version = str[2:]
		}
		if len(str) == 0 {
			if name == "" || version == "" {
				return nil, errors.New("missing name or version from APKINDEX")
			}
			return &pkg{
				name:    name,
				version: version,
			}, nil
		}
	}
}

func (i *apkImpl) parseDBGUnzipped(r io.Reader, cb impl.PacketCallback) error {
	found := false
	eof := false
	reader := tar.NewReader(r)
	for !eof {
		file, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("error while reading tar: %w", err)
		}
		if file.FileInfo().IsDir() {
			continue
		}
		if file.Name != "APKINDEX" {
			found = true
			continue
		}
		buf := bufio.NewReader(reader)
		for {
			p, err := parsePkgInfo(buf)
			if errors.Is(err, io.EOF) {
				eof = true
				break
			}
			if err != nil {
				return err
			}
			cb(p)
		}
	}

	if !found {
		return errors.New("missing APKINDEX in APKINDEX.tar.gz")
	}

	return nil
}

func (*apkImpl) GetDB(repopath string) string {
	return path.Join(repopath, "APKINDEX.tar.gz")
}
