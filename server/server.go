package server

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/veecue/pacman-smartmirror/cache"
	"github.com/veecue/pacman-smartmirror/database"
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

	if parts[2] != "os" {
		http.NotFound(w, r)
		return
	}

	repo := &database.Repository{
		Name: parts[1],
		Arch: parts[3],
	}

	filename := parts[4]

	if strings.HasSuffix(filename, ".db") {
		if repo.Name != strings.TrimSuffix(filename, ".db") {
			http.NotFound(w, r)
			return
		}

		reader, modtime, err := s.packetCache.GetDBFile(repo)
		if err != nil {
			// Proxy database directly from mirror if not in cache
			s.packetCache.ProxyRepo(w, r, repo)
			return
		}

		defer reader.Close()
		http.ServeContent(w, r, filename, modtime, reader)
		return
	}

	p, err := packet.FromFilename(filename)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	if _, ok := r.URL.Query()["bg"]; r.Method == "HEAD" && ok {
		s.packetCache.AddPacket(p, repo)
		w.WriteHeader(200)
		return
	}

	reader, err := s.packetCache.GetPacket(p, repo)
	if err != nil {
		log.Println("Error serving", p.Filename(), err)
		http.NotFound(w, r)
		return
	}

	defer reader.Close()
	http.ServeContent(w, r, filename, time.Time{}, reader)
}
