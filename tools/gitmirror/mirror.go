package main

import (
	"fmt"

	"github.com/pkg/errors"
)

// Mirror changes from specified source repositories to target repositories upon
// receiving a ReleasePayload via Github webhooks. The function does the following actions.
// - Checkout the release tag in the source repo
// - Copy the directories we desire to a mirror repo's folder
// - Git add all, git commit, and git push to master branch of the mirror repo
func mirrorChanges(config *Config, manager GitManager, payload ReleasePayload) error {
	repoName := payload.Repository.Name
	log.Infof("Fetching changes from %s", repoName)
	if err := manager.Fetch(repoName); err != nil {
		return errors.Wrapf(err, "could not fetch repository %s", repoName)
	}
	log.Infof("Checking out branch %s in %s", payload.Release.TagName, repoName)
	if err := manager.Checkout(repoName, payload.Release.TagName); err != nil {
		return errors.Wrapf(err, "could not checkout tag %s from repository %s", payload.Release.TagName, repoName)
	}
	var repo ConfigRepo
	var found bool
	for _, repository := range config.Repositories {
		if repository.RemoteName == repoName {
			repo = repository
			found = true
		}
	}
	if !found {
		return fmt.Errorf("could not find repository %s from release event in config", repoName)
	}
	if len(repo.MirrorDirectories) == 0 {
		log.Info("Mirroring entire repository")
		log.Infof("Fetching latest changes from mirror repository %s", repo.MirrorName)
		if err := manager.Fetch(repo.MirrorName); err != nil {
			return errors.Wrapf(err, "could not fetch repository %s", repo.MirrorName)
		}
		if err := manager.CopyDir(repoName, repo.MirrorName, "."); err != nil {
			return errors.Wrapf(
				err,
				"could not copy source %s to mirror repository %s",
				repoName,
				repo.MirrorName,
			)
		}
	} else {
		for _, dir := range repo.MirrorDirectories {
			log.Infof("Fetching latest changes from mirror repository %s", repo.MirrorName)
			if err := manager.Fetch(repo.MirrorName); err != nil {
				return errors.Wrapf(err, "could not fetch repository %s", repo.MirrorName)
			}
			log.Infof("Copying directory %s from source %s to mirror repo %s", dir, repoName, repo.MirrorName)
			if err := manager.CopyDir(repoName, repo.MirrorName, dir); err != nil {
				return errors.Wrapf(err, "could not copy directory %s from source %s to mirror repository %s", dir, repoName, repo.MirrorName)
			}
		}
	}
	log.Infof("Staging all changes in mirror %s", repo.MirrorName)
	if err := manager.Add(repo.MirrorName); err != nil {
		return errors.Wrapf(err, "could not add git changes to repository %s", repo.MirrorName)
	}
	commitMsg := payload.Release.Name
	log.Infof("Committing changes in mirror %s with message `%s`", repo.MirrorName, commitMsg)
	if err := manager.Commit(repo.MirrorName, commitMsg); err != nil {
		return errors.Wrapf(err, "could not commit changes to repository %s", repo.MirrorName)
	}
	log.Infof("Pushing changes to mirror %s", repo.MirrorName)
	if err := manager.Push(repo.MirrorName); err != nil {
		return errors.Wrapf(err, "could not push changes to repository %s", repo.MirrorName)
	}
	return nil
}
