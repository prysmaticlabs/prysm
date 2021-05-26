package main

import (
	"flag"
	"os"

	"github.com/prysmaticlabs/prysm/shared/fileutil"
	//"github.com/prysmaticlabs/prysm/shared/fileutil"
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
	githubSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if githubSecret == "" {
		log.Fatal("Expected GITHUB_WEBHOOK_SECRET env variable")
	}
	githubMirrorPush := os.Getenv("GITHUB_MIRROR_PUSH_SECRET")
	if githubMirrorPush == "" {
		log.Fatal("Expected GITHUB_MIRROR_PUSH_SECRET env variable")
	}
	hook, err := New(Options.Secret(githubSecret))
	if err != nil {
		log.Fatal(err)
	}
	_ = hook
	// Clone repos from config if not exist.
	if *configPathFlag == "" {
		log.Fatal("Wanted path to config")
	}
	config, err := loadConfig(*configPathFlag)
	if err != nil {
		log.Fatal(err)
	}
	log.Info(config)
	if err := fileutil.MkdirAll(config.CloneBasePath); err != nil {
		log.Fatal(err)
	}
	manager := newGitCLI(config.CloneBasePath)
	for _, repo := range config.Repositories {
		if err := manager.Clone(repo.RemoteName, repo.RemoteUrl); err != nil {
			log.Fatal(err)
		}
		if err := manager.Clone(repo.MirrorName, repo.MirrorUrl); err != nil {
			log.Fatal(err)
		}
	}

	if err := mirrorChanges(config, manager, ReleasePayloadMinimal{
		Action:     "release",
		Release:    &ReleaseMinimal{TagName: "v1", Name: "Prysm v1"},
		Repository: &RepositoryMinimal{Name: "testingrepo"},
	}); err != nil {
		log.Fatal(err)
	}

	//fakeRelease := &ReleasePayload{
	//	Release: {},
	//}
	//if err := mirrorRepoChanges(
	//	manager,
	//	fakeRelease,
	//	nil,
	//	nil,
	//); err != nil {
	//	log.Fatal(err)
	//}
	//http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
	//	log.Info("Received github event")
	//	payload, err := hook.Parse(r, ReleaseEvent, PullRequestEvent)
	//	if err != nil {
	//		if err == ErrEventNotFound {
	//			log.Error("Github event not found in subscribed items, please check configuration")
	//		}
	//		log.WithError(err).Error("Could not parse Github webhook event")
	//	}
	//	release, ok := payload.(PullRequestPayload)
	//	if !ok {
	//		return
	//	}
	//	fmt.Printf("%+v", release)
	//	w.WriteHeader(http.StatusOK)
	//	fmt.Fprintf(w, "%+v", release)
	//})
	//log.Fatal(http.ListenAndServe(":3000", nil))
}
