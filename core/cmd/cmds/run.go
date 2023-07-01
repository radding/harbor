package cmds

import (
	"github.com/radding/harbor/internal/runners"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a command in the workspace/project",
	Run: func(cmd *cobra.Command, args []string) {
		err := runners.RunCommand(args[0], args[1:])
		if err != nil {
			log.Error().Err(err).Msg("couldn't run command")
		}
	},
}
