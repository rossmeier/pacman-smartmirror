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
