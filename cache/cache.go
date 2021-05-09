package cache

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/veecue/pacman-smartmirror/cache/downloadmanager"
	"github.com/veecue/pacman-smartmirror/config"
	"github.com/veecue/pacman-smartmirror/database"
	"github.com/veecue/pacman-smartmirror/packet"
)

// Cache is a cache that caches packages in the filesystem.
// The currently only implementation at the moment is storing
// it in a directory in the filesystem.
type Cache struct {
	directory       string
	packets         map[string]packet.Set
	repos           map[string]struct{}
	downloadManager *downloadmanager.DownloadManager
	r               *database.Router
	mu              sync.Mutex
	repoMu          sync.Mutex
}

// ReadSeekCloser implements io.ReadSeeker and io.Closer
type ReadSeekCloser interface {
	io.ReadSeeker
	io.Closer
}

// New creates a new cache from a given directory
func New(directory string, cfg config.RepoConfigs) (*Cache, error) {
	c := &Cache{
		directory:       directory,
		downloadManager: downloadmanager.New(),
		packets:         make(map[string]packet.Set),
		repos:           make(map[string]struct{}),
		r:               database.NewRouter(cfg),
	}

	err := c.init()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// init scans the cache directory and inits the packet and database caches accordingly
func (c *Cache) init() error {
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

		match := c.r.MatchPath(rel)
		if match == nil {
			// ignore unmatched files
			log.Printf("Found unkown path %s, migrate data or adapt config", rel)
			return nil
		}

		if match.Path() == match.DBPath() {
			// repository file
			c.repos[match.MatchedPath] = struct{}{}
			return nil
		}

		packetinfo, err := match.Packet()
		if err != nil {
			return fmt.Errorf("error parsing packet filename %s: %w", match.Filename, err)
		}

		if _, ok := c.packets[match.MatchedPath]; !ok {
			c.packets[match.MatchedPath] = make(packet.Set)
		}
		c.packets[match.MatchedPath].Insert(packetinfo)

		return nil
	})

	return err
}

// GetPacket serves a packet either from the cache or proxies it from a mirror
// Returns an io.ReadSeaker with access to the packet data
func (c *Cache) GetPacket(searchpath string) (ReadSeekCloser, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	match := c.r.MatchPath(searchpath)
	if match == nil {
		return nil, fmt.Errorf("no match for packet path %s", searchpath)
	}

	p, err := match.Packet()
	if err != nil {
		return nil, fmt.Errorf("error parsing packet for filename %s: %w", match.Filename, err)
	}

	// Bail out if newer package version exists
	for _, cachedP := range c.packets[match.MatchedPath].FindOtherVersions(p) {
		versionDiff := packet.CompareVersions(p.Version(), cachedP.Version())
		if versionDiff < 0 {
			return nil, errors.New("newer version available")
		}
	}

	// Download packet to cache (or serve from ongoing download)
	result := make(chan error)
	rd, async, err := c.downloadManager.GetFile(filepath.Join(c.directory, filepath.FromSlash(match.Path())), match.UpstreamURLs, result, false)
	if err != nil {
		return nil, fmt.Errorf("error downloading the packet: %w", err)
	}

	if async {
		go func() {
			err := <-result
			if err != nil {
				log.Println("Error downloading packet:", err)
				return
			}
			c.finalizeDownload(match.MatchedPath, p)
		}()
	}

	// Download the packet's repo in the backround if we don't have it yet
	go c.addRepo(match.MatchedPath)

	return rd, nil
}

// AddPacket synchronously downloads the given packet in the background when possible and
// adds it to the cache afterwards
func (c *Cache) AddPacket(searchpath string) error {
	match := c.r.MatchPath(searchpath)
	if match == nil {
		return fmt.Errorf("no match for packet path %s", searchpath)
	}

	p, err := match.Packet()
	if err != nil {
		return fmt.Errorf("error parsing packet for filename %s: %w", match.Filename, err)
	}

	// Bail out if newer package version exists
	c.mu.Lock()
	for _, cachedP := range c.packets[match.MatchedPath].FindOtherVersions(p) {
		versionDiff := packet.CompareVersions(p.Version(), cachedP.Version())
		if versionDiff < 0 {
			c.mu.Unlock()
			return errors.New("newer version available")
		}
	}
	c.mu.Unlock()

	// Download packet to cache (or serve from ongoing download)
	err = c.downloadManager.BackgroundDownload(filepath.Join(c.directory, filepath.FromSlash(match.Path())), match.UpstreamURLs)
	if err != nil {
		return fmt.Errorf("error downloading the packet: %w", err)
	}

	c.finalizeDownload(match.MatchedPath, p)

	// Download the packet's repo in the backround if we don't have it yet
	c.addRepo(match.MatchedPath)

	return nil
}
