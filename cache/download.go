package cache

import (
	"log"
	"os"
	"path/filepath"

	"github.com/veecue/pacman-smartmirror/packet"
)

// Asynchronous callback for a finished download
// The function will register the downloaded file in the registry
func (c *Cache) finalizeDownload(repopath string, pkg packet.Packet) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove old versions
	for _, p := range c.packets[repopath].FindOtherVersions(pkg) {
		diff := packet.CompareVersions(p.Version(), pkg.Version())
		if diff < 0 {
			os.Remove(filepath.Join(c.directory, repopath, p.Filename()))
			c.packets[repopath].Delete(p.Filename())
			log.Println("Removed old packet", filepath.Join(repopath, p.Filename()))
		}
	}

	if _, ok := c.packets[repopath]; !ok {
		c.packets[repopath] = make(packet.Set)
	}

	c.packets[repopath].Insert(pkg)

	log.Println("Packet", repopath, pkg.Filename(), "now available!")
}
