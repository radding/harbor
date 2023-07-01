package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/radding/harbor/internal/config"
)

func InstallPlugin(pluginURL string) (config.Plugin, error) {
	return installLocal(pluginURL)
	// urlObj, err := url.Parse(pluginURL)
	// if err != nil {
	// 	return config.Plugin{}, errors.Wrap(err, "error parsing plugin URL")
	// }
	// if urlObj.Scheme == "file" {
	// 	return installLocal(urlObj.Path)
	// }
	// return config.Plugin{}, errors.New("not implemented")
}

func installLocal(localLocation string) (config.Plugin, error) {
	fullPath, err := filepath.Abs(localLocation)
	fmt.Println("fullPath:", localLocation)
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
