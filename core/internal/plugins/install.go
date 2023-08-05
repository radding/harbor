package plugins

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	plugins "github.com/radding/harbor-plugins"
	"github.com/radding/harbor/internal/config"
	"github.com/rs/zerolog/log"
)

func InstallPlugin(pluginURL string) (config.Plugin, error) {
	pluginConf, err := installLocal(pluginURL)
	if err != nil {
		return pluginConf, errors.Wrap(err, "can't install local plugin")
	}

	plugin, err := plugins.NewClient(pluginConf.PluginLocation, log.Logger)
	defer plugin.Kill()
	if err != nil {
		return pluginConf, errors.Wrap(err, "can't start plugin")
	}

	conf, err := plugin.Install()
	if err != nil {
		return pluginConf, errors.Wrap(err, "can't install plugin")
	}
	pluginConf.Name = conf.Name
	return pluginConf, err
}

func installLocal(localLocation string) (config.Plugin, error) {
	fullPath, err := filepath.Abs(localLocation)
	if err != nil {
		return config.Plugin{}, err
	}
	pluginFileName := path.Join(fullPath, "plugin.json")
	fiContents, err := os.ReadFile(pluginFileName)
	if err != nil {
		return config.Plugin{}, errors.Wrap(err, "error getting plugin config")
	}
	pluginStuff := Plugin{}
	err = json.Unmarshal(fiContents, &pluginStuff)
	if err != nil {
		return config.Plugin{}, errors.Wrap(err, "error unmarshalling plugin configurations")
	}

	return config.Plugin{
		Name:           pluginStuff.Name,
		PluginLocation: path.Join(fullPath, pluginStuff.PluginExePath),
		IsActive:       true,
		SettingsPath:   pluginFileName,
	}, nil
}
