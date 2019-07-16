package test

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Callback is the function to use the HTTP Server
type Callback func(w http.ResponseWriter, filename string, repo string, arch string)

// Server is a simple local HTTP server listening on the given URL
type Server struct {
	URL    string
	server http.Server
}

// NewServer creates a server used for testing
func NewServer(t *testing.T, f Callback) *Server {
	s := &Server{}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port

	s.server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.RequestURI, "/")
		if len(parts) != 5 {
			http.NotFound(w, r)
			return
		}

		if parts[2] != "os" {
			http.NotFound(w, r)
			return
		}

		repo := parts[1]
		arch := parts[3]

		filename := parts[4]

		if strings.HasSuffix(filename, ".db") {
			// Ignore database files for now, maybe in the future
			http.NotFound(w, r)
			return
		}

		f(w, filename, repo, arch)
	})

	s.URL = "http://127.0.0.1:" + strconv.Itoa(port) + "/$repo/os/$arch/"

	go s.server.Serve(l)
	return s
}

// StopServer stops the server
func (s *Server) StopServer(t *testing.T) {
	assert.NoError(t, s.server.Shutdown(context.Background()))
}
