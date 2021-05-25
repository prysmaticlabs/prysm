package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

const (
	path = "/webhooks"
)

var log = logrus.WithField("prefix", "gitmirror")

func main() {
	githubSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if githubSecret == "" {
		log.Fatal("Expected GITHUB_WEBHOOK_SECRET env variable")
	}
	hook, err := New(Options.Secret(githubSecret))
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		log.Info("Received github event")
		payload, err := hook.Parse(r, ReleaseEvent, PullRequestEvent)
		if err != nil {
			if err == ErrEventNotFound {
				log.Error("Github event not found in subscribed items, please check configuration")
			}
			log.WithError(err).Error("Could not parse Github webhook event")
		}
		release, ok := payload.(PullRequestPayload)
		if !ok {
			return
		}
		fmt.Printf("%+v", release)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%+v", release)
	})
	log.Fatal(http.ListenAndServe(":3000", nil))
}
