package plugins

import (
	"context"
	"os"
	"os/exec"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"github.com/radding/harbor-plugins/proto"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

type PluginDefinition proto.PluginDefinition

type CacheItem struct {
	LogItem      string
	ArtifactPath string
}

type PluginClient interface {
	Run(RunRequest, ...CallOption) (ClientTask, error)
	Install() (*PluginDefinition, error)
	GetCacheKey(string, []string, []string) (string, error)
	Cache(string, string, chan CacheItem) error
	ReplayCache(string, string) (chan CacheItem, bool, error)
	Kill()
}

type pluginClient struct {
	plugin.Plugin
	managerClient proto.ManagerClient
	runnerClient  proto.RunnerClient
	installClient proto.InstallerClient
	cacheClient   proto.CacherClient
	clientImpl    *plugin.Client

	logger *LogBroker
}

func (p *pluginClient) Kill() {
	p.clientImpl.Kill()
}

func NewClient(pluginLocation string, logger zerolog.Logger) (PluginClient, error) {
	internalLogger := &LogBroker{
		logger:       logger,
		logCapturers: sync.Map{},
	}
	p := &pluginClient{}

	hclLogger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     internalLogger,
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
		SyncStdout:       os.Stdout,
		SyncStderr:       os.Stderr,
	})
	p.clientImpl = client
	cli, err := client.Client()
	if err != nil {
		return nil, errors.Wrap(err, "can't get plugin client")
	}

	impl, err := cli.Dispense("client")
	if err != nil {
		return nil, errors.Wrap(err, "can't dispense client")
	}
	client2 := impl.(*pluginClient)
	client2.clientImpl = client
	client2.logger = internalLogger
	return client2, nil
}

func (p *pluginClient) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	// proto.Register
	return errors.New("attempting to start a server from client implementation")
}

func (p *pluginClient) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &pluginClient{
		managerClient: proto.NewManagerClient(c),
		runnerClient:  proto.NewRunnerClient(c),
		installClient: proto.NewInstallerClient(c),
		cacheClient:   proto.NewCacherClient(c),
	}, nil
}

func (p *pluginClient) Install() (*PluginDefinition, error) {
	_resp, err := p.installClient.InstallPlugin(context.Background(), &proto.InstallRequest{})
	return (*PluginDefinition)(_resp), err
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
