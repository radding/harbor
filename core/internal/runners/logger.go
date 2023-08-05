package runners

import (
	"encoding/json"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type logEntry struct {
	Level            string  `json:"@level"`
	Message          string  `json:"@message"`
	Module           string  `json:"@module"`
	Timestamp        string  `json:"@timestamp"`
	Identifier       string  `json:"@identifier"`
	LogSchemaVersion *string `json:"@log_schema_version"`
	PluginName       string  `json:"@plugin_name"`
}

type longEntry2 struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

func getEvent(level string) *zerolog.Event {
	switch level {
	case "trace":
		return log.Trace()
	case "debug":
		return log.Debug()
	case "info":
		return log.Info()
	case "warn":
		return log.Warn()
	case "error":
		return log.Error()
	default:
		log.Warn().Msgf("unsupported log level: %s", level)
		return log.Info()
	}
}

type replayer struct{}

func (r *replayer) Write(b []byte) (int, error) {
	entry := logEntry{}
	err := json.Unmarshal(b, &entry)
	if err != nil {
		return 0, err
	}
	if entry.LogSchemaVersion == nil {
		e2 := &longEntry2{}
		err := json.Unmarshal(b, e2)
		if err != nil {
			return 0, err
		}
		entry.Level = e2.Level
		entry.Message = e2.Message
	}
	event := getEvent(entry.Level)
	event.Str("Identifier", entry.Identifier)
	event.Msg(strings.Trim(entry.Message, "\n"))
	return len(b), nil
}
