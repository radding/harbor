package plugins

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/google/uuid"
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

type LogEntry struct {
	Level            string  `json:"@level"`
	Message          string  `json:"@message"`
	Module           string  `json:"@module"`
	Timestamp        string  `json:"@timestamp"`
	Identifier       string  `json:"@identifier"`
	LogSchemaVersion *string `json:"@log_schema_version"`
	PluginName       string  `json:"@plugin_name"`
}

type LogEventCapturer interface {
	Capture(*LogEntry)
}

type LogBroker struct {
	logger       zerolog.Logger
	logCapturers sync.Map
}

func NewLogBroker(logger zerolog.Logger) *LogBroker {
	return &LogBroker{
		logger:       logger,
		logCapturers: sync.Map{},
	}
}

func (d *LogBroker) getLoggerWithLevel(lvl string) *zerolog.Event {
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

func (d *LogBroker) Write(b []byte) (int, error) {
	logEntry := &LogEntry{}
	err := json.Unmarshal(b, logEntry)
	if err != nil {
		log.Error().Err(err).Msg("an error occured trying to JSON serialize message")
		return 0, err
	}
	d.logCapturers.Range(func(_, value any) bool {
		capturer := value.(LogEventCapturer)
		capturer.Capture(logEntry)
		return true
	})
	event := d.getLoggerWithLevel(logEntry.Level)
	if logEntry.LogSchemaVersion != nil {
		event.Str("Identifier", logEntry.Identifier)
		event.Msg(strings.Trim(logEntry.Message, "\n"))
	}
	return len(b), nil
}

func (d *LogBroker) AddCapturer(capt LogEventCapturer) string {
	uid := uuid.NewString()
	d.logCapturers.Store(uid, capt)
	return uid
}

func (d *LogBroker) RemoveCapturer(uid string) {
	d.logCapturers.Delete(uid)
}
