package database

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/pkg/errors"
)

// Repository describes a Repo as found on an upstream server
type Repository struct {
	Name string
	Arch string
}

type DatabaseEntry struct {
	Name        string
	Base        string
	Version     string
	Desc        string
	CSize       string
	ISize       string
	MD5Sum      string
	PGPSig      string
	URL         string
	License     string
	Arch        string
	BuildDate   string
	Packager    string
	Replaces    string
	Conflicts   string
	Provides    string
	Dependes    string
	MakeDepends string
}

func DbScratch() error {
	file, err := os.Open("core.db")
	if err != nil {
		return errors.Wrap(err, "Error reading file")
	}

	zr, err := gzip.NewReader(file)
	defer zr.Close()

	fmt.Printf("Name: %s\nComment: %s\nModTime: %s\n\n", zr.Name, zr.Comment, zr.ModTime.UTC())

	reader := bufio.NewReader(zr)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "Error reading mirrorlist!")
		}

		if line == "%URL%\n" {
			fmt.Println(line)
		}

	}

	if err := zr.Close(); err != nil {
		log.Fatal(err)
	}

	return nil
}
