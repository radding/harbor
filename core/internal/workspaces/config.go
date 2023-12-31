package workspaces

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	plugins "github.com/radding/harbor-plugins"
	mathparser "github.com/radding/harbor/internal/MathParser"
	"github.com/radding/harbor/internal/config"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type Package struct {
	Name *string `yaml:"name"`
	Path string  `yaml:"path"`
}

type Dependency struct {
	PackageName string `yaml:"pkg"`
	CommandName string `yaml:"command"`
}

type RunCondition struct {
	Expr *mathparser.Expression
}

func (r *RunCondition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	data := ""
	unmarshal(&data)
	var err error = nil
	r.Expr, err = mathparser.Parse(data)
	return err
}

type Command struct {
	Type          string                 `yaml:"type"`
	Command       string                 `yaml:"command"`
	RunConditions []*RunCondition        `yaml:"conditions"`
	Dependencies  []Dependency           `yaml:"depends_on"`
	Settings      map[string]interface{} `yaml:"options"`
}

type CacheSettings struct {
	Provider string                 `yaml:"provider"`
	Settings map[string]interface{} `yaml:"Settings"`
}

type WorkspaceConfig struct {
	Name          string             `yaml:"workspace_name"`
	Packages      []Package          `yaml:"packages"`
	CacheSettings *CacheSettings     `yaml:"cache"`
	Commands      map[string]Command `yaml:"commands"`

	location    string
	subPackages map[string]WorkspaceConfig
}

func (w *WorkspaceConfig) VariableLookUpService() mathparser.VariableLookUp {
	n := newVariableResolver()
	n.registerProvider("env", ProviderFunc(getEnvVariable))
	return n
}

func (w *WorkspaceConfig) GetLocalCacheDir() string {
	if w.CacheSettings != nil && w.CacheSettings.Settings != nil {
		localCache, ok := w.CacheSettings.Settings["local_cache_dir"]
		if ok {
			return localCache.(string)
		}
	}
	return filepath.Join(w.WorkspaceRoot(), ".harbor")
}

func (w *WorkspaceConfig) GetCacher() (plugins.PluginClient, error) {
	cacheSettings := w.CacheSettings
	if cacheSettings == nil {
		cacheSettings = &CacheSettings{
			Provider: "local_cache",
		}
	}
	return config.Get().GetPlugin(cacheSettings.Provider)
}

func (w *WorkspaceConfig) AddSubPackage(name string, conf WorkspaceConfig) {
	if w.subPackages == nil {
		w.subPackages = map[string]WorkspaceConfig{}
	}
	conf.Name = name
	w.subPackages[name] = conf
}

func (w *WorkspaceConfig) GetPackageConfig(packageName string) (WorkspaceConfig, error) {
	conf, ok := w.subPackages[packageName]
	if !ok {
		return WorkspaceConfig{}, fmt.Errorf("error getting package named %s: does not exsist", packageName)
	}
	return conf, nil
}

func (w *WorkspaceConfig) GetAllSubPackages() map[string]WorkspaceConfig {
	return w.subPackages
}

func (w *WorkspaceConfig) Save() error {
	log.Trace().Msgf("saving current configuration to %s", w.location)
	bts, err := yaml.Marshal(w)
	if err != nil {
		return errors.Wrap(err, "error marshalling config file")
	}

	return os.WriteFile(w.location, bts, 0755)
}

func (w *WorkspaceConfig) WorkspaceRoot() string {
	return filepath.Dir(w.location)
}

func (w *WorkspaceConfig) Location() string {
	return w.location
}

func loadConfig(path string) (WorkspaceConfig, error) {
	defaultConf := &WorkspaceConfig{
		Commands: map[string]Command{},
		Packages: []Package{},
	}

	fi, err := os.Open(path)
	if fi != nil {
		defer fi.Close()
	}
	if err != nil {
		return WorkspaceConfig{}, err
	}

	bts, err := ioutil.ReadAll(fi)
	if err != nil {
		return *defaultConf, errors.Wrap(err, "error reading configuration file")
	}

	err = yaml.Unmarshal(bts, defaultConf)
	if err != nil {
		return *defaultConf, errors.Wrapf(err, "error unmarshalling yaml")
	}
	defaultConf.location = path
	return *defaultConf, nil
}

func (w *WorkspaceConfig) loadSubPackages() error {
	matches := []string{}
	for _, pkg := range w.Packages {
		ms, err := filepath.Glob(strings.Join([]string{w.WorkspaceRoot(), pkg.Path}, "/"))
		if err != nil {
			return errors.Wrapf(err, "error running glob: %s", pkg.Path)
		}
		matches = append(matches, ms...)
	}
	matches = filterNonDirs(matches)
	log.Trace().Msgf("Found packages: %s", matches)
	for _, pkg := range matches {
		conf, err := loadConfig(filepath.Join(pkg, "harbor.conf"))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		} else if errors.Is(err, os.ErrNotExist) {
			log.Trace().Msgf("%s is not a harbor workspace, ignoring", pkg)
			continue
		}
		w.subPackages[conf.Name] = conf
	}
	subPackages := []string{}
	for name, pkg := range w.subPackages {
		subPackages = append(subPackages, fmt.Sprintf("%s@%s", name, pkg.location))
	}
	log.Trace().Msgf("loaded subpackages: %s", strings.Join(subPackages, ", "))
	return nil
}

func filterNonDirs(paths []string) []string {
	res := []string{}
	for _, path := range paths {
		fi, err := os.Open(path)
		if err != nil {
			log.Fatal().Err(err).Msgf("error opening path: %s", path)
		}
		if stat, _ := fi.Stat(); stat.IsDir() {
			res = append(res, path)
		}
	}
	return res
}
