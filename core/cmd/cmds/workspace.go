package cmds

import (
	"github.com/radding/harbor/internal/workspaces"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var initDir *string

func init() {
	rootCmd.AddCommand(workspaceCMD)
	workspaceCMD.AddCommand(initWsCMD)
	initDir = initWsCMD.Flags().StringP("dir", "d", "", "Specify a directory to start a workspace in, defaults to [name]")
}

var workspaceCMD = &cobra.Command{
	Use:     "workspace",
	Short:   "manipulates a harbour workspace",
	Long:    "Workspace manipulates harbor workspaces, adding packages, managing building, and more",
	Aliases: []string{"ws"},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var initWsCMD = &cobra.Command{
	Use:   "init",
	Short: "create a new workspace at the specified directory",
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		if len(*initDir) == 0 {
			initDir = &args[0]
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info().Msg("Initializing Workspace!")
		return workspaces.Initialize(args[0], *initDir, "workspace.conf")
	},
}
