package runners

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/radding/harbor/internal/config"
	"github.com/radding/harbor/internal/workspaces"
	"github.com/rs/zerolog/log"
)

func RunCommand(command string, args []string) error {
	rootConf, err := workspaces.GetConfig()

	if err != nil {
		return errors.Wrap(err, "error getting workspace config")
	}
	log.Trace().Msgf("Getting recipe for %s", command)
	runStep, err := getRootRecipe(command, rootConf)
	if err != nil {
		return errors.Wrap(err, "Can't get root recipe")
	}
	globalContext, globalCancel := context.WithCancel(context.Background())
	err = runStep.Run(args, config.Get().GetPlugin, runContext{
		cancelCtx:  globalContext,
		cancelFunc: globalCancel,
	})

	return err
}

func getRootRecipe(command string, rootConfig workspaces.WorkspaceConfig) (*RunRecipe, error) {
	recipeGraph := map[string]*RunRecipe{}
	runStep := &RunRecipe{
		Pkg:         rootConfig.Name,
		CommandName: command,
		Needs:       []*RunRecipe{},
		wg:          &sync.WaitGroup{},
		pkgObject:   rootConfig,
	}
	recipeGraph[runStep.HashKey()] = runStep
	cycleTracker := make(visitedSet)

	var getDependencies func(command string, pkgConfig workspaces.WorkspaceConfig) (*RunRecipe, error)
	getDependencies = func(command string, pkgConfig workspaces.WorkspaceConfig) (*RunRecipe, error) {
		key := fmt.Sprintf("%s:%s", pkgConfig.Name, command)
		runStep, ok := recipeGraph[key]
		if !ok {
			cmd, ok := pkgConfig.Commands[command]
			if !ok {
				return nil, fmt.Errorf("command with name %s not found in pkg %s", command, pkgConfig.Name)
			}
			runStep = &RunRecipe{
				Pkg:         pkgConfig.Name,
				CommandName: command,
				Needs:       []*RunRecipe{},
				wg:          &sync.WaitGroup{},
				runConfig:   &cmd,
				pkgObject:   pkgConfig,
			}
			recipeGraph[runStep.HashKey()] = runStep
		}
		if cycleTracker.Has(runStep) {
			runSteps := []string{}
			for key := range cycleTracker {
				runSteps = append(runSteps, key)
			}
			return nil, errors.Errorf("error: cycle detected %s", runSteps)
		}
		cycleTracker.Add(runStep)
		defer cycleTracker.Remove(runStep)

		cmd := pkgConfig.Commands[command]
		for _, dep := range cmd.Dependencies {
			if dep.PackageName == "." {
				dep.PackageName = pkgConfig.Name
			}
			conf, err := rootConfig.GetPackageConfig(dep.PackageName)
			if err != nil {
				return runStep, errors.Wrapf(err, "error getting subpackage config for %s", dep.PackageName)
			}
			depRecipe, err := getDependencies(dep.CommandName, conf)
			if err != nil {
				return runStep, errors.Wrapf(err, "can't build recipe for %s", dep.PackageName)
			}
			runStep.Needs = append(runStep.Needs, depRecipe)

		}

		return runStep, nil
	}

	if cmd, ok := rootConfig.Commands[command]; ok && len(cmd.Dependencies) > 0 {
		rootCmd, err := getDependencies(command, rootConfig)
		return rootCmd, err
	} else {
		for _, conf := range rootConfig.GetAllSubPackages() {
			_, ok := conf.Commands[command]
			if !ok {
				log.Trace().Msgf("package %s does not have command %s", conf.Name, command)
				continue
			}
			depRecipe, err := getDependencies(command, conf)
			if err != nil {
				return runStep, errors.Wrapf(err, "can't get recipe for command %s in package %s", command, conf.Name)
			}
			runStep.Needs = append(runStep.Needs, depRecipe)
		}
		if len(runStep.Needs) == 0 {
			return runStep, fmt.Errorf("no command named %q", command)
		}
		return runStep, nil
	}

}

type commandNotFoundErr struct {
	pkg     string
	command string
}

func (c commandNotFoundErr) Error() string {
	return fmt.Sprintf("can't find command %s in package %s", c.command, c.pkg)
}

func IsCommandNotFoundError(err error) bool {
	_, ok := err.(commandNotFoundErr)
	return ok
}
