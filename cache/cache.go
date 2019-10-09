package cache

import (
	"io"
	"os"
	"path/filepath"
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
	packets       map[database.Repository]packet.Set
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

// New creates a new cache from a given directory
func New(directory string, mirrors mirrorlist.Mirrorlist) (*Cache, error) {
	c := &Cache{
		directory:     directory,
		packets:       make(map[database.Repository]packet.Set),
		mirrors:       mirrors,
		downloads:     make(map[string]*ongoingDownload),
		repos:         make(map[database.Repository]struct{}),
		repoDownloads: make(map[database.Repository]struct{}),
	}

	err := c.init()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// init scans the cache directory and inits the packet and database caches accordingly
func (c *Cache) init() error {
	// Migrate packages stored directly in the dir to their proper repo location
	migrationList := make([]*packet.Packet, 0)

	err := filepath.Walk(c.directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(c.directory, path)
		if err != nil {
			return err
		}

		if strings.HasSuffix(rel, ".part") {
			return os.Remove(path)
		}

		parts := strings.Split(rel, string(filepath.Separator))

		if len(parts) == 1 {
			filename := parts[0]
			p, err := packet.FromFilename(filename)
			if err != nil {
				return errors.Wrapf(err, "Invalid packet in directory")
			}

			migrationList = append(migrationList, p)
		}

		if len(parts) == 2 {
			arch := parts[0]
			db := parts[1]

			if !strings.HasSuffix(db, ".db") {
				return nil
			}

			c.repos[database.Repository{
				Name: strings.TrimSuffix(db, ".db"),
				Arch: arch,
			}] = struct{}{}
			return err
		}

		if len(parts) == 3 {
			filename := parts[2]
			p, err := packet.FromFilename(filename)
			if err != nil {
				return errors.Wrapf(err, "Invalid packet in directory")
			}

			repo := database.Repository{
				Arch: parts[0],
				Name: parts[1],
			}

			if _, ok := c.packets[repo]; !ok {
				c.packets[repo] = make(packet.Set)
			}

			c.packets[repo].Insert(p)
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "Error reading cache directory")
	}

	return errors.Wrap(c.migrate(migrationList), "Error migrating")
}

// GetPacket serves a packet either from the cache or proxies it from a mirror
// Returns an io.ReadSeaker with access to the packet data
func (c *Cache) GetPacket(p *packet.Packet, repo *database.Repository) (ReadSeekCloser, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Download the packet's repo in the backround if we don't have it yet
	go c.addRepo(repo, nil)

	// First: check if the packet is currently being downloaded
	if download, ok := c.downloads[(&download{P: *p, R: *repo}).Path()]; ok && download.Dl.P == *p {
		return download.GetReader()
	}

	// Second: check if the packet already is available in cache
	if cachedP := c.packets[*repo].ByFilename(p.Filename()); cachedP != nil {
		f, err := os.Open(filepath.Join(c.directory, repo.Arch, repo.Name, cachedP.Filename()))
		if err != nil {
			return nil, errors.Wrap(err, "Error opening cached packet file")
		}

		return f, nil
	}

	// Bail out if newer package version exists
	for _, cachedP := range c.packets[*repo].FindOtherVersions(p) {
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
