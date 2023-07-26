package cmds

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/radding/harbor/internal/config"
	"github.com/radding/harbor/internal/workspaces"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type LogLevel zerolog.Level

func (l *LogLevel) String() string {
	switch *l {
	case LogLevel(zerolog.DebugLevel):
		return "debug"
	case LogLevel(zerolog.InfoLevel):
		return "info"
	case LogLevel(zerolog.PanicLevel):
		return "panic"
	case LogLevel(zerolog.FatalLevel):
		return "fatal"
	case LogLevel(zerolog.ErrorLevel):
		return "error"
	case LogLevel(zerolog.TraceLevel):
		return "trace"
	case LogLevel(zerolog.WarnLevel):
		return "warn"
	default:
		return "unknown"
	}
}

func (l *LogLevel) Set(val string) error {
	switch strings.ToLower(val) {
	case "info":
		*l = LogLevel(zerolog.InfoLevel)
	case "warn":
		*l = LogLevel(zerolog.WarnLevel)
	case "panic":
		*l = LogLevel(zerolog.PanicLevel)
	case "fatal":
		*l = LogLevel(zerolog.FatalLevel)
	case "error":
		*l = LogLevel(zerolog.ErrorLevel)
	case "debug":
		*l = LogLevel(zerolog.DebugLevel)
	case "trace":
		*l = LogLevel(zerolog.TraceLevel)
	default:
		return errors.New(fmt.Sprintf("unknown log level: %s", val))
	}
	return nil
}

func (l *LogLevel) Type() string {
	return "LogLevel"
}

func LogLevelPtr(logLevel LogLevel) *LogLevel {
	return &logLevel
}

var machineReadableLogs *bool
var logLevel *LogLevel = LogLevelPtr(LogLevel(zerolog.InfoLevel))

func init() {
	machineReadableLogs = rootCmd.PersistentFlags().BoolP("machine-readable", "m", false, "Produce machine readable JSON logs?")
	rootCmd.PersistentFlags().VarP(logLevel, "log-level", "v", "The Log level to set the logger to. Can be: Panic, Fatal, Error, Warn, Info, Debug, and Trace")
}

var rootCmd = &cobra.Command{
	Short: "Harbor is a tool to manage workspaces for projects",
	Long:  `Harbor is a workspace management and build tool that enables developers to manage their projects more effectively.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

		// plManager := plugins.New()
		// testPlugin, err := plManager.GetClient("C:\\Users\\raddi\\code\\harbor\\gitplugin\\plugin.exe")
		// if err != nil {
		// 	log.Fatal().Err(err).Msg("Can not initialize plugin!")
		// }

		// managerObs, err := testPlugin.Dispense("manager")
		// if err != nil {
		// 	log.Fatal().Err(err).Msg("Can not initialize plugin!")
		// }
		// manager := managerObs.(*plugins.ManagerClient)
		// canHandle, err := manager.CanHandle(&plugins.CanHandleRequest{
		// 	Url: "SomeURL",
		// })

		// log.Info().Msgf("can handle: %s with err %s", canHandle, err)

		zerolog.SetGlobalLevel(zerolog.Level(*logLevel))
		if !*machineReadableLogs {
			out := zerolog.ConsoleWriter{Out: os.Stdout}
			out.PartsOrder = []string{
				"Identifier",
				"time",
				"level",
				"message",
			}
			out.FieldsExclude = []string{
				"Identifier",
			}
			out.FormatFieldValue = func(i interface{}) string {
				if i == nil {
					return ""
				}
				return fmt.Sprintf("%s", i)
			}

			log.Logger = log.Output(out)
		}

		log.Trace().Msgf("starting logging with level: %s", logLevel.String())
		if _, err := workspaces.GetConfig(); err != nil {
			log.Fatal().Err(err).Msg("error getting config")
		}
		c := config.LoadConfig(".", os.ExpandEnv("${APPDATA}/harbor/"), os.ExpandEnv("${ProgramFiles}/harbor"), "/etc/harbor/", os.ExpandEnv("${HOMEPATH}/.harbor"), os.ExpandEnv("${HOME}/.harbor"))
		err := c.Save()
		if err != err {
			log.Warn().Err(err).Msg("error saving configuration. This is fine, but could impact performance this time around")
		}
		log.Trace().Msgf("Loading plugins")
		err = c.LoadPlugins()
		if err != err {
			log.Fatal().Err(err).Msg("error saving configuration. This is fine, but could impact performance this time around")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		config.Get().KillAllPlugins()
		os.Exit(1)
	}
	config.Get().KillAllPlugins()
}
