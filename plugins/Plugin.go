package plugins

import (
	"encoding/json"

	"github.com/radding/harbor-plugins/proto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Plugin interface {
	proto.ManagerClient
}

type CanHandleRequest proto.CanHandleMessage
type CloneRequest proto.CloneMessage

type ManagerPlugin interface {
	CanHandle(CanHandleRequest) (bool, error)
	Clone(CloneRequest) (string, error)
}

// type gRPCManagerServer struct {
// 	plugin.Plugin
// 	managerImpl ManagerPlugin
// 	runnerImpl  TaskRunner
// }

// func (g *gRPCManagerServer) createPluginMap() plugin.PluginSet {
// 	pluginMap := map[string]plugin.Plugin{}
// 	pluginMap["client"] = g
// 	return pluginMap
// }

// func (g *gRPCManagerServer) WithManager(manager ManagerPlugin) PluginBuilder {
// 	g.managerImpl = manager
// 	return g
// }

// func (g *gRPCManagerServer) WithRunner(runner TaskRunner) PluginBuilder {
// 	g.runnerImpl = runner
// 	return g
// }

type LogEntry struct {
	Level      string `json:"@level"`
	Message    string `json:"@message"`
	Module     string `json:"@module"`
	Timestamp  string `json:"@timestamp"`
	Identifier string `json:"identifier"`
}

type dumbLogWrapper struct {
	logger zerolog.Logger
}

func (d *dumbLogWrapper) getLoggerWithLevel(lvl string) *zerolog.Event {
	switch lvl {
	case "trace":
		return d.logger.Trace()
	case "debug":
		return d.logger.Debug()
	case "info":
		return d.logger.Info()
	case "warn":
		return d.logger.Warn()
	case "error":
		return d.logger.Error()
	default:
		d.logger.Warn().Msgf("unsupported log level: %s", lvl)
		return d.logger.Info()
	}
}

func (d *dumbLogWrapper) Write(b []byte) (int, error) {
	val := &LogEntry{}
	err := json.Unmarshal(b, val)
	if err != nil {
		log.Error().Err(err).Msg("an error occured trying to JSON serialize message")
		return 0, err
	}
	logMsg := map[string]interface{}{}
	if err := json.Unmarshal([]byte(val.Message), &logMsg); err != nil {
		d.getLoggerWithLevel(val.Level).Str("Identifier", val.Identifier).Msg(val.Message)
	} else {
		lvl := logMsg["level"].(string)
		msg := logMsg["message"].(string)
		event := d.getLoggerWithLevel(lvl)
		for key, val := range logMsg {
			event = event.Any(key, val)
		}
		event.Msg(msg)
	}
	return len(b), nil
}

// Client implements PluginBuilder
// func (g *gRPCManagerServer) GetClient(pluginLocation string, logger zerolog.Logger) (plugin.ClientProtocol, error) {
// 	hclLogger := hclog.New(&hclog.LoggerOptions{
// 		Level: hclog.Trace,
// 		Output: &dumbLogWrapper{
// 			logger: logger,
// 		},
// 		JSONFormat: true,
// 	})
// 	client := plugin.NewClient(&plugin.ClientConfig{
// 		HandshakeConfig:  HandShake,
// 		Plugins:          g.createPluginMap(),
// 		Cmd:              exec.Command(pluginLocation),
// 		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
// 		Logger:           hclLogger,
// 	})

// 	return client.Client()
// }

// func (p *gRPCManagerServer) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
// 	proto.RegisterManagerServer(s, p)
// 	proto.RegisterRunnerServer(s, p)
// 	// proto.Register
// 	return nil
// }

// func (p *gRPCManagerServer) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
// 	return &PluginClient{
// 		managerClient: proto.NewManagerClient(c),
// 		runnerClient:  proto.NewRunnerClient(c),
// 	}, nil
// }

// func (g *gRPCManagerServer) CanHandle(ctx context.Context, req *proto.CanHandleMessage) (*proto.CanHandleResponse, error) {
// 	msg, err := g.managerImpl.CanHandle(CanHandleRequest(*req))
// 	return &proto.CanHandleResponse{
// 		CanHandle: msg,
// 	}, err
// }

// func (g *gRPCManagerServer) Clone(ctx context.Context, req *proto.CloneMessage) (*proto.CloneResponse, error) {
// 	resp, err := g.managerImpl.Clone(CloneRequest(*req))
// 	errMsg := ""
// 	statusCode := 0
// 	if err != nil {
// 		errMsg = err.Error()
// 		statusCode = -1
// 	}
// 	return &proto.CloneResponse{
// 		Destination:  resp,
// 		WasSuccess:   err == nil,
// 		ErrorMessage: errMsg,
// 		ErrorCode:    int64(statusCode),
// 	}, nil
// }

// // func (g *gRPCManagerServer) Register(ctx context.Context, req *proto.RegisterRunnerRequest) (*proto.RegisterRunnerResponse, error) {
// // 	resp, err := g.runnerImpl.Register()
// // 	if err != nil {
// // 		return nil, err
// // 	}
// // 	return &proto.RegisterRunnerResponse{
// // 		RunnerName: resp.RunnerName,
// // 	}, nil
// // }

// func (g *gRPCManagerServer) Run(ctx context.Context, req *proto.RunRequest) (*proto.RunResponse, error) {
// 	responce, err := g.runnerImpl.RunTask(RunRequest(*req))
// 	return &proto.RunResponse{
// 		ExitCode: responce.ExitCode,
// 	}, err
// }

// func New() PluginBuilder {
// 	return &gRPCManagerServer{
// 		managerImpl: &DefaulManagerServer{},
// 		runnerImpl:  &DefaultRunner{},
// 	}
// }

// func (g *gRPCManagerServer) Serve() error {
// 	log.Trace().Msg("Attempting to serve!")
// 	plugin.Serve(&plugin.ServeConfig{
// 		HandshakeConfig: HandShake,
// 		Plugins:         g.createPluginMap(),
// 		GRPCServer:      plugin.DefaultGRPCServer,
// 	})
// 	return nil
// }
