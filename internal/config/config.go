package config

import (
	"github.com/oidc-mytoken/utils/utils/fileutil"
	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/zachmann/tip/pkg"
)

var conf *Config

// Get returns the Config
func Get() *Config {
	return conf
}

// Config holds the configuration for this application
type Config struct {
	Server  serverConf    `yaml:"server"`
	Logging loggingConf   `yaml:"logging"`
	TIP     pkg.TIPConfig `yaml:"tip"`
}

type serverConf struct {
	Port int     `yaml:"port"`
	TLS  tlsConf `yaml:"tls"`
}

type tlsConf struct {
	Enabled      bool   `yaml:"enabled"`
	RedirectHTTP bool   `yaml:"redirect_http"`
	Cert         string `yaml:"cert"`
	Key          string `yaml:"key"`
}

type loggingConf struct {
	Access   LoggerConf         `yaml:"access"`
	Internal internalLoggerConf `yaml:"internal"`
}

type internalLoggerConf struct {
	LoggerConf `yaml:",inline"`
	Smart      smartLoggerConf `yaml:"smart"`
}

// LoggerConf holds configuration related to logging
type LoggerConf struct {
	Dir    string `yaml:"dir"`
	StdErr bool   `yaml:"stderr"`
	Level  string `yaml:"level"`
}

type smartLoggerConf struct {
	Enabled bool   `yaml:"enabled"`
	Dir     string `yaml:"dir"`
}

func checkLoggingDirExists(dir string) error {
	if dir != "" && !fileutil.FileExists(dir) {
		return errors.Errorf("logging directory '%s' does not exist", dir)
	}
	return nil
}

func (log *loggingConf) validate() error {
	if err := checkLoggingDirExists(log.Access.Dir); err != nil {
		return err
	}
	if err := checkLoggingDirExists(log.Internal.Dir); err != nil {
		return err
	}
	if log.Internal.Smart.Enabled {
		if log.Internal.Smart.Dir == "" {
			log.Internal.Smart.Dir = log.Internal.Dir
		}
		if err := checkLoggingDirExists(log.Internal.Smart.Dir); err != nil {
			return err
		}
	}
	return nil
}

var possibleConfigLocations = []string{
	".",
	"config",
	"/etc/mytoken",
}

func validate() error {
	if conf == nil {
		return errors.New("config not set")
	}
	if err := conf.Logging.validate(); err != nil {
		return err
	}
	return nil
}

// Load reads the config file and populates the Config struct; then validates the Config
func Load() {
	load()
	if err := validate(); err != nil {
		log.Fatalf("%s", err)
	}
}

func load() {
	data, _ := fileutil.MustReadConfigFile("config.yaml", possibleConfigLocations)
	conf = &Config{}
	err := yaml.Unmarshal(data, conf)
	if err != nil {
		log.WithError(err).Fatal()
		return
	}
}
