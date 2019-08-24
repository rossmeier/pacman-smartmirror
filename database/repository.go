package database

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// Repository describes a Repo as found on an upstream server
type Repository struct {
	Name string
	Arch string
}

type DatabaseEntry struct {
	Filename    string
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
	Depends     string
	MakeDepends string
}

const (
	none = iota
	filename
	name
	base
	version
	desc
	cSize
	iSize
	mD5Sum
	pGPSig
	uRL
	license
	arch
	buildDate
	packager
	replaces
	conflicts
	provides
	depends
	makeDepends
)

// DbScratchFromFile reads a pacman .db file and creats a []DatabaseEntry directly from File
func DbScratchFromFile(filename string) ([]DatabaseEntry, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading file")
	}
	defer file.Close()

	return DbScratchFromReader(file)
}

func DbScratchFromReader(r io.Reader) ([]DatabaseEntry, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "Error gunzipping file")
	}
	defer zr.Close()

	return DbScratchFromGUnzippedReader(zr)
}

// DbScratchFromGZipReader reads a pacman .db file and creats a []DatabaseEntry
func DbScratchFromGUnzippedReader(r io.Reader) ([]DatabaseEntry, error) {

	readDb := make([]DatabaseEntry, 0)

	reader := bufio.NewReader(r)
	categoryRegex := regexp.MustCompile(`(?m)%[0-9A-Z]*%\n`)
	var current DatabaseEntry
	state := none
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, errors.Wrap(err, "Error reading mirrorlist!")
		}

		switch state {
		case none:
			{
				line = strings.TrimPrefix(strings.TrimSuffix(categoryRegex.FindString(line), "%\n"), "%")

				if len(line) == 0 {
					continue
				}

				switch line {
				case "FILENAME":
					state = filename
				case "NAME":
					state = name
				case "BASE":
					state = base
				case "VERSION":
					state = version
				case "DESC":
					state = desc
				case "CSIZE":
					state = cSize
				case "ISIZE":
					state = iSize
				case "MD5SUM":
					state = mD5Sum
				case "PGPSIG":
					state = pGPSig
				case "URL":
					state = uRL
				case "LICENSE":
					state = license
				case "ARCH":
					state = arch
				case "BUILDDATE":
					state = buildDate
				case "PACKAGER":
					state = packager
				case "REPLACES":
					state = replaces
				case "CONFLICTS":
					state = conflicts
				case "PROVIDES":
					state = provides
				case "DEPENDS":
					state = depends
				case "MAKEDEPENDS":
					state = makeDepends
				}
			}
		case filename:
			// append old, initalize new entry
			if current.Filename != "" {
				readDb = append(readDb, current)
			}
			current = DatabaseEntry{}

			current.Filename = strings.TrimSuffix(line, "\n")
			state = none
		case name:
			current.Name = strings.TrimSuffix(line, "\n")
			state = none
		case base:
			current.Base = strings.TrimSuffix(line, "\n")
			state = none
		case version:
			current.Version = strings.TrimSuffix(line, "\n")
			state = none
		case desc:
			current.Desc = strings.TrimSuffix(line, "\n")
			state = none
		case cSize:
			current.CSize = strings.TrimSuffix(line, "\n")
			state = none
		case iSize:
			current.ISize = strings.TrimSuffix(line, "\n")
			state = none
		case mD5Sum:
			current.MD5Sum = strings.TrimSuffix(line, "\n")
			state = none
		case pGPSig:
			current.PGPSig = strings.TrimSuffix(line, "\n")
			state = none
		case uRL:
			current.URL = strings.TrimSuffix(line, "\n")
			state = none
		case license:
			current.License = strings.TrimSuffix(line, "\n")
			state = none
		case arch:
			current.Arch = strings.TrimSuffix(line, "\n")
			state = none
		case buildDate:
			current.BuildDate = strings.TrimSuffix(line, "\n")
			state = none
		case packager:
			current.Packager = strings.TrimSuffix(line, "\n")
			state = none
		case replaces:
			current.Replaces = strings.TrimSuffix(line, "\n")
			state = none
		case conflicts:
			current.Conflicts = strings.TrimSuffix(line, "\n")
			state = none
		case provides:
			current.Provides = strings.TrimSuffix(line, "\n")
			state = none
		case depends:
			current.Depends = strings.TrimSuffix(line, "\n")
			state = none
		case makeDepends:
			current.MakeDepends = strings.TrimSuffix(line, "\n")
			state = none
		}
	}

	if current.Filename != "" {
		readDb = append(readDb, current)
	}

	return readDb, nil
}
