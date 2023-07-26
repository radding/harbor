package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"
	"github.com/radding/harbor-plugins/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

type RegisterRunnerResponse proto.RegisterRunnerResponse
type RunRequest proto.StartRequest
type RunResponse proto.RunResponse
type TaskStatus proto.RunStatus

const (
	RUNNING         TaskStatus = TaskStatus(proto.RunStatus_RUNNING)
	FINISHED        TaskStatus = TaskStatus(proto.RunStatus_FINISHED)
	CRASHED         TaskStatus = TaskStatus(proto.RunStatus_CRASHED)
	STARTING        TaskStatus = TaskStatus(proto.RunStatus_STARTING)
	CANCELED        TaskStatus = TaskStatus(proto.RunStatus_CANCELED)
	STARTREQUESTED  TaskStatus = 5
	CANCELREQUESTED TaskStatus = 6
)

func YamlToStruct(yml map[string]interface{}) *_struct.Struct {
	json, _ := json.Marshal(yml)
	st := &_struct.Struct{}
	protojson.Unmarshal(json, st)
	return st
}

type Task interface {
	Status() RunResponse
	Stop(signal int64, timeoutMS int64) error
}

type ClientTask interface {
	Wait() RunResponse
	Task
}

type clientTask struct {
	srv         proto.Runner_RunClient
	statusMutex *sync.Mutex
	lastStatus  proto.RunResponse
}

func (c *clientTask) Wait() RunResponse {
	for {
		c.statusMutex.Lock()
		status := c.lastStatus
		c.statusMutex.Unlock()
		switch status.Status {
		case proto.RunStatus_FINISHED, proto.RunStatus_CRASHED, proto.RunStatus_CANCELED:
			return c.Status()
		}
	}
}

func (c *clientTask) Status() RunResponse {
	c.statusMutex.Lock()
	defer c.statusMutex.Unlock()
	return RunResponse(c.lastStatus)
}

func (c *clientTask) watchSrv() {
	for {
		resp, _ := c.srv.Recv()
		c.statusMutex.Lock()
		c.lastStatus = *resp
		c.statusMutex.Unlock()
		switch c.lastStatus.Status {
		case proto.RunStatus_FINISHED, proto.RunStatus_CRASHED, proto.RunStatus_CANCELED:
			return
		}
	}
}

func (c *clientTask) Stop(signal int64, timeout int64) error {
	runReq := proto.CancelRequest{
		Signal:    signal,
		TimeoutMS: timeout,
	}
	renReq2 := proto.RunRequest_CancelRequest{
		CancelRequest: &runReq,
	}
	return c.srv.Send(&proto.RunRequest{
		Request: &renReq2,
	})
}

func newClientTask(srv proto.Runner_RunClient, runRequest RunRequest) ClientTask {
	task := &clientTask{
		srv:         srv,
		statusMutex: &sync.Mutex{},
	}

	runReq := proto.StartRequest(runRequest)
	renReq2 := proto.RunRequest_StartRequest{
		StartRequest: &runReq,
	}
	srv.Send(&proto.RunRequest{
		Request: &renReq2,
	})
	go task.watchSrv()
	return task
}

type RunnerCancelFunc func(signalCode int64, timeoutMS int64) error

type TaskRunner interface {
	Run(r RunRequest, ctx context.Context) (Task, error)
}

type TaskRunnerFunc func(RunRequest, context.Context) (Task, error)

func (t TaskRunnerFunc) Run(r RunRequest, ctx context.Context) (Task, error) {
	return t(r, ctx)
}

func runRequestPtr(r RunRequest) *proto.StartRequest {
	v := proto.StartRequest(r)
	return &v
}

// Client implementation of Runner
func (p *pluginClient) Run(r RunRequest) (ClientTask, error) {
	stream, err := p.runnerClient.Run(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "can't start streaming server")
	}
	task := newClientTask(stream, r)
	return task, nil
}

// Server implementation
func (p *pluginProvider) Run(serv proto.Runner_RunServer) error {
	if p.runnerImpl == nil {
		return errors.New("plugin does not support Run")
	}
	ctx := serv.Context()
	first, err := serv.Recv()
	if err != nil {
		return errors.Wrap(err, "error getting first command")
	}
	startReq := first.GetStartRequest()
	if startReq == nil {
		return fmt.Errorf("can't get start request")
	}
	ctx2 := context.WithValue(ctx, "Logger",
		p.logger.With("@Identifier", startReq.StepIdentifier),
	)
	t, err := p.runnerImpl.Run(RunRequest(*startReq), ctx2)
	serv.Send(&proto.RunResponse{
		Status:      proto.RunStatus_STARTING,
		ExitCode:    0,
		TimeElapsed: 0,
	})

	runChan := make(chan struct {
		req *proto.RunRequest
		err error
	})

	go func() {
		d, err := serv.Recv()
		if err != nil {
			runChan <- struct {
				req *proto.RunRequest
				err error
			}{
				req: nil,
				err: err,
			}
		}
		runChan <- struct {
			req *proto.RunRequest
			err error
		}{
			req: d,
			err: nil,
		}
	}()

	for {
		select {
		case <-ctx.Done():
			t.Stop(9, 0)
			return nil
		case req := <-runChan:
			d := req.req
			err := req.err
			if err != nil {
				return errors.Wrap(err, "error getting request")
			}
			if d.GetCancelRequest() == nil {
				return fmt.Errorf("unexpected start request")
			}
			cancel := d.GetCancelRequest()
			timeoutCtx, cancelFunc := context.WithTimeout(ctx, time.Duration(cancel.TimeoutMS+10))
			go func() {
				defer cancelFunc()
				t.Stop(cancel.Signal, cancel.TimeoutMS)
			}()
			<-timeoutCtx.Done()
			err = timeoutCtx.Err()
			if err != nil {
				p.logger.Error(fmt.Sprintf("error canceling: %s", err.Error()))
			}
		default:
			status := t.Status()
			resp := proto.RunResponse(status)
			serv.Send(&resp)
		}
	}
	// resp := proto.RunResponse(_resp)
}
