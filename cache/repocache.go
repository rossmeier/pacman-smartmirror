package cache

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/veecue/pacman-smartmirror/database"
)

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

		// seems to work, use this mirror
		var serverModTime *time.Time
		if t, err := http.ParseTime(resp.Header.Get("Last-Modified")); err == nil {
			serverModTime = &t
		}

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

		// create the temporary file to store the download
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

			//TODO: Make sure that all updated packages from the repo are downloaded at this point

			c.repos[*repo] = struct{}{}
			delete(c.repoDownloads, *repo)

			callback(err)
		}()

		return nil
	}

	return errors.New("Database could not be downloaded from any mirror")
}

func (c *Cache) initRepos() error {
	return filepath.Walk(c.directory, func(path string, info os.FileInfo, err error) error {
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

		arch, db := filepath.Split(rel)
		if arch == "" {
			return nil
		}
		if !strings.HasSuffix(db, ".db") {
			if strings.HasSuffix(db, ".part") {
				return os.Remove(path)
			}
			return nil
		}

		// remove final slash
		arch = filepath.Clean(arch)

		c.repos[database.Repository{
			Name: strings.TrimSuffix(db, ".db"),
			Arch: arch,
		}] = struct{}{}
		return err
	})
}

// GetDBFile serves the latest cached version of a given database
func (c *Cache) GetDBFile(repo *database.Repository) (ReadSeekCloser, error) {
	if _, ok := c.repos[*repo]; ok {
		c.repoMu.Lock()
		defer c.repoMu.Unlock()

		path := filepath.Join(c.directory, repo.Arch, repo.Name+".db")
		file, err := os.Open(path)
		if err != nil {
			return nil, errors.Wrap(err, "Error opening repository file")
		}

		return file, nil
	}

	return nil, errors.New("Database file not found")
}

// ProxyRepo will proxy the given repository database file from a mirror
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
// TODO: also update packages
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
