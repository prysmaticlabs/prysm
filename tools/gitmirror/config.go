package main

import (
	"fmt"
	"io"
	"os/exec"

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

func initializeGitConfig(accessToken, user, email string) error {
	cmdStrings := [][]string{
		{
			"config",
			"--global",
			"user.email",
			email,
		},
		{
			"config",
			"--global",
			"user.name",
			user,
		},
		{
			"config",
			"--global",
			fmt.Sprintf(`url."https://git:%s@github.com/".insteadOf`, accessToken),
			"git@github.com/",
		},
	}
	for _, str := range cmdStrings {
		cmd := exec.Command("git", str...)
		stdout, err := cmd.StderrPipe()
		if err != nil {
			log.Fatal(err)
		}
		if err := cmd.Start(); err != nil {
			return err
		}
		data, err := io.ReadAll(stdout)
		if err != nil {
			return err
		}
		if err := cmd.Wait(); err != nil {
			return err
		}
	}
	return nil
}
