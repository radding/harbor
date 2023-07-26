package main

import (
	"fmt"
	"log"
	"os"

	plugins "github.com/radding/harbor-plugins"
)

func main() {
	logOut, err := os.Create("./plugin.log")
	if err != nil {
		panic(err)
	}
	log.SetOutput(logOut)
	log.Println("Starting plugin")
	defer func() {
		if err := recover(); err != nil {
			log.Println(fmt.Sprintf("plugin panicked: %s", err))
		}
		log.Println("plugin is exiting!")
		logOut.Close()
	}()
	plugins.NewPlugin("shell").
		WithTaskRunner("shell", plugins.TaskRunnerFunc(run)).
		ServePlugin()
	log.Println("Done serving, exiting")
}
