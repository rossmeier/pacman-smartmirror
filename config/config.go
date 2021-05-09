package config

import (
	"flag"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

var filepath = flag.String("c", "config.yml", "Configuration file")

// RepoConfig is the configuration for one specific remote repository
type RepoConfig struct {
	Impl      string            `yaml:"impl"`
	Upstreams []string          `yaml:"upstreams"`
	Args      map[string]string `yaml:"args"`
}

// RepoConfigs are many RepoConfigs at once
type RepoConfigs map[string]RepoConfig

// C represents the applications current config
var C struct {
	CacheDirectory string      `yaml:"cache_dir"`
	Listen         string      `yaml:"listen"`
	Repos          RepoConfigs `yaml:"repos"`
}

// Init intializes the config from the parsed commandline flags
// needs to be called after flag.Parse()
func Init() {
	data, err := ioutil.ReadFile(*filepath)
	if err != nil {
		log.Fatal("Error reading config file:", err)
	}
	err = yaml.Unmarshal(data, &C)
	if err != nil {
		log.Fatal("Error parsing config file:", err)
	}
}
