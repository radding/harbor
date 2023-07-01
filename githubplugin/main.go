package main

import (
	"fmt"

	plugins "github.com/radding/harbor-plugins"
)

type Dummy struct{}

func (d *Dummy) CanHandle(req plugins.CanHandleRequest) (bool, error) {
	return false, nil
}

func (d *Dummy) Clone(req plugins.CloneRequest) (string, error) {
	return fmt.Sprintf("Cloning %s", req.Source), nil
}

func main() {
	plugins.New().
		WithManager(&Dummy{}).
		Serve()
}
