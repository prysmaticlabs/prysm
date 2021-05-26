package main

import (
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// Config for the tool.
type Config struct {
	CloneBasePath string       `mapstructure:"cloneBasePath"`
	Repositories  []ConfigRepo `mapstructure:"repositories"`
}

// ConfigRepo for git repo info.
type ConfigRepo struct {
	RemoteUrl         string   `mapstructure:"remoteUrl"`
	RemoteName        string   `mapstructure:"remoteName"`
	MirrorUrl         string   `mapstructure:"mirrorUrl"`
	MirrorName        string   `mapstructure:"mirrorName"`
	MirrorDirectories []string `mapstructure:"mirrorDirectories"`
}

func loadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		return nil, errors.New("wanted path to config, received none")
	}
	config := &Config{}
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	return config, nil
}
