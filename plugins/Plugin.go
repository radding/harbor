package plugins

import (
	"encoding/json"
	"strings"

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
	logEntry := &LogEntry{}
	err := json.Unmarshal(b, logEntry)
	if err != nil {
		log.Error().Err(err).Msg("an error occured trying to JSON serialize message")
		return 0, err
	}
	event := d.getLoggerWithLevel(logEntry.Level)
	if logEntry.LogSchemaVersion != nil {
		event.Str("Identifier", logEntry.Identifier)
		event.Msg(strings.Trim(logEntry.Message, "\n"))
	}
	// logMsg := map[string]interface{}{}
	// if err := json.Unmarshal([]byte(val.Message), &logMsg); err != nil {
	// 	d.getLoggerWithLevel(val.Level).Str("Identifier", val.Identifier).Msg(val.Message)
	// } else {
	// 	lvl := logMsg["level"].(string)
	// 	msg := logMsg["message"].(string)
	// 	event := d.getLoggerWithLevel(lvl)
	// 	for key, val := range logMsg {
	// 		event = event.Any(key, val)
	// 	}
	// 	event.Msg(msg)
	// }
	return len(b), nil
}
