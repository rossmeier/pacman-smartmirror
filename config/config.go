package config

import "flag"

// C represents the applications current config
var C struct {
	CacheDirectory string
	MirrorlistFile string
	Listen         string
}

func init() {
	flag.StringVar(&C.CacheDirectory, "d", "", "Directory to use for the cached packages")
	flag.StringVar(&C.MirrorlistFile, "m", "", "Filename of the mirrorlist to use")
	flag.StringVar(&C.Listen, "l", ":41234", "Address and port for the HTTP server to listen on")
	flag.Parse()
}
