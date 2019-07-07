package cache

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/veecue/pacman-smartmirror/database"
	"github.com/veecue/pacman-smartmirror/mirrorlist"
	"github.com/veecue/pacman-smartmirror/packet"
)

// Cache is a cache that caches packages in the filesystem.
// The currently only implementation at the moment is storing
// it in a directory in the filesystem.
type Cache struct {
	directory string
	mirrors   mirrorlist.Mirrorlist
	packets   map[*packet.Packet]struct{}
	downloads map[*ongoingDownload]struct{}
	mu        sync.Mutex
}

func (c *Cache) init() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	files, err := ioutil.ReadDir(c.directory)
	if err != nil {
		return errors.Wrap(err, "Error opening cache dir")
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".part") {
			os.Remove(path.Join(c.directory, file.Name()))
		}

		p, err := packet.FromFilename(file.Name())
		if err != nil {
			return errors.Wrapf(err, "Invalid packet in directory")
		}

		c.packets[p] = struct{}{}
	}

	return nil
}

// New creates a new cache from a given directory
func New(directory string, mirrors mirrorlist.Mirrorlist) (*Cache, error) {
	c := &Cache{
		directory: directory,
		packets:   make(map[*packet.Packet]struct{}),
		mirrors:   mirrors,
		downloads: make(map[*ongoingDownload]struct{}),
	}

	err := c.init()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// GetPacket serves a packet either from the cache or proxies it from a mirror
// Returns an io.ReadSeaker with access to the packet data
// If the returned io.ReadSeaker also is an io.Closer, it should be Closed after use.
func (c *Cache) GetPacket(p *packet.Packet, repo *database.Repository) (io.ReadSeeker, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// first: check if the packet is currently being downloaded
	for download := range c.downloads {
		if download.P == *p {
			return download.GetReader()
		}
	}

	// second: check if the packet already is available in cache
	for cachedP := range c.packets {
		if *cachedP == *p {
			f, err := os.Open(path.Join(c.directory, cachedP.Filename()))
			if err != nil {
				return nil, errors.Wrap(err, "Error opening cached packet file")
			}
			return f, nil
		}
	}

	// third: download packet to cache
	download, err := c.startDownload(p, repo)
	if err != nil {
		return nil, errors.Wrap(err, "Error downloading the packet")
	}
	return download.GetReader()
}
