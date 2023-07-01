package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	plugins "github.com/radding/harbor-plugins"
	"github.com/rs/zerolog/log"
)

type stream struct {
	onWrite func(bts []byte) (int, error)
}

func (s *stream) Write(bts []byte) (int, error) {
	return s.onWrite(bts)
}

func run(req plugins.RunRequest, ctx context.Context) (resp plugins.RunResponse, err error) {
	// logger := hclog.New(&hclog.LoggerOptions{
	// 	Name:       req.PackageName,
	// 	Level:      hclog.Trace,
	// 	JSONFormat: true,
	// }).With("identifier", req.StepIdentifier)
	logger := ctx.Value("Logger").(hclog.Logger)
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
		}
	}()
	body, _ := json.Marshal(req)

	logger.Debug(fmt.Sprintf("Running command %s", string(body)))
	curPWD, err := os.Getwd()
	if err != nil {
		log.Error().Msgf("failed to get PWD: %s", err)
		return plugins.RunResponse{
			ExitCode: -1,
		}, err
	}
	os.Chdir(req.Path)
	defer os.Chdir(curPWD)
	cmd := exec.Command("/bin/sh", "-c", req.RunCommand)
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr
	if err := cmd.Run(); err != nil {
		logger.Error(err.Error())
		return plugins.RunResponse{ExitCode: -1}, err
	}
	return plugins.RunResponse{
		ExitCode: int64(cmd.ProcessState.ExitCode()),
	}, nil
}
