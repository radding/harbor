package runners

import (
	"fmt"
	"sync"

	plugins "github.com/radding/harbor-plugins"
	"github.com/radding/harbor/internal/config"
	"github.com/radding/harbor/internal/workspaces"
	"github.com/rs/zerolog/log"
)

type Hasher interface {
	HashKey() string
}

type visitedSet map[string]Hasher

func (v *visitedSet) Has(h Hasher) bool {
	_, ok := (*v)[h.HashKey()]
	return ok
}

func (v *visitedSet) Add(h Hasher) {
	(*v)[h.HashKey()] = h
}

func (v *visitedSet) Remove(h Hasher) {
	delete((*v), h.HashKey())
}

type RunRecipe struct {
	Pkg         string
	CommandName string
	Needs       []*RunRecipe
	wg          *sync.WaitGroup
	runConfig   *workspaces.Command
	done        bool
	err         error
	pkgObject   workspaces.WorkspaceConfig
}

func (r RunRecipe) Eq(r2 RunRecipe) bool {
	return r.Pkg == r2.Pkg && r.CommandName == r2.CommandName
}

func (r RunRecipe) HashKey() string {
	return fmt.Sprintf("%s:%s", r.Pkg, r.CommandName)
}

func (r *RunRecipe) Run(args []string) error {
	// logger :=
	r.wg.Wait()
	if r.done {
		log.Trace().Str("Identifier", r.HashKey()).Msg("Step has been run, skipping")
		return r.err
	}
	r.wg.Add(1)
	defer r.wg.Done()

	log.Trace().Str("Identifier", r.HashKey()).Msg("starting to run")

	wg := sync.WaitGroup{}
	for _, dep := range r.Needs {
		log.Trace().Str("Identifier", r.HashKey()).Msgf("Waiting for dependency %s", dep.HashKey())
		wg.Add(1)
		go func(dep *RunRecipe) {
			defer wg.Done()
			r.err = dep.Run(args)
			if r.err != nil {
				log.Error().Err(r.err).Msgf("Failed to run %s got %s", dep.CommandName, r.err)
			}
		}(dep)
	}
	wg.Wait()
	if r.err != nil {
		return r.err
	}
	log.Trace().Str("Identifier", r.HashKey()).Str("Path", r.pkgObject.WorkspaceRoot()).Msg("Actually Running")
	if r.runConfig == nil {
		r.done = true
		return r.err
	}
	runner, err := config.Get().GetPlugin(r.runConfig.Type)
	if err != nil {
		r.err = err
		return err
	}
	var status plugins.RunResponse
	status, r.err = runner.Run(plugins.RunRequest{
		RunCommand:     r.runConfig.Command,
		Args:           []string{},
		Path:           r.pkgObject.WorkspaceRoot(),
		PackageName:    r.Pkg,
		CommandName:    r.CommandName,
		Settings:       plugins.YamlToStruct(r.runConfig.Settings),
		StepIdentifier: r.HashKey(),
	})
	if status.ExitCode != 0 {
		r.err = fmt.Errorf("recieved non-zero exit code (%d) from %s", status.ExitCode, r.HashKey())
	}
	r.done = true
	return r.err
}
