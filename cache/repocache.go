package cache

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/veecue/pacman-smartmirror/database"
)

func (c *Cache) downloadRepo(repo *database.Repository) {
	c.repoMu.Lock()
	defer c.repoMu.Unlock()
	if _, ok := c.repos[*repo]; ok {
		return
	}

	if _, ok := c.repoDownloads[*repo]; ok {
		return
	}

	for _, mirror := range c.mirrors {
		req, _ := http.NewRequest("GET", mirror.RepoURL(repo), nil)
		req.Header.Set("User-Agent", "pacman-smartmirror/0.0")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			//TODO: log?
			continue
		}

		if resp.StatusCode != 200 {
			continue
		}

		// seems to work, use this mirror
		file := filepath.Join(c.directory, repo.Arch, repo.Name+".db")

		err = os.Mkdir(filepath.Join(c.directory, repo.Arch), 0755)
		if err != nil && !os.IsExist(err) {
			log.Println("Error creating cache dir:", err)
			return
		}

		// create the temporary file to store the download
		f, err := os.Create(file + ".part")
		if err != nil {
			log.Println("Error creating repo file:", errors.Wrap(err, "Error creating cache file"))
			return
		}

		c.repoDownloads[*repo] = struct{}{}

		go func() {
			_, err := io.CopyN(f, resp.Body, resp.ContentLength)
			if err != nil {
				log.Println("Error downloading repo file:", err)
				os.Remove(file + ".part")
				return
			}

			c.repoMu.Lock()
			defer c.repoMu.Unlock()

			os.Remove(file)
			err = os.Rename(file+".part", file)
			if err != nil {
				log.Println("Error moving repo file:", err)
				os.Remove(file + ".part")
				return
			}

			//TODO: Make sure that all updated packages from the repo are downloaded at this point

			c.repos[*repo] = struct{}{}
			delete(c.repoDownloads, *repo)
		}()

		break
	}
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
func (c *Cache) GetDBFile(repo *database.Repository) (io.ReadSeeker, error) {
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
