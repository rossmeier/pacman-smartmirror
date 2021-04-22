package cache

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/veecue/pacman-smartmirror/database"
	"github.com/veecue/pacman-smartmirror/packet"
)

// checks if the given repository is already in the repository cache and downloads
// it asynchronously. If the function returns no immediate error (nil), it will write
// the final error to the channel if the channel is not nil.
func (c *Cache) addRepo(repo *database.Repository, result chan<- error) error {
	c.repoMu.Lock()
	_, ok := c.repos[*repo]
	c.repoMu.Unlock()
	if ok {
		return errors.New("Repo already available")
	}

	log.Println("Downloading repo", repo)
	err := c.downloadRepo(repo, result)
	if err == nil {
		log.Println("Repo", repo, "now available")
	} else {
		log.Println("Error downloading", repo, err)
	}

	return err
}

// downloadRepo will download the database file of the given repository and add
// it to the repository cache. If no immediate error occurs (nil is returned),
// the final error will be pushed to the given channel if the channel is not nil.
func (c *Cache) downloadRepo(repo *database.Repository, result chan<- error) error {
	callback := func(err error) {
		if result != nil {
			result <- err
		}
	}

	c.repoMu.Lock()
	defer c.repoMu.Unlock()

	if _, ok := c.repoDownloads[*repo]; ok {
		return errors.New("Repo is already being downloaded")
	}

	file := filepath.Join(c.directory, repo.Arch, repo.Name+".db")

	// Send the modtime of the cached file to the server so only a later
	// version is downloaded.
	var modTime *time.Time
	if _, ok := c.repos[*repo]; ok {
		stat, err := os.Stat(file)
		if err == nil {
			t := stat.ModTime()
			modTime = &t
		}
	}

	for _, mirror := range c.mirrors {
		req, _ := http.NewRequest("GET", mirror.RepoURL(repo), nil)

		req.Header.Set("User-Agent", "pacman-smartmirror/0.0")
		if modTime != nil {
			req.Header.Set("If-Modified-Since", modTime.Format(http.TimeFormat))
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			//TODO: log?
			continue
		}

		if resp.StatusCode == 304 {
			log.Println("Database", repo, "already up to date")
			go callback(nil)
			return nil
		}

		if resp.StatusCode != 200 {
			continue
		}

		if resp.ContentLength <= 0 {
			continue
		}

		// Seems to work, use this mirror
		var serverModTime *time.Time
		if t, err := http.ParseTime(resp.Header.Get("Last-Modified")); err == nil {
			serverModTime = &t
		}

		// Cancel download if the file given by the server is older than the local file
		if modTime != nil && serverModTime != nil && (modTime.After(*serverModTime) || modTime.Equal(*serverModTime)) {
			log.Println("Database", repo, "already up to date")
			go callback(nil)
			return nil
		}

		err = os.Mkdir(filepath.Join(c.directory, repo.Arch), 0755)
		if err != nil && !os.IsExist(err) {
			err = errors.Wrap(err, "Error creating cache dir")
			log.Println(err)
			return err
		}

		// Create the temporary file to store the download
		f, err := os.Create(file + ".part")
		if err != nil {
			err = errors.Wrap(err, "Error creating repo file")
			log.Println(err)
			return err
		}

		c.repoDownloads[*repo] = struct{}{}

		go func() {
			_, err := io.CopyN(f, resp.Body, resp.ContentLength)
			if err != nil {
				err = errors.Wrap(err, "Error downloading repo file")
				log.Println(err)
				os.Remove(file + ".part")
				callback(err)
				return
			}

			c.repoMu.Lock()
			defer c.repoMu.Unlock()

			os.Remove(file)
			err = os.Rename(file+".part", file)
			if err != nil {
				err = errors.Wrap(err, "Error moving repo file")
				log.Println(err)
				os.Remove(file + ".part")
				callback(err)
				return
			}

			if serverModTime != nil {
				os.Chtimes(file, time.Now(), *serverModTime)
			}

			c.repos[*repo] = struct{}{}
			delete(c.repoDownloads, *repo)

			callback(err)
		}()

		return nil
	}

	return errors.New("Database could not be downloaded from any mirror")
}

// updatePackets will update all locally cached packages that are part of the given repository
func (c *Cache) updatePackets(repo database.Repository) {
	// List of packages that are out of date
	toDownload := make([]packet.Packet, 0)
	err := database.ParseDBFromFile("pacman", filepath.Join(c.directory, repo.Arch, repo.Name+".db"), func(p packet.Packet, _ io.Reader) {
		c.mu.Lock()
		for _, other := range c.packets[repo].FindOtherVersions(p) {
			if packet.CompareVersions(p.Version(), other.Version()) > 0 {
				// Version in the repository is later than the local one
				toDownload = append(toDownload, p)
				break
			}
		}
		c.mu.Unlock()
	})

	if err != nil {
		log.Println("Error parsing db file:", err)
		return
	}

	// Update all outdated packages
	for _, p := range toDownload {
		if c.backgroundDownload(&download{p, repo, nil}) == nil {
			if err != nil {
				log.Println(errors.Wrapf(err, "Error downloading %s", p.Filename()))
			}
		}
	}

	log.Println("All cached packages for", repo, "up to date")
}

// GetDBFile returns the latest cached version of a given database together with
// the time it was updated.
func (c *Cache) GetDBFile(repo *database.Repository) (ReadSeekCloser, time.Time, error) {
	if _, ok := c.repos[*repo]; ok {
		c.repoMu.Lock()
		defer c.repoMu.Unlock()

		path := filepath.Join(c.directory, repo.Arch, repo.Name+".db")
		file, err := os.Open(path)
		if err != nil {
			return nil, time.Time{}, errors.Wrap(err, "Error opening repository file")
		}

		modtime := time.Time{}
		if stat, err := os.Stat(path); err == nil {
			modtime = stat.ModTime()
		}

		return file, modtime, nil
	}

	return nil, time.Time{}, errors.New("Database file not found")
}

// ProxyRepo will proxy the given repository database file from a mirror
// with out downloading it to the cache.
func (c *Cache) ProxyRepo(w http.ResponseWriter, r *http.Request, repo *database.Repository) {
	for _, mirror := range c.mirrors {
		req, _ := http.NewRequest("GET", mirror.RepoURL(repo), nil)
		req.Header = r.Header
		req.Header.Set("User-Agent", "pacman-smartmirror/0.0")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != 200 && resp.StatusCode != 304 {
			continue
		}

		// seems to work, use this mirror
		for key := range resp.Header {
			w.Header().Add(key, resp.Header.Get(key))
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		return
	}

	http.NotFound(w, r)
}

// UpdateDatabases will update all cached database files in the background
// The error (may be nil) WILL be sent to the channel EXACTLY ONCE
//
// For each updated database, updating their accoring cached packages
// will be started in the background.
func (c *Cache) UpdateDatabases(result chan<- error) {
	// Gather list of all databases
	toUpdate := make([]database.Repository, 0)
	c.repoMu.Lock()
	for repo := range c.repos {
		toUpdate = append(toUpdate, repo)
	}
	c.repoMu.Unlock()

	go func() {
		var lastErr error
		subresults := make(chan error)
		for _, repo := range toUpdate {
			log.Println("Updating", repo)
			err := c.downloadRepo(&repo, subresults)
			if err != nil {
				lastErr = errors.Wrap(err, "Error updating databases")
				log.Println(lastErr)
				continue
			}

			err = <-subresults
			if err != nil {
				lastErr = errors.Wrap(err, "Error updating databases")
				log.Println(lastErr)
			}

			go c.updatePackets(repo)
		}
		if lastErr == nil {
			log.Println("All databases updated successfully")
		} else {
			log.Println("Error(s) during database updates")
		}

		if result != nil {
			result <- lastErr
		}
	}()
}
