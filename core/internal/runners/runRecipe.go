package runners

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	plugins "github.com/radding/harbor-plugins"
	"github.com/radding/harbor-plugins/proto"
	"github.com/radding/harbor/internal/workspaces"
	"github.com/rs/zerolog"
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
	cacher     Cacher
}

func newRunContext(cacher Cacher) *runContext {
	ctx, cancel := context.WithCancel(context.Background())
	return &runContext{
		cancelCtx:  ctx,
		cancelFunc: cancel,
		cacher:     cacher,
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
	runConfig   *workspaces.Command
	done        bool
	err         error
	pkgObject   workspaces.WorkspaceConfig
	lock        *sync.Mutex
}

func (r RunRecipe) Eq(r2 RunRecipe) bool {
	return r.Pkg == r2.Pkg && r.CommandName == r2.CommandName
}

func (r RunRecipe) HashKey() string {
	return fmt.Sprintf("%s:%s", r.Pkg, r.CommandName)
}

type runnerFetcher func(name string) (plugins.PluginClient, error)

func (r *RunRecipe) Run(args []string, fetcher runnerFetcher, runCtx *runContext) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.runConfig != nil {
		shouldRun := true
		for _, cond := range r.runConfig.RunConditions {
			res, err := cond.Expr.Evaluate(r.pkgObject.VariableLookUpService())
			if err != nil {
				log.Warn().Str("Identifier", r.HashKey()).Msg("failed to evaluate run conditions, assuming true")
				continue
			}
			shouldRun = shouldRun && res
		}
		if !shouldRun {
			r.done = true
			log.Info().Str("Identifier", r.HashKey()).Msg("Run conditions evaluated to false, skipping")
			return nil
		}

	}
	if r.done {
		log.Trace().Str("Identifier", r.HashKey()).Msg("Step has been run, skipping")
		return r.err
	}

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
	cacheKey, err := runCtx.cacher.CalculateCacheKey(r)
	if err != nil {
		return errors.Wrap(err, "can't get cache key")
	}
	fromCache, err := runCtx.cacher.ReplayCachedLogs(cacheKey, &replayer{})
	if err != nil {
		log.Warn().Err(err).Msg("error retrieving from cache, just redoing it")
	}
	if !fromCache {
		log.Debug().Msgf("%s was not cached, performing it now", r.HashKey())
		buf := bytes.NewBuffer([]byte{})
		logger := zerolog.New(buf)
		log.Trace().Str("Identifier", r.HashKey()).Str("Path", r.pkgObject.WorkspaceRoot()).Msg("Actually Running")
		logger.Trace().Str("Identifier", r.HashKey()).Str("Path", r.pkgObject.WorkspaceRoot()).Msg("Actually Running")
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
		logger.Info().Msgf("Starting command %s", r.HashKey())
		log.Info().Msgf("Starting command %s", r.HashKey())
		task, r.err = runner.Run(plugins.RunRequest{
			RunCommand:     r.runConfig.Command,
			Args:           args,
			Path:           r.pkgObject.WorkspaceRoot(),
			PackageName:    r.Pkg,
			CommandName:    r.CommandName,
			Settings:       plugins.YamlToStruct(r.runConfig.Settings),
			StepIdentifier: r.HashKey(),
		}, plugins.WithLogCapture(buf, r.HashKey()))
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
			logger.Trace().Msgf("Caught cancel message, canceling")
			log.Trace().Msgf("Caught cancel message, canceling")
			signal, timeoutMs := runCtx.SignalAndTimeoutValue()
			if signal == 0 {
				signal = 2
			}
			task.Stop(signal, timeoutMs)
			r.err = fmt.Errorf("global run context was canceled, canceling my tasks")
		case <-done:
			stats := task.Status()
			logger.Debug().Msgf("task result: {status = %s, exitcode = %d, time elapsed = %d", stats.Status, stats.ExitCode, stats.TimeElapsed)
			log.Debug().Msgf("task result: {status = %s, exitcode = %d, time elapsed = %d", stats.Status, stats.ExitCode, stats.TimeElapsed)
			if stats.Status == proto.RunStatus_CRASHED {
				r.err = fmt.Errorf("task failed with exit code: %d", stats.ExitCode)
				runCtx.Cancel(9, 0)
			}
		}
		r.done = true
		logger.Info().Msgf("%s finished", r.HashKey())
		log.Info().Msgf("%s finished", r.HashKey())
		err = runCtx.cacher.WriteLogsToCache(cacheKey, buf)
		if err != nil {
			logger.Error().Err(err).Msgf("failed to cache %s", r.HashKey())
			log.Error().Err(err).Msgf("failed to cache %s", r.HashKey())
			r.err = err
		}
	} else {
		log.Debug().Msgf("%s was cached, replaying it now", r.HashKey())
	}
	return r.err
}
