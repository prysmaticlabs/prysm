package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/sirupsen/logrus"
)

const (
	path = "/webhooks"
)

var (
	log            = logrus.WithField("prefix", "gitmirror")
	configPathFlag = flag.String("config", "", "path to config yaml file")
)

func main() {
	flag.Parse()

	// Read required environment variable secrets.
	githubSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if githubSecret == "" {
		log.Fatal("Expected GITHUB_WEBHOOK_SECRET env variable")
	}
	githubMirrorPush := os.Getenv("GITHUB_PUSH_ACCESS_TOKEN")
	if githubMirrorPush == "" {
		log.Fatal("Expected GITHUB_MIRROR_PUSH_SECRET env variable")
	}

	// Setup a github webhook handler.
	hook, err := NewWebhookClient(Options.Secret(githubSecret))
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the configuration and git CLI.
	log.Infof("Loading server configuration")
	config, err := loadConfig(*configPathFlag)
	if err != nil {
		log.Fatal(err)
	}
	manager := newGitCLI(config.CloneBasePath)

	log.Infof("Initializing git configuration")
	if err := initializeGitConfig(githubMirrorPush); err != nil {
		log.Fatal(err)
	}

	log.Infof("Cloning specified repositories in config")
	// Clone repositories specified in the config. No-op if the repositories
	// have already been cloned before.
	if err := cloneRepos(config, manager); err != nil {
		log.Fatal(err)
	}

	// Setup HTTP handler for webhook events which then mirrors required changes
	// to specified directories via configuration.
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		payload, err := hook.Parse(r, ReleaseEvent)
		if err != nil {
			if err == ErrEventNotFound {
				log.Error("Github event not found in subscribed items, please check configuration")
			}
			log.WithError(err).Error("Could not parse Github webhook event")
		}
		release, ok := payload.(ReleasePayload)
		if !ok {
			return
		}
		log.Info("Received github release event via webhooks")
		if err := mirrorChanges(config, manager, release); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	log.Info("Listening on port 3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func initializeGitConfig(accessToken string) error {
	cmdStrings := [][]string{
		{
			"config",
			"--global",
			fmt.Sprintf(`url."https://api:%s@github.com".insteadOf"`, accessToken),
			"https://github.com/",
		},
		{
			"config",
			"--global",
			fmt.Sprintf(`url."https://ssh:%s@github.com/".insteadOf`, accessToken),
			"ssh://git@github.com/",
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
		data, err := io.ReadAll(stdout)
		if err != nil {
			return err
		}
		log.Errorf("%s", data)
		if err := cmd.Start(); err != nil {
			return err
		}
		if err := cmd.Wait(); err != nil {
			return err
		}
	}
	return nil
}

func cloneRepos(config *Config, manager *gitCLI) error {
	if err := fileutil.MkdirAll(config.CloneBasePath); err != nil {
		return err
	}
	for _, repo := range config.Repositories {
		if err := manager.Clone(repo.RemoteName, repo.RemoteUrl); err != nil {
			return err
		}
		if err := manager.Clone(repo.MirrorName, repo.MirrorUrl); err != nil {
			return err
		}
	}
	return nil
}
