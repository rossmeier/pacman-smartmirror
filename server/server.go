package server

import (
	"errors"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/veecue/pacman-smartmirror/cache"
	"github.com/veecue/pacman-smartmirror/impl"
)

// Server is an http proxy server that uses a cache
type Server struct {
	packetCache *cache.Cache
}

// New will create a new Server from the given packet cache
func New(packetCache *cache.Cache) *Server {
	return &Server{
		packetCache: packetCache,
	}
}

// ServeHTTP implements the http.Server interface serving cached
// packets and automatically retrieving missing packages from
// the cache.
// Requests should be in the following form:
// /$repo/os/$arch/$file.pkg.tar.xz
// This is how most arch upstream mirrors are called
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// avoid infinite self-loopback
	if strings.HasPrefix(r.UserAgent(), "pacman-smartmirror/") {
		w.WriteHeader(403)
		return
	}

	if _, ok := r.URL.Query()["bg"]; r.Method == "HEAD" && ok {
		go s.packetCache.AddPacket(r.URL.Path)
		w.WriteHeader(200)
		return
	}

	reader, err := s.packetCache.GetPacket(r.URL.Path)
	if err == nil {
		defer reader.Close()
		http.ServeContent(w, r, "", time.Time{}, reader)
		return
	}

	if !errors.Is(err, impl.ErrInvalidFilename) {
		log.Println(err)
		http.NotFound(w, r)
		return
	}

	// maybe it is a repo file
	reader, t, err := s.packetCache.GetDBFile(path.Dir(r.URL.Path))
	if err == nil {
		defer reader.Close()
		http.ServeContent(w, r, path.Base(r.URL.Path), t, reader)
		return
	}

	s.packetCache.ProxyRepo(w, r)
}
