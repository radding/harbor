package runners

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	plugins "github.com/radding/harbor-plugins"
	"github.com/radding/harbor-plugins/proto"
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

type runContext struct {
	cancelCtx  context.Context
	cancelFunc context.CancelFunc
}

func newRunContext() *runContext {
	ctx, cancel := context.WithCancel(context.Background())
	return &runContext{
		cancelCtx:  ctx,
		cancelFunc: cancel,
	}
}

func (r *runContext) Cancel(signal int64, timeoutMS int64) {
	r.cancelCtx = context.WithValue(r.cancelCtx, "signal", signal)
	r.cancelCtx = context.WithValue(r.cancelCtx, "timeout", timeoutMS)
	r.cancelFunc()
}

func (r *runContext) SignalAndTimeoutValue() (int64, int64) {
	val, ValOk := r.cancelCtx.Value("signal").(int64)
	if !ValOk {
		val = 0
	}
	timeout, timeoutOk := r.cancelCtx.Value("timeout").(int64)
	if !timeoutOk {
		timeout = 0
	}
	return val, timeout
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
	ctx         context.Context
	cancelFunc  context.CancelFunc
}

func (r RunRecipe) Eq(r2 RunRecipe) bool {
	return r.Pkg == r2.Pkg && r.CommandName == r2.CommandName
}

func (r RunRecipe) HashKey() string {
	return fmt.Sprintf("%s:%s", r.Pkg, r.CommandName)
}

type runnerFetcher func(name string) (plugins.PluginClient, error)

func (r *RunRecipe) Run(args []string, fetcher runnerFetcher, runCtx *runContext) error {
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
			if !errors.Is(runCtx.cancelCtx.Err(), context.Canceled) {
				err := dep.Run(args, fetcher, runCtx)
				if err != nil {
					if r.err == nil {
						r.err = err
					} else {
						r.err = errors.Wrap(r.err, err.Error())
					}
				}
			}
		}(dep)
	}
	wg.Wait()
	if errors.Is(runCtx.cancelCtx.Err(), context.Canceled) {
		if r.err != nil {
			return r.err
		}
		r.err = fmt.Errorf("run Context was canceled")
	}
	if r.err != nil {
		return r.err
	}
	log.Trace().Str("Identifier", r.HashKey()).Str("Path", r.pkgObject.WorkspaceRoot()).Msg("Actually Running")
	if r.runConfig == nil {
		r.done = true
		return r.err
	}
	runner, err := fetcher(r.runConfig.Type)
	if err != nil {
		r.err = err
		return err
	}
	var task plugins.ClientTask
	task, r.err = runner.Run(plugins.RunRequest{
		RunCommand:     r.runConfig.Command,
		Args:           args,
		Path:           r.pkgObject.WorkspaceRoot(),
		PackageName:    r.Pkg,
		CommandName:    r.CommandName,
		Settings:       plugins.YamlToStruct(r.runConfig.Settings),
		StepIdentifier: r.HashKey(),
	})
	if r.err != nil {
		return r.err
	}
	done := make(chan struct{})

	go func() {
		task.Wait()
		done <- struct{}{}
	}()

	select {
	case <-runCtx.cancelCtx.Done():
		log.Trace().Msgf("Caught cancel message, canceling")
		signal, timeoutMs := runCtx.SignalAndTimeoutValue()
		if signal == 0 {
			signal = 2
		}
		task.Stop(signal, timeoutMs)
		r.err = fmt.Errorf("global run context was canceled, canceling my tasks")
	case <-done:
		stats := task.Status()
		log.Debug().Msgf("task result: {status = %s, exitcode = %d, time elapsed = %d", stats.Status, stats.ExitCode, stats.TimeElapsed)
		if stats.Status == proto.RunStatus_CRASHED {
			r.err = fmt.Errorf("task failed with exit code: %d", stats.ExitCode)
			runCtx.Cancel(9, 0)
		}
	}

	r.done = true
	return r.err
}
