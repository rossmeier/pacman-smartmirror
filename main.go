package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/veecue/pacman-smartmirror/cache"
	"github.com/veecue/pacman-smartmirror/mirrorlist"
	"github.com/veecue/pacman-smartmirror/server"
)

var (
	cacheDirectory = flag.String("d", "", "Directory to use for the cached packages")
	mirrorlistFile = flag.String("m", "", "Filename of the mirrorlist to use")
	listen         = flag.String("l", ":41234", "Address and port for the HTTP server to listen on")
)

func main() {
	flag.Parse()
	log.Printf(`Loading mirrorlist file: "%s"`, *mirrorlistFile)
	m, err := mirrorlist.FromFile(*mirrorlistFile)
	if err != nil {
		log.Fatalf(`Error reading mirrorlist "%s": %v`, *mirrorlistFile, err)
	}

	log.Printf(`Initing package cache in "%s"`, *cacheDirectory)
	c, err := cache.New(*cacheDirectory, m)
	if err != nil {
		log.Fatalf(`Error initing cache "%s": %v`, *cacheDirectory, err)
	}

	s := server.New(c)
	log.Println("Listening on", *listen)
	err = http.ListenAndServe(*listen, s)
	if err != nil {
		log.Fatalf("Error listening on %s: %v", *listen, err)
	}
}
