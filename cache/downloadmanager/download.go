package downloadmanager

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// ReadSeekCloser implements io.ReadSeeker and io.Closer
type ReadSeekCloser interface {
	io.ReadSeeker
	io.Closer
}

// ongoingDownload stores neccessary information about an ongoing download to use its data or resume it
type ongoingDownload struct {
	// force alignment of atomically accessed "written" by putting it at the beginning
	// of the struct. 32 bit machines crash otherwise.
	written int64

	filesize int64

	tmpPath       string
	path          string
	remainingURLs []string
	result        chan<- error
}

func (d *ongoingDownload) Callback(err error) {
	if d.result != nil {
		go func() {
			d.result <- err
		}()
	}
}

// GetReader returns a ReadSeeker that will read the already downloaded content from the file
// and wait for any undownloaded content (for serving to the client)
func (d *ongoingDownload) GetReader() (ReadSeekCloser, error) {
	r, err := os.Open(d.tmpPath)
	if err != nil {
		return nil, err
	}
	return &dynamicLimitReaderWithSize{
		R:     r,
		Size:  d.filesize,
		Limit: &d.written,
	}, nil
}

// DownloadManager supports concurrency-safe downloading of files from different mirror-locations
// and accessing those files while they are still being downloaded
type DownloadManager struct {
	ongoing    map[string]*ongoingDownload
	mu         sync.Mutex
	bgDownload sync.Mutex
}

// New create a new DownloadManager instance
func New() *DownloadManager {
	return &DownloadManager{
		ongoing: make(map[string]*ongoingDownload),
	}
}

// GetFile downloads the file from one of the given url alternatives to the given file path
// iff err == nil and async is true the final error value will be sent through the result channel.
// forceRedownload can be used to force a redownload of existing files instead of just accessing
// the existing one
func (m *DownloadManager) GetFile(path string, urlAlternatives []string, result chan<- error, forceRedownload bool) (ReadSeekCloser, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ongoing, ok := m.ongoing[path]; ok {
		rd, err := ongoing.GetReader()
		return rd, false, err
	}

	exists := false
	info, err := os.Stat(path)
	if err == nil {
		if !forceRedownload {
			// Just return already downloaded file
			f, err := os.Open(path)
			return f, false, err
		}
		exists = true
	} else if !os.IsNotExist(err) {
		return nil, false, fmt.Errorf("fatal error accessing file: %w", err)
	}

	for i, url := range urlAlternatives {
		req, _ := http.NewRequest("GET", url, nil)
		//req.Header.Set("User-Agent", "pacman-smartmirror/0.0")
		if exists {
			req.Header.Set("If-Modified-Since", info.ModTime().Format(http.TimeFormat))
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			//TODO: log?
			continue
		}

		if resp.StatusCode == 304 {
			f, err := os.Open(path)
			return f, false, err
		}

		if resp.StatusCode != 200 {
			continue
		}

		// seems to work, use this mirror
		dl := &ongoingDownload{
			filesize:      resp.ContentLength,
			tmpPath:       path + ".part",
			path:          path,
			remainingURLs: urlAlternatives[i:],
			result:        result,
		}

		// create the directory to store the file in if neccessary
		err = os.MkdirAll(filepath.Dir(dl.tmpPath), 0755)
		if err != nil && !os.IsExist(err) {
			return nil, false, fmt.Errorf("error creating dir %s: %w", filepath.Dir(dl.tmpPath), err)
		}

		// create the temporary file to store the download
		f, err := os.Create(dl.tmpPath)
		if err != nil {
			return nil, false, fmt.Errorf("error creating cache file: %w", err)
		}

		// store this download to the currently ongoing downloads
		m.ongoing[dl.path] = dl

		// get handle to downloaded os file
		rd, err := dl.GetReader()

		// do actual download in the background
		go func() {
			w, err := io.Copy(&countWriter{f, &dl.written}, resp.Body)
			f.Close()

			m.mu.Lock()
			defer m.mu.Unlock()

			//TODO: better error handling
			if err != nil {
				err = fmt.Errorf("error downloading to local cache: %w", err)
				log.Println(err)
				os.Remove(dl.tmpPath)
				delete(m.ongoing, dl.path)
				dl.Callback(err)
				return
			}

			if w < dl.filesize {
				err = fmt.Errorf("too few bytes read while downloading to cache")
				log.Println(err)
				os.Remove(dl.tmpPath)
				delete(m.ongoing, dl.path)
				dl.Callback(err)
				return
			}

			// Rename donwloaded file to final filename in cache
			err = os.Rename(dl.tmpPath, dl.path)
			if err != nil {
				err = fmt.Errorf("failed moving file: %w", err)
				log.Println(err)
				os.Remove(dl.tmpPath)
				delete(m.ongoing, dl.path)
				dl.Callback(err)
				return
			}

			delete(m.ongoing, dl.path)
			dl.Callback(nil)
		}()

		return rd, true, err
	}

	return nil, false, errors.New("file could not be downloaded from any given URL")
}

// BackgroundDownload will start the given download in the background.
// Only one background download will be active at a given time
func (m *DownloadManager) BackgroundDownload(path string, urlAlternatives []string) error {
	// lock mutex so only one download can happen at a time
	m.bgDownload.Lock()
	defer m.bgDownload.Unlock()

	log.Println("Downloading", path)
	result := make(chan error)
	rd, async, err := m.GetFile(path, urlAlternatives, result, false)

	if err != nil {
		err = fmt.Errorf("error on starting background download: %w", err)
		log.Println(err)
		return err
	}

	// we don't care about the data
	// TODO: Optimize
	rd.Close()

	if !async {
		// did not start a real download, file was already there or being downloaded
		return nil
	}

	// wait for download to finish
	err = <-result
	if err != nil {
		err = fmt.Errorf("error during background download: %w", err)
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
		return d.pos, errors.New("invalid whence")
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
