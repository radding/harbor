package plugins

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"github.com/radding/harbor-plugins/proto"
	"google.golang.org/grpc"
)

type NotSupportedError struct {
	name       string
	pluginType string
}

func (n NotSupportedError) Error() string {
	return fmt.Sprintf("plugin %s does not support %s", n.name, n.pluginType)
}

func newNotSupportedError(name, pluginType string) error {
	return NotSupportedError{
		name:       name,
		pluginType: pluginType,
	}
}

type PluginProvider interface {
	WithManager(ManagerPlugin) PluginProvider
	WithTaskRunner(string, TaskRunner) PluginProvider
	WithLogger(logger hclog.Logger) PluginProvider
	WithCacheProvider(CacheProvider) PluginProvider
	ServePlugin()
}

type pluginProvider struct {
	plugin.Plugin
	proto.UnimplementedRunnerServer
	proto.UnimplementedInstallerServer
	proto.UnimplementedCacherServer
	runnerImpl     TaskRunner
	managerImpl    ManagerPlugin
	cachProvider   CacheProvider
	name           string
	logger         hclog.Logger
	runnerSettings struct {
		typeName string
	}
}

func (p *pluginProvider) wrapContext(ctx context.Context, ident string) context.Context {
	newCtx := context.WithValue(ctx, "Logger", p.logger.With("Identifier", ident))

	return newCtx
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

func (p *pluginProvider) WithCacheProvider(cacheProvider CacheProvider) PluginProvider {
	p.cachProvider = cacheProvider
	return p
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
	proto.RegisterCacherServer(s, p)
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
