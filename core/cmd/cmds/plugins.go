package cmds

import (
	"github.com/radding/harbor/internal/plugins"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.AddCommand(plugins.ListPlugins)
	pluginCmd.AddCommand(plugins.InstallPlugins)
}

var pluginCmd = &cobra.Command{
	Use:     "plugins",
	Aliases: []string{"p"},
	Short:   "Manage harbor plugins",
	Long:    "Install, remove, and list plugins",
}
