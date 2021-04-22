package mirrorlist

import (
	"bufio"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/veecue/pacman-smartmirror/database"
	"github.com/veecue/pacman-smartmirror/packet"
)

// Mirrorlist represents a list of Arch Linux mirror server URLs
type Mirrorlist []Mirror

// Mirror represents the URL of an Arch Linux mirror server
type Mirror string

// FromFile reads the given mirrorlist file and returns the content URLs
func FromFile(filename string) (Mirrorlist, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading file")
	}
	defer file.Close()

	return FromReader(file)
}

// FromReader reads a mirrorlist from the given reader and returns the URLs
func FromReader(r io.Reader) (Mirrorlist, error) {
	m := make([]Mirror, 0)

	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, errors.Wrap(err, "Error reading mirrorlist!")
		}

		if strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		if strings.TrimSpace(parts[0]) != "Server" {
			continue
		}

		uri := strings.TrimSpace(parts[1])
		_, err = url.Parse(uri)
		if err != nil {
			return nil, errors.Wrapf(err, `"%s" is not a valid url`, uri)
		}

		m = append(m, Mirror(strings.TrimSpace(parts[1])))
	}

	return m, nil
}

// PacketURL returns the actual URL of a given packet
func (m Mirror) PacketURL(p packet.Packet, repo *database.Repository) string {
	r := strings.ReplaceAll(string(m), "$repo", repo.Name)
	r = strings.ReplaceAll(r, "$arch", repo.Arch)
	r = strings.TrimSuffix(r, "/")
	return r + "/" + p.Filename()
}

// RepoURL returns the actual URL of a given repo db
func (m Mirror) RepoURL(repo *database.Repository) string {
	r := strings.ReplaceAll(string(m), "$repo", repo.Name)
	r = strings.ReplaceAll(r, "$arch", repo.Arch)
	r = strings.TrimSuffix(r, "/")
	return r + "/" + repo.Name + ".db"
}
