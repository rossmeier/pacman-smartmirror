package database

import (
	"path"
	"strings"

	"github.com/veecue/pacman-smartmirror/config"
	"github.com/veecue/pacman-smartmirror/impl"
	"github.com/veecue/pacman-smartmirror/packet"
)

// Router is a type that dynamically routes package and repo paths according to the given
// RepoConfigs and returns the corresponding implementations
type Router struct {
	cfg config.RepoConfigs
}

// RouterMatch is a specific route that was found by the router
type RouterMatch struct {
	UpstreamURLs []string
	Impl         impl.Impl
	MatchedPath  string
	Filename     string
}

// DBPath returns the path of the databse index file for the current
// router match (for the current repository)
func (r *RouterMatch) DBPath() string {
	return r.Impl.GetDB(r.MatchedPath)
}

// Path returns the cannonical matched path to the matched file
func (r *RouterMatch) Path() string {
	return path.Join(r.MatchedPath, r.Filename)
}

// Packet parses the implementation-specific packet information
// form the matched file
func (r *RouterMatch) Packet() (packet.Packet, error) {
	return r.Impl.PacketFromFilename(r.Filename)
}

// NewRouter creates a new router from the given RepoConfigs
func NewRouter(cfg config.RepoConfigs) *Router {
	return &Router{
		cfg: cfg,
	}
}

// MatchPath matches the given slash-separated to one of the configured
// RepoConfigs and returns the resulting RouterMatch.
// Returns nil if no match was found.
func (r *Router) MatchPath(searchpath string) *RouterMatch {
	searchpath = path.Clean(searchpath)
	searchpath = strings.TrimPrefix(searchpath, "/")
	for p, v := range r.cfg {
		vars := make(map[string]string)
		pParts := strings.Split(p, "/")
		searchpathParts := strings.Split(searchpath, "/")
		if len(pParts) > len(searchpathParts) || len(pParts)+1 < len(searchpathParts) {
			continue
		}

		matches := true
		for i, part := range pParts {
			if part[0] == '$' {
				vars[part] = searchpathParts[i]
			} else if part != searchpathParts[i] {
				matches = false
				break
			}
		}

		if matches {
			replacevars := func(s string) string {
				for key, val := range vars {
					s = strings.ReplaceAll(s, key, val)
				}
				return s
			}

			implargs := make(map[string]string)
			for key := range v.Args {
				implargs[key] = replacevars(v.Args[key])
			}

			res := &RouterMatch{
				UpstreamURLs: make([]string, len(v.Upstreams)),
				Impl:         impl.Get(v.Impl, implargs),
			}
			copy(res.UpstreamURLs, v.Upstreams)

			if len(pParts) < len(searchpathParts) {
				res.Filename = searchpathParts[len(searchpathParts)-1]
			}
			res.MatchedPath = path.Join(searchpathParts[:len(pParts)]...)

			// replace variables in upstream
			for i := range res.UpstreamURLs {
				res.UpstreamURLs[i] = replacevars(res.UpstreamURLs[i])
				if res.Filename != "" {
					res.UpstreamURLs[i] = strings.TrimSuffix(res.UpstreamURLs[i], "/") + "/" + res.Filename
				}
			}

			return res
		}
	}

	return nil
}
