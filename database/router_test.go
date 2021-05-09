package database_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/veecue/pacman-smartmirror/config"
	"github.com/veecue/pacman-smartmirror/database"

	_ "github.com/veecue/pacman-smartmirror/impl/apk"
	_ "github.com/veecue/pacman-smartmirror/impl/pacman"
)

var cfg = config.RepoConfigs{
	"archlinux/$repo/os/$arch": {
		Impl: "pacman",
		Args: map[string]string{
			"reponame": "$repo",
		},
		Upstreams: []string{"https://ftp.halifax.rwth-aachen.de/archlinux/$repo/os/$arch"},
	},
	"alpine/$version/$repo/$arch": {
		Impl:      "apk",
		Upstreams: []string{"https://dl-cdn.alpinelinux.org/alpine/$version/$repo/$arch"},
	},
}

func TestCfg(t *testing.T) {
	r := database.NewRouter(cfg)
	res := r.MatchPath("archlinux/core/os/x86_64/zstd-1.4.9-1-x86_64.pkg.tar.zst")
	assert.ElementsMatch(t, res.UpstreamURLs, []string{
		"https://ftp.halifax.rwth-aachen.de/archlinux/core/os/x86_64/zstd-1.4.9-1-x86_64.pkg.tar.zst",
	})
	res = r.MatchPath("alpine/v3.13/main/armhf/a52dec-dev-0.7.4-r7.apk")
	assert.ElementsMatch(t, res.UpstreamURLs, []string{
		"https://dl-cdn.alpinelinux.org/alpine/v3.13/main/armhf/a52dec-dev-0.7.4-r7.apk",
	})
}
