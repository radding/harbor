package plugins

import (
	"github.com/hashicorp/go-plugin"
	"github.com/radding/harbor/internal/config"
	"github.com/rs/zerolog/log"
)

func LoadPlugins() (plugin.PluginSet, error) {
	// var handshakeConfig = plugin.HandshakeConfig{
	// 	ProtocolVersion:  1,
	// 	MagicCookieKey:   "HARBOR_PLUGIN",
	// 	MagicCookieValue: "harborv1",
	// }

	pluginSet := plugin.PluginSet{}
	conf := config.Get()
	log.Info().Msgf("plugin dir: %s", conf.PluginsDir)

	// for _, pluginConf := range conf.Plugins {
	// 	pl := plugin.Plugin
	// }

	// client := plugin.NewClient(&plugin.ClientConfig{
	// 	Plugins: ,
	// })
	return pluginSet, nil
}
