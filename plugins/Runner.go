package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"

	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/radding/harbor-plugins/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

type RegisterRunnerResponse proto.RegisterRunnerResponse
type RunRequest proto.RunRequest
type RunResponse proto.RunResponse

func YamlToStruct(yml map[string]interface{}) *_struct.Struct {
	json, _ := json.Marshal(yml)
	st := &_struct.Struct{}
	protojson.Unmarshal(json, st)
	return st
}

type Streamer struct {
	buf        *bytes.Buffer
	ctx        context.Context
	cancelFunc context.CancelFunc
	lock       *sync.Mutex
}

func (s *Streamer) Write(p []byte) (n int, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.buf.Write(p)
}

func (s *Streamer) Close() error {
	s.cancelFunc()
	return nil
}

func (s *Streamer) IsDone() bool {
	err := s.ctx.Err()
	return err != nil
}

func (s *Streamer) HasData() bool {
	return s.buf.Len() > 0
}

func (s *Streamer) Read(b []byte) (int, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.buf.Read(b)
}

func NewStreamer() *Streamer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Streamer{
		ctx:        ctx,
		cancelFunc: cancel,
		buf:        &bytes.Buffer{},
		lock:       &sync.Mutex{},
	}
}

type TaskRunner interface {
	RunTask(r RunRequest, ctx context.Context) (RunResponse, error)
}

type TaskRunnerFunc func(RunRequest, context.Context) (RunResponse, error)

func (t TaskRunnerFunc) RunTask(r RunRequest, ctx context.Context) (RunResponse, error) {
	return t(r, ctx)
}

type DefaultRunner struct {
}

func (d *DefaultRunner) Register() (RegisterRunnerResponse, error) {
	return RegisterRunnerResponse{
		RunnerName: "",
	}, nil
}

func (d *DefaultRunner) Run(r RunRequest) (RunResponse, error) {
	return RunResponse{
		ExitCode: 0,
	}, nil
}
