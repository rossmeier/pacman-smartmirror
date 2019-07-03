package server

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/veecue/pacman-smartmirror/cache"
	"github.com/veecue/pacman-smartmirror/packet"
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

	parts := strings.Split(r.RequestURI, "/")
	if len(parts) != 5 {
		http.NotFound(w, r)
		return
	}

	repo := parts[1]
	if parts[2] != "os" {
		http.NotFound(w, r)
		return
	}

	arch := parts[3]
	filename := parts[4]

	if strings.HasSuffix(filename, ".db") {
		// Ignore database files for now, maybe in the future
		http.NotFound(w, r)
		return
	}

	p, err := packet.FromFilename(filename)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	if arch != p.Arch {
		http.NotFound(w, r)
		return
	}

	reader, err := s.packetCache.GetPacket(p, repo)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	http.ServeContent(w, r, filename, time.Time{}, reader)

	if closer, ok := reader.(io.Closer); ok {
		closer.Close()
	}
}
