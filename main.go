package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/veecue/pacman-smartmirror/cache"
	"github.com/veecue/pacman-smartmirror/config"
	"github.com/veecue/pacman-smartmirror/server"

	_ "github.com/veecue/pacman-smartmirror/impl/apk"
	_ "github.com/veecue/pacman-smartmirror/impl/pacman"
)

func main() {
	flag.Parse()
	config.Init()

	log.Printf(`Initing package cache in "%s"`, config.C.CacheDirectory)
	c, err := cache.New(config.C.CacheDirectory, config.C.Repos)
	if err != nil {
		log.Fatalf(`Error initing cache "%s": %v`, config.C.CacheDirectory, err)
	}
	c.UpdateDatabases(nil)
	go func() {
		res := make(chan error)
		for range time.Tick(20 * time.Minute) {
			c.UpdateDatabases(res)
			<-res
		}
	}()

	s := server.New(c)
	log.Println("Listening on", config.C.Listen)
	err = http.ListenAndServe(config.C.Listen, s)
	if err != nil {
		log.Fatalf("Error listening on %s: %v", config.C.Listen, err)
	}
}
