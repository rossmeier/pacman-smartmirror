package test

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {
	s := NewServer(t, func(w http.ResponseWriter, filename, repo, arch string) {
		assert.Equal(t, "linux-5.2.arch2-1-x86_64.pkg.tar.xz", filename)
		assert.Equal(t, "core", repo)
		assert.Equal(t, "x86_64", arch)
		io.WriteString(w, "heyoo")
	})
	defer s.StopServer(t)
	u := strings.ReplaceAll(s.URL, "$repo", "core")
	u = strings.ReplaceAll(u, "$arch", "x86_64")
	r, err := http.Get(u + "linux-5.2.arch2-1-x86_64.pkg.tar.xz")
	assert.NoError(t, err)
	b, err := ioutil.ReadAll(r.Body)
	assert.NoError(t, err)
	assert.Equal(t, "heyoo", string(b))
}
