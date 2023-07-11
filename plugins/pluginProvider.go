package plugins

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/radding/harbor-plugins/proto"
	"google.golang.org/grpc"
)

type PluginProvider interface {
	WithManager(ManagerPlugin) PluginProvider
	WithTaskRunner(string, TaskRunner) PluginProvider
	WithLogger(logger hclog.Logger) PluginProvider

	ServePlugin()
}

type pluginProvider struct {
	plugin.Plugin
	proto.UnimplementedRunnerServer
	proto.UnimplementedInstallerServer
	runnerImpl     TaskRunner
	managerImpl    ManagerPlugin
	name           string
	logger         hclog.Logger
	runnerSettings struct {
		typeName string
	}
}

func NewPlugin(name string) PluginProvider {
	return &pluginProvider{
		name:           name,
		logger:         hclog.Default().With("plugin_name", name),
		runnerSettings: struct{ typeName string }{},
	}
}

func (p *pluginProvider) WithManager(m ManagerPlugin) PluginProvider {
	p.managerImpl = m
	return p
}

func (p *pluginProvider) WithTaskRunner(typeName string, r TaskRunner) PluginProvider {
	p.runnerImpl = r
	p.runnerSettings.typeName = typeName
	return p
}

func (p *pluginProvider) WithLogger(logger hclog.Logger) PluginProvider {
	p.logger = logger
	return p
}

func (p *pluginProvider) ServePlugin() {
	p.logger.Trace("Attempting to serve!")

	pluginMap := map[string]plugin.Plugin{}
	pluginMap["client"] = p

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: HandShake,
		Plugins:         pluginMap,
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}

func (p *pluginProvider) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	// proto.RegisterManagerServer(s, p)
	proto.RegisterRunnerServer(s, p)
	proto.RegisterInstallerServer(s, p)
	// proto.Register
	return nil
}

func (p *pluginProvider) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return nil, errors.New("attempting to create client out of server implementation")
}

func (p *pluginProvider) Run(ctx context.Context, req *proto.RunRequest) (*proto.RunResponse, error) {
	if p.runnerImpl != nil {
		_req := RunRequest(*req)
		ctx = context.WithValue(ctx,
			"Logger",
			p.logger.With("Indentifier", req.StepIdentifier),
		)
		_resp, err := p.runnerImpl.RunTask(_req, ctx)
		resp := proto.RunResponse(_resp)
		return &resp, err
	}

	return &proto.RunResponse{}, fmt.Errorf("attempted to run, but plugin %q doesn't support runners", p.name)
}

func (p *pluginProvider) InstallPlugin(ctx context.Context, in *proto.InstallRequest) (*proto.PluginDefinition, error) {
	caps := []proto.PluginCapabilities{}
	if p.managerImpl != nil {
		caps = append(caps, proto.PluginCapabilities_DEPENDENCY_PROVIDER)
	}
	if p.runnerImpl != nil {
		caps = append(caps, proto.PluginCapabilities_TASK_RUNNER)
	}
	return &proto.PluginDefinition{
		Name:         p.name,
		Capabilities: caps,
	}, nil
}

// WithManager(ManagerPlugin) PluginBuilder
// WithRunner(Runner) PluginBuilder
