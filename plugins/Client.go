package plugins

import (
	"context"
	"errors"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/radding/harbor-plugins/proto"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

type PluginClient interface {
	Run(RunRequest) (RunResponse, error)
	GetClient(pluginLocation string, logger zerolog.Logger) (plugin.ClientProtocol, error)
}

type pluginClient struct {
	plugin.Plugin
	managerClient proto.ManagerClient
	runnerClient  proto.RunnerClient
}

func NewClient() PluginClient {
	return &pluginClient{}
}

func (p *pluginClient) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	// proto.Register
	return errors.New("attempting to start a server from client implementation")
}

func (p *pluginClient) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &pluginClient{
		managerClient: proto.NewManagerClient(c),
		runnerClient:  proto.NewRunnerClient(c),
	}, nil
}

func (p *pluginClient) CanHandle(req CanHandleRequest) (bool, error) {
	_req := proto.CanHandleMessage(req)
	resp, err := p.managerClient.CanHandle(context.Background(), &_req)
	if resp == nil {
		return false, err
	}
	return resp.CanHandle, err
}

func (p *pluginClient) Clone(req CloneRequest) (string, error) {
	_req := proto.CloneMessage(req)

	resp, err := p.managerClient.Clone(context.Background(), &_req)
	if resp == nil {
		return "", err
	}
	return resp.Destination, err
}

func runRequestPtr(r RunRequest) *proto.RunRequest {
	v := proto.RunRequest(r)
	return &v
}

func (p *pluginClient) Run(r RunRequest) (RunResponse, error) {
	_req := runRequestPtr(r)
	resp, err := p.runnerClient.Run(context.Background(), _req)
	if err != nil {
		return RunResponse{}, err
	}
	return RunResponse(*resp), nil
}

func (p *pluginClient) GetClient(pluginLocation string, logger zerolog.Logger) (plugin.ClientProtocol, error) {
	hclLogger := hclog.New(&hclog.LoggerOptions{
		Level: hclog.Trace,
		Output: &dumbLogWrapper{
			logger: logger,
		},
		JSONFormat: true,
	})
	plugins := map[string]plugin.Plugin{}
	plugins["client"] = p
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  HandShake,
		Plugins:          plugins,
		Cmd:              exec.Command(pluginLocation),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           hclLogger,
	})
	return client.Client()
}
