package cache

import (
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/veecue/pacman-smartmirror/packet"
)

// ongoingDownload stores neccessary information about an ongoing download to use its data or resume it
type ongoingDownload struct {
	P        packet.Packet
	filesize int64

	// force alignment of atomically accessed "written"
	_       int32
	written int64

	filename string
}

// GetReader returns a ReadSeeker that will read the already downloaded content from the file
// and wait for any undownloaded content (for serving to the client)
func (dl *ongoingDownload) GetReader() (io.ReadSeeker, error) {
	r, err := os.Open(dl.filename)
	if err != nil {
		return nil, err
	}
	return &dynamicLimitReaderWithSize{
		R:     r,
		Size:  dl.filesize,
		Limit: &dl.written,
	}, nil
}

// startDownload will start downloading the given packet from a mirror on the mirrorlist in the
// background and add it to the cache once finished.
//
// Returns info about the ongoing download so it can be served to the client.
func (c *Cache) startDownload(p *packet.Packet, repo string) (*ongoingDownload, error) {
	for _, mirror := range c.mirrors {
		req, _ := http.NewRequest("GET", mirror.PacketURL(p, repo), nil)
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
		dl := &ongoingDownload{
			P:        *p,
			filesize: resp.ContentLength,
			filename: path.Join(c.directory, p.Filename()+".part"),
		}

		// create the temporary file to store the download
		f, err := os.Create(dl.filename)
		if err != nil {
			return nil, errors.Wrap(err, "Error creating cache file")
		}

		// store this download to the currently ongoing downloads
		c.downloads[dl] = struct{}{}

		// do actual download in the background
		go func() {
			w, err := io.Copy(&countWriter{f, &dl.written}, resp.Body)
			f.Close()

			// Download done, lock Cache to move around stuff
			c.mu.Lock()
			defer c.mu.Unlock()

			//TODO: better error handling (#9)
			if err != nil {
				log.Println("Error downloading to local cache:", err)
				os.Remove(dl.filename)
				delete(c.downloads, dl)
				return
			}

			if w < dl.filesize {
				log.Println("Too few bytes read while downloading to cache")
				os.Remove(dl.filename)
				delete(c.downloads, dl)
				return
			}

			// Rename donwloaded file to final filename in cache
			err = os.Rename(dl.filename, path.Join(c.directory, dl.P.Filename()))
			if err != nil {
				log.Println("Failed moving file")
				os.Remove(dl.filename)
				delete(c.downloads, dl)
				return
			}

			c.packets[&dl.P] = struct{}{}
			delete(c.downloads, dl)

			log.Println("Packet", dl.P.Filename(), "now available!")
		}()

		// Return info about ongoing download so it can be served right away
		return dl, nil
	}

	return nil, errors.New("Packet could not be downloaded from any mirror")
}

type countWriter struct {
	W       io.Writer
	Written *int64
}

func (l *countWriter) Write(p []byte) (int, error) {
	n, err := l.W.Write(p)
	atomic.AddInt64(l.Written, int64(n))
	return n, err
}

// dynamicLimitReaderWithSize allows files to be read that aren't written completly
// Expects that
//  - Limit grows steadily
//  - R returns EOF after Size
// Guarantuees that
//  - R is not read after limit
//
// Additionally passes through close commands if R also is a closer
type dynamicLimitReaderWithSize struct {
	R     io.ReadSeeker
	Size  int64
	Limit *int64
	pos   int64
}

func (d *dynamicLimitReaderWithSize) Read(p []byte) (n int, err error) {
	limit := atomic.LoadInt64(d.Limit)
	if d.pos >= limit {
		// still waiting for data to get available
		return 0, nil
	}

	if d.pos+int64(len(p)) > limit {
		// reading would go beyond limit
		p = p[:int(limit-d.pos)]
	}

	n, err = d.R.Read(p)
	d.pos += int64(n)
	return
}

func (d *dynamicLimitReaderWithSize) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		d.pos = offset
	case io.SeekCurrent:
		d.pos += offset
	case io.SeekEnd:
		d.pos = d.Size + offset
	default:
		return d.pos, errors.New("Invalid whence")
	}

	// Also tell the underlying reader to seek so that we read properly
	//TODO: error handling
	d.R.Seek(offset, whence)

	if d.pos > d.Size || d.pos < 0 {
		return d.pos, errors.New("Seek out of bounds")
	}

	return d.pos, nil
}

func (d *dynamicLimitReaderWithSize) Close() error {
	if closer, ok := d.R.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
