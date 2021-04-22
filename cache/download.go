package cache

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/veecue/pacman-smartmirror/database"
	"github.com/veecue/pacman-smartmirror/packet"
)

// ongoingDownload stores neccessary information about an ongoing download to use its data or resume it
type ongoingDownload struct {
	// force alignment of atomically accessed "written" by putting it at the beginning
	// of the struct. 32 bit machines crash otherwise.
	written int64

	Dl       download
	filesize int64

	filename string
}

type download struct {
	P    packet.Packet
	R    database.Repository
	Chan chan<- error
}

func (d *download) Callback(err error) {
	if d.Chan != nil {
		d.Chan <- err
	}
}

func (d *download) Path() string {
	return filepath.Join(d.R.Arch, d.R.Name, d.P.Filename())
}

func (d *download) DirPath() string {
	return filepath.Join(d.R.Arch, d.R.Name)
}

// GetReader returns a ReadSeeker that will read the already downloaded content from the file
// and wait for any undownloaded content (for serving to the client)
func (dl *ongoingDownload) GetReader() (ReadSeekCloser, error) {
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
// When the returned error is nil, the channel will receive a follow-up error (can be nil)
// exactly once
func (c *Cache) startDownload(d *download) (*ongoingDownload, error) {
	for _, mirror := range c.mirrors {
		req, _ := http.NewRequest("GET", mirror.PacketURL(d.P, &d.R), nil)
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
			Dl:       *d,
			filesize: resp.ContentLength,
			filename: filepath.Join(c.directory, d.Path()+".part"),
		}

		// create the directory to store the file in if neccessary
		err = os.MkdirAll(filepath.Join(c.directory, d.DirPath()), 0755)
		if err != nil && !os.IsExist(err) {
			return nil, errors.Wrapf(err, "Error creating dir %s", d.DirPath())
		}

		// create the temporary file to store the download
		f, err := os.Create(dl.filename)
		if err != nil {
			return nil, errors.Wrap(err, "Error creating cache file")
		}

		// store this download to the currently ongoing downloads
		c.downloads[dl.Dl.Path()] = dl

		// do actual download in the background
		go func() {
			w, err := io.Copy(&countWriter{f, &dl.written}, resp.Body)
			f.Close()

			c.mu.Lock()
			defer c.mu.Unlock()

			//TODO: better error handling (#9)
			if err != nil {
				err = errors.Wrap(err, "Error downloading to local cache")
				log.Println(err)
				os.Remove(dl.filename)
				delete(c.downloads, dl.Dl.Path())
				dl.Dl.Callback(err)
				return
			}

			if w < dl.filesize {
				err = errors.New("Too few bytes read while downloading to cache")
				log.Println(err)
				os.Remove(dl.filename)
				delete(c.downloads, dl.Dl.Path())
				dl.Dl.Callback(err)
				return
			}

			go c.finalizeDownload(dl, err)
		}()

		// Return info about ongoing download so it can be served right away
		return dl, nil
	}

	return nil, errors.New("Packet could not be downloaded from any mirror")
}

// Asynchronous callback for a finished download
// The function will rename the .part file to the original file and register it
// in the cache registry.
func (c *Cache) finalizeDownload(dl *ongoingDownload, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Rename donwloaded file to final filename in cache
	err = os.Rename(dl.filename, filepath.Join(c.directory, dl.Dl.Path()))
	if err != nil {
		err = errors.Wrap(err, "Failed moving file")
		log.Println(err)
		os.Remove(dl.filename)
		delete(c.downloads, dl.Dl.Path())
		dl.Dl.Callback(err)
		return
	}

	// Remove old versions
	for _, p := range c.packets[dl.Dl.R].FindOtherVersions(dl.Dl.P) {
		diff := packet.CompareVersions(p.Version(), dl.Dl.P.Version())
		if diff < 0 {
			os.Remove(filepath.Join(c.directory, dl.Dl.R.Arch, dl.Dl.R.Name, p.Filename()))
			c.packets[dl.Dl.R].Delete(p.Filename())
			log.Println("Removed old packet", filepath.Join(dl.Dl.R.Arch, dl.Dl.R.Name, p.Filename()))
		}
	}

	if _, ok := c.packets[dl.Dl.R]; !ok {
		c.packets[dl.Dl.R] = make(packet.Set)
	}

	c.packets[dl.Dl.R].Insert(dl.Dl.P)
	delete(c.downloads, dl.Dl.Path())

	log.Println("Packet", dl.Dl.R, dl.Dl.P.Filename(), "now available!")
	dl.Dl.Callback(nil)
}

// backgroundDownload will start the given download in the background.
// Only one background download will be active at a given time
func (c *Cache) backgroundDownload(dl *download) error {
	c.bgDownload.Lock()
	defer c.bgDownload.Unlock()
	c.mu.Lock()
	if _, ok := c.downloads[dl.Path()]; ok {
		c.mu.Unlock()
		return errors.New("Packet already being downloaded")
	}

	if c.packets[dl.R].ByFilename(dl.P.Filename()) != nil {
		c.mu.Unlock()
		return errors.New("Packet already in cache")
	}

	log.Println("Downloading", dl.Path())
	result := make(chan error)
	dl.Chan = result
	_, err := c.startDownload(dl)
	c.mu.Unlock()

	if err != nil {
		err = errors.Wrap(err, "Error on starting background download")
		log.Println(err)
		return err
	}

	err = <-result
	if err != nil {
		err = errors.Wrap(err, "Error during background download")
		log.Println(err)
		return err
	}

	return nil
}

// countWriter wraps a writer. The total number of bytes written will be appended to *Written
// in an atomic manner.
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
