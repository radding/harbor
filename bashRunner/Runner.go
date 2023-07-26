package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	plugins "github.com/radding/harbor-plugins"
	"github.com/radding/harbor-plugins/proto"
)

type task struct {
	cmd         *exec.Cmd
	done        chan struct{}
	timeStarted time.Time
	wasCanceled bool
	logger      hclog.Logger
}

func (t *task) Status() plugins.RunResponse {
	now := time.Now()
	timeElapsed := now.Unix() - t.timeStarted.Unix()
	select {
	case <-t.done:
		// defer close(t.done)
		t.logger.Trace("Task has completed")
		status := proto.RunStatus_FINISHED
		exitCode := t.cmd.ProcessState.ExitCode()
		if exitCode != 0 && !t.wasCanceled {
			status = proto.RunStatus_CRASHED
		} else if exitCode != 0 && t.wasCanceled {
			status = proto.RunStatus_CANCELED
		}
		return plugins.RunResponse{
			Status:      status,
			ExitCode:    int64(exitCode),
			TimeElapsed: timeElapsed,
		}
	default:
		return plugins.RunResponse{
			Status:      proto.RunStatus_RUNNING,
			ExitCode:    0,
			TimeElapsed: timeElapsed,
		}

	}
}

func (t *task) start() {
	t.done = make(chan struct{})
	t.timeStarted = time.Now()
	err := t.cmd.Run()
	if err != nil {
		t.logger.Trace(fmt.Sprintf("error running cmd: %s", err.Error()))
	}
	t.done <- struct{}{}
}

func (t *task) Stop(signal int64, timeoutMS int64) error {
	t.wasCanceled = true
	return t.cmd.Process.Signal(syscall.Signal(signal))
}

type stream struct {
	onWrite func(bts []byte) (int, error)
}

func (s *stream) Write(bts []byte) (int, error) {
	return s.onWrite(bts)
}

func run(req plugins.RunRequest, ctx context.Context) (t plugins.Task, err error) {
	logger := ctx.Value("Logger").(hclog.Logger)
	logger.Trace("> running command")
	stdOut := &stream{
		onWrite: func(bts []byte) (int, error) {
			logger.Info(string(bts))
			return len(bts), nil
		},
	}

	stdErr := &stream{
		onWrite: func(bts []byte) (int, error) {
			logger.Error(string(bts))
			return len(bts), nil
		},
	}
	defer func() {
		if panicRec := recover(); panicRec != nil {
			err = fmt.Errorf("can't run command, panicked: %s", panicRec)
			logger.Error(fmt.Sprintf("error in running: %s", err.Error()))
		}
	}()
	body, _ := json.Marshal(req)

	logger.Debug(fmt.Sprintf("Running command %s", string(body)))
	curPWD, err := os.Getwd()
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get PWD: %s", err))
		return nil, err
	}
	os.Chdir(req.Path)
	defer os.Chdir(curPWD)
	shellFunc := fmt.Sprintf(`
	anon() {
		%s
	}
	anon %s`, req.RunCommand, strings.Join(req.Args, " "))
	logger.Debug(fmt.Sprintf("executing %s", shellFunc))
	cmd := exec.Command("/bin/bash", "-ce", shellFunc)
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr
	t2 := &task{
		cmd:    cmd,
		logger: logger,
	}

	go t2.start()
	t = t2
	return

}
