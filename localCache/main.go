package main

import (
	"io"
	"log"
	"os"

	plugins "github.com/radding/harbor-plugins"
)

func openFile(fileName string) (io.ReadCloser, error) {
	return os.Open(fileName)
}

func main() {
	logOut, err := os.Create("./plugin.log")
	if err != nil {
		panic(err)
	}
	log.SetOutput(logOut)
	log.Println("Starting plugin")
	defer func() {
		if err := recover(); err != nil {
			log.Printf("plugin panicked: %s\n", err)
		}
		log.Println("plugin is exiting!")
		logOut.Close()
	}()
	plugins.NewPlugin("local_cache").
		WithCacheProvider(newCacher(openFile)).
		ServePlugin()
	log.Println("Done serving, exiting")
}
