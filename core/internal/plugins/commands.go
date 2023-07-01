package plugins

import (
	"github.com/radding/harbor/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var ListPlugins = &cobra.Command{
	Use:   "list",
	Short: "List all plugins currently installed",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Msg("Listing out plugins")
		for _, plugin := range config.Get().Plugins {
			log.Info().Msgf("PLUGIN %s", plugin.Name)
			log.Info().Msgf("\tInstall Path: %s", plugin.PluginLocation)
			log.Info().Msgf("\tSettings: %s", plugin.SettingsPath)
		}
	},
}

var InstallPlugins = &cobra.Command{
	Use:   "install",
	Short: "Install a plugin",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conf := config.Get()
		log.Info().Msgf("Installing %s to %s", args[0], conf.PluginsDir)
		plugin, err := InstallPlugin(args[0])
		if err != nil {
			log.Fatal().Msgf("failed to install plugin: %s", err)
		}
		config.Get().Plugins[plugin.Name] = plugin
		err = config.Get().Save()
		if err != nil {
			log.Fatal().Msgf("failed to save updated config: %s", err)
		}
		log.Info().Msg("successfully install plugin")

	},
}
