package main

import (
	"flag"
	"net/http"
	"os"

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
	githubMirrorPush := os.Getenv("GITHUB_MIRROR_PUSH_SECRET")
	if githubMirrorPush == "" {
		log.Fatal("Expected GITHUB_MIRROR_PUSH_SECRET env variable")
	}

	// Setup a github webhook handler.
	hook, err := NewWebhookClient(Options.Secret(githubSecret))
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the configuration and git CLI.
	config, err := loadConfig(*configPathFlag)
	if err != nil {
		log.Fatal(err)
	}
	manager := newGitCLI(config.CloneBasePath)

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
	log.Fatal(http.ListenAndServe(":3000", nil))
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
