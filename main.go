package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/veecue/pacman-smartmirror/cache"
	"github.com/veecue/pacman-smartmirror/config"
	"github.com/veecue/pacman-smartmirror/mirrorlist"
	"github.com/veecue/pacman-smartmirror/server"
)

func main() {
	flag.Parse()
	log.Printf(`Loading mirrorlist file: "%s"`, config.C.MirrorlistFile)
	m, err := mirrorlist.FromFile(config.C.MirrorlistFile)
	if err != nil {
		log.Fatalf(`Error reading mirrorlist "%s": %v`, config.C.MirrorlistFile, err)
	}

	log.Printf(`Initing package cache in "%s"`, config.C.CacheDirectory)
	c, err := cache.New(config.C.CacheDirectory, m)
	if err != nil {
		log.Fatalf(`Error initing cache "%s": %v`, config.C.CacheDirectory, err)
	}

	s := server.New(c)
	log.Println("Listening on", config.C.Listen)
	err = http.ListenAndServe(config.C.Listen, s)
	if err != nil {
		log.Fatalf("Error listening on %s: %v", config.C.Listen, err)
	}
}
