package cache

import (
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	"github.com/veecue/pacman-smartmirror/database"
	"github.com/veecue/pacman-smartmirror/packet"
)

func (c *Cache) migrate(toMigrate []*packet.Packet) error {
	if len(toMigrate) > 0 {
		log.Println("Starting migration of", len(toMigrate), "packages...")
	}
	sizes := make(map[packet.Packet]string)
	for _, p := range toMigrate {
		fi, err := os.Stat(filepath.Join(c.directory, p.Filename()))
		if err != nil {
			return errors.Wrapf(err, "Error stating %s", p.Filename())
		}
		sizes[*p] = strconv.FormatInt(fi.Size(), 10)
	}

	type hasRepo struct {
		R database.Repository
		B bool
	}
	cache := make(map[packet.Packet]*hasRepo)
	for repo := range c.repos {
		err := database.ParseDBFromFile(filepath.Join(c.directory, repo.Arch, repo.Name+".db"),
			func(p *packet.Packet, r io.Reader) {
				size, ok := sizes[*p]
				if !ok {
					return
				}
				br := bufio.NewReader(r)
				for {
					line, err := br.ReadString('\n')
					if err != nil {
						break
					}
					if line == "%CSIZE%\n" {
						line, err = br.ReadString('\n')
						if err != nil {
							break
						}
						if line == size+"\n" {
							// Found candidate
							if cached, ok := cache[*p]; ok {
								// Double match, discarding
								cached.B = false
								log.Printf("Double match for %s: found in %s and %s with size %s",
									p.Filename(), cached.R, repo, size)
								break
							}
							cache[*p] = &hasRepo{
								R: repo,
								B: true,
							}
						}
						break
					}
				}
			})
		if err != nil {
			return errors.Wrapf(err, "Error opening %s", filepath.Join(repo.Arch, repo.Name+".db"))
		}
	}

	for p, has := range cache {
		delete(sizes, p)
		if has.B {
			err := os.MkdirAll(filepath.Join(c.directory, has.R.Arch, has.R.Name), 0755)
			if err != nil {
				return errors.Wrapf(err, "Error creating %s", filepath.Join(has.R.Arch, has.R.Name))
			}

			err = os.Rename(
				filepath.Join(c.directory, p.Filename()),
				filepath.Join(c.directory, has.R.Arch, has.R.Name, p.Filename()))

			if err != nil {
				return errors.Wrapf(err, "Error moving %s", p.Filename())
			}

			c.mu.Lock()
			if _, ok := c.packets[has.R]; !ok {
				c.packets[has.R] = make(packet.Set)
			}

			c.packets[has.R].Insert(&p)
			c.mu.Unlock()
		}
	}

	for p := range sizes {
		log.Println("No match found for", p.Filename())
	}

	log.Println("Migration done")
	return nil
}
