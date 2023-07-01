package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	plugins "github.com/radding/harbor-plugins"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const CONFIG_FILENAME = "harbor.global"

type PluginType int

const (
	Manager PluginType = 1
	Builder PluginType = iota << 1
	Runner
)

type Plugin struct {
	Name           string `yaml:"name"`
	ArtifactURL    string `yaml:"artifact_url"`
	PluginLocation string `yaml:"plugin_location"`
	IsActive       bool   `yaml:"is_active"`
	SettingsPath   string `yaml:"settings_path"`
}

type AuthSchemes string

const (
	JWT      AuthSchemes = "jwt"
	KeyPair  AuthSchemes = "keypair"
	Basic    AuthSchemes = "basic"
	External AuthSchemes = "external"
	None     AuthSchemes = "none"
)

type PluginRepos struct {
	Name                 string      `yaml:"name"`
	BaseURL              string      `yaml:"url"`
	AuthenticationScheme AuthSchemes `yaml:"authentication_scheme"`
	AuthenticationAssets string      `yaml:"authentication_assets"`
}

type GlobalConfig struct {
	PluginsDir         string            `yaml:"plugin_dir"`
	Plugins            map[string]Plugin `yaml:"plugins"`
	PluginRepositories []PluginRepos     `yaml:"plugin_repos"`

	location           string
	management_plugins []*plugins.PluginClient
	plugins            map[string]*plugins.PluginClient
	// plugi
}

var globalConfig *GlobalConfig

func Get() *GlobalConfig {
	if globalConfig != nil {
		return globalConfig
	}
	return LoadConfig(".", GetDefaultConfigDir())
}

func LoadConfig(pathsToSearch ...string) *GlobalConfig {
	os.MkdirAll(GetDefaultPluginDirectory(), 0755)
	globalConfig = &GlobalConfig{
		PluginsDir:         GetDefaultPluginDirectory(),
		Plugins:            map[string]Plugin{},
		PluginRepositories: []PluginRepos{},

		location: filepath.Join(GetDefaultConfigDir(), CONFIG_FILENAME),
		plugins:  map[string]*plugins.PluginClient{},
	}
	for _, i := range pathsToSearch {
		fullPath, err := filepath.Abs(filepath.Join(i, CONFIG_FILENAME))
		log.Trace().Msgf("checking to see if config is %s", fullPath)
		if err != nil {
			log.Trace().Msgf("can't find full path: %s", err)
			return nil
		}
		data := openConfig(fullPath)
		if data != nil {
			d, err := ioutil.ReadAll(data)
			if err != nil {
				log.Error().Err(err).Msg("error reading file")
				return globalConfig
			}
			err = yaml.Unmarshal(d, globalConfig)
			if err != nil {
				log.Error().Err(err).Msg("Error unmarshalling config")
			}
			globalConfig.location = fullPath
			return globalConfig
		}
	}
	return globalConfig
}

func (g *GlobalConfig) Save() error {
	d, err := yaml.Marshal(g)
	log.Trace().Str("configuration", string(d)).Msgf("Saving configuration to %s", g.location)
	if err != nil {
		return errors.Wrap(err, "error marshalling to yaml")
	}
	os.MkdirAll(filepath.Dir(g.location), 0755)
	return os.WriteFile(g.location, d, 0755)
}

func openConfig(fullPath string) io.Reader {
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		log.Trace().Msg("config not found here")
		return nil
	}
	fi, err := os.Open(fullPath)
	if err != nil {
		log.Error().Err(err).Msgf("error opening %s", fi)
		return nil
	}
	log.Trace().Msgf("found config, opening it at %s", fullPath)
	return fi
}

func (g *GlobalConfig) LoadPlugins() error {
	plugins := map[string]Plugin{}
	for key, value := range g.Plugins {
		plugins[strings.ToLower(key)] = value
	}
	g.Plugins = plugins
	// plManager := plugs.New()
	// wg := &sync.WaitGroup{}
	// ctx, cancelFn := context.WithCancel(context.Background())
	// defer cancelFn()
	// for key, value := range g.Plugins {
	// 	wg.Add(1)
	// 	go func() {
	// 		defer wg.Done()
	// 		if ctx.Err() != nil {
	// 			return
	// 		}
	// 		err := g.loadPlugin(key, value)
	// 		if err != nil {
	// 			log.Error().Err(err)
	// 			cancelFn()
	// 			panic(err)
	// 		}
	// 	}()
	// }
	// wg.Wait()
	// err := context.Cause(ctx)
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (g *GlobalConfig) loadPlugin(name string, plugin Plugin) error {
	log.Trace().Msgf("loading %s", name)
	plugImpl, err := plugins.New().GetClient(plugin.PluginLocation, log.Logger)
	// plugImpl, err := plugins.New().GetClient("C:\")
	if err != nil {
		return err
	}
	impl, err := plugImpl.Dispense("client")
	if err != nil {
		return err
	}
	plug := impl.(*plugins.PluginClient)
	g.plugins[name] = plug
	return nil
}

func (g *GlobalConfig) GetPlugin(name string) (*plugins.PluginClient, error) {
	pl, ok := g.plugins[name]
	if !ok {
		pluginDef, ok := g.Plugins[name]
		if !ok {
			return nil, fmt.Errorf("can't get plugin with name %s, is not registered", name)
		}
		g.loadPlugin(name, pluginDef)
		pl, ok = g.plugins[name]
		if !ok {
			return nil, fmt.Errorf("can't get plugin instance, something went wrong")
		}
	}
	return pl, nil
}
