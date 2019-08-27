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
	directory     string
	mirrors       mirrorlist.Mirrorlist
	packets       packet.Set
	downloads     map[string]*ongoingDownload
	repos         map[database.Repository]struct{}
	repoDownloads map[database.Repository]struct{}
	mu            sync.Mutex
	repoMu        sync.Mutex
	bgDownload    sync.Mutex
}

// ReadSeekCloser implements io.ReadSeeker and io.Closer
type ReadSeekCloser interface {
	io.ReadSeeker
	io.Closer
}

func (c *Cache) initPackets() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	files, err := ioutil.ReadDir(c.directory)
	if err != nil {
		return errors.Wrap(err, "Error opening cache dir")
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if strings.HasSuffix(file.Name(), ".part") {
			os.Remove(path.Join(c.directory, file.Name()))
			continue
		}

		p, err := packet.FromFilename(file.Name())
		if err != nil {
			return errors.Wrapf(err, "Invalid packet in directory")
		}

		c.packets.Insert(p)
	}

	return nil
}

// New creates a new cache from a given directory
func New(directory string, mirrors mirrorlist.Mirrorlist) (*Cache, error) {
	c := &Cache{
		directory:     directory,
		packets:       make(packet.Set),
		mirrors:       mirrors,
		downloads:     make(map[string]*ongoingDownload),
		repos:         make(map[database.Repository]struct{}),
		repoDownloads: make(map[database.Repository]struct{}),
	}

	err := c.initPackets()
	if err != nil {
		return nil, err
	}

	err = c.initRepos()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// GetPacket serves a packet either from the cache or proxies it from a mirror
// Returns an io.ReadSeaker with access to the packet data
func (c *Cache) GetPacket(p *packet.Packet, repo *database.Repository) (ReadSeekCloser, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Download the packet's repo in the backround if we don't have it yet
	go c.addRepo(repo, nil)

	// First: check if the packet is currently being downloaded
	if download, ok := c.downloads[p.Filename()]; ok && download.Dl.P == *p {
		return download.GetReader()
	}

	// Second: check if the packet already is available in cache
	if cachedP := c.packets.ByFilename(p.Filename()); cachedP != nil {
		f, err := os.Open(path.Join(c.directory, cachedP.Filename()))
		if err != nil {
			return nil, errors.Wrap(err, "Error opening cached packet file")
		}

		return f, nil
	}

	// Bail out if newer package version exists
	for _, cachedP := range c.packets.FindOtherVersions(p) {
		versionDiff := packet.CompareVersions(p.Version, cachedP.Version)
		if versionDiff < 0 {
			return nil, errors.New("Newer version available")
		}
	}

	// Third: download packet to cache
	download, err := c.startDownload(&download{*p, *repo, nil})
	if err != nil {
		return nil, errors.Wrap(err, "Error downloading the packet")
	}
	return download.GetReader()
}
