package workspaces

import (
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

func Initialize(name, dir, configName string) error {
	pathToWrite := filepath.Join(dir, configName)
	log.Info().Msgf("writing new configurations to: %s", pathToWrite)
	if !validateDoesNotExsist(dir) {
		return errors.New("can't validate target directory")
	}

	conf := WorkspaceConfig{
		Name:     name,
		Packages: []Package{},
	}

	data, err := yaml.Marshal(conf)
	if err != nil {
		return errors.Wrap(err, "unable to marshal to yaml")
	}

	err = os.MkdirAll(dir, 0744)
	if err != nil {
		return errors.Wrapf(err, "error ensuring directory exists")
	}
	err = os.WriteFile(pathToWrite, data, 0744)
	return err
}

func validateDoesNotExsist(dir string) bool {
	fullPath, err := filepath.Abs(os.ExpandEnv(dir))
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get full path of %s", dir)
		return false
	}

	info, err := os.Stat(fullPath)
	if err != nil && !os.IsNotExist(err) {
		log.Error().
			Err(err).
			Msgf("got error getting info for path %s:", fullPath)
		return false
	} else if os.IsNotExist(err) {
		return true
	}
	if info != nil && !info.IsDir() {
		return false
	}

	f, err := os.Open(fullPath)

	if err != nil {
		log.Error().Err(err).Msgf("failed to open dir %s", fullPath)
		return false
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	if err != io.EOF {
		if err == nil {
			log.Error().Msgf("%s is not an empty directory, refusing to initialize", fullPath)
			return false
		}
		log.Error().Err(err).Msgf("failed to read dir %s", fullPath)
		return false
	}

	return true
}
