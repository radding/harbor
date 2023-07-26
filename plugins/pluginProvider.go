package plugins

import (
	"context"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
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
	hclLogger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		JSONFormat: true,
	})
	return &pluginProvider{
		name:           name,
		logger:         hclLogger.With("@plugin_name", name).With("@log_schema_version", "1.0.0"),
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
