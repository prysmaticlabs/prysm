package main

import "github.com/spf13/viper"

type Config struct {
	CloneBasePath string `mapstructure:"cloneBasePath"`
	Repositories  []Repo `mapstructure:"repositories"`
}

type Repo struct {
	RemoteUrl         string   `mapstructure:"remoteUrl"`
	RemoteName        string   `mapstructure:"remoteName"`
	MirrorUrl         string   `mapstructure:"mirrorUrl"`
	MirrorName        string   `mapstructure:"mirrorName"`
	MirrorDirectories []string `mapstructure:"mirrorDirectories"`
}

func loadConfig(content string) (*Config, error) {
	config := &Config{}
	viper.SetConfigFile(content)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	return config, nil
}
