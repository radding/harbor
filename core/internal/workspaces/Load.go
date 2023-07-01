package workspaces

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const MAX_DISTANCE = 1000

var Config *WorkspaceConfig = nil

func GetConfig() (WorkspaceConfig, error) {
	if Config != nil {
		return *Config, nil
	}
	configFi, configPath, err := findConfig("workspace.conf", ".", 0, "")
	if err != nil {
		return WorkspaceConfig{}, errors.Wrap(err, "error finding configuration")
	}

	defaultConf := &WorkspaceConfig{
		Commands:    map[string]Command{},
		Packages:    []Package{},
		subPackages: map[string]WorkspaceConfig{},
	}

	bts, err := ioutil.ReadAll(configFi)
	if err != nil {
		return *defaultConf, errors.Wrap(err, "error reading configuration file")
	}

	err = yaml.Unmarshal(bts, defaultConf)
	if err != nil {
		return *defaultConf, errors.Wrapf(err, "error unmarshalling yaml")
	}

	Config = defaultConf
	Config.location = configPath

	err = Config.loadSubPackages()

	return *Config, err
}

func findConfig(name, dir string, dist int64, lastCheck string) (io.Reader, string, error) {
	if dist > MAX_DISTANCE {
		log.Trace().Msgf("exceeded max recursive distance, can not find config file")
		return nil, "", errors.New("not in a harbor workspace")
	}
	fullPath, err := filepath.Abs(filepath.Join(dir, name))
	if fullPath == lastCheck {
		return nil, "", errors.New("not in a harbor workspace")
	}
	log.Trace().Msgf("checking to see if config is %s", fullPath)
	if err != nil {
		return nil, "", errors.Wrapf(err, "error getting absolute path of %s/%s", dir, name)
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		log.Trace().Msg("config does not exsist here, checking parent")
		return findConfig(name, filepath.Join(filepath.Dir(fullPath), ".."), dist+1, fullPath) // Search parent
	}
	fi, err := os.Open(fullPath)
	return fi, fullPath, err
}
