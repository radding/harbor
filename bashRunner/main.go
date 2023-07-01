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
	defer func() {
		if err := recover(); err != nil {
			log.Println(fmt.Sprintf("plugin panicked: %s", err))
		}
		log.Println("plugin is exiting!")
		logOut.Close()
	}()

	log.SetOutput(logOut)
	log.Println("Starting plugin")
	plugins.NewPlugin("bash").
		WithTaskRunner("bash", plugins.TaskRunnerFunc(run)).
		ServePlugin()
}
