package main

import (
	"fmt"
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
		return err
	}
	log.Infof("Checking out branch %s in %s", payload.Release.TagName, repoName)
	if err := manager.Checkout(repoName, payload.Release.TagName); err != nil {
		return err
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
		return fmt.Errorf("could not find repo %s from release event in config", repoName)
	}
	for _, dir := range repo.MirrorDirectories {
		log.Infof("Fetching latest changes from mirror repo %s", repo.MirrorName)
		if err := manager.Fetch(repo.MirrorName); err != nil {
			return err
		}
		log.Infof("Copying directory %s from source %s to mirror repo %s", dir, repoName, repo.MirrorName)
		if err := manager.CopyDir(repoName, repo.MirrorName, dir); err != nil {
			return err
		}
	}
	log.Infof("Staging all changes in mirror %s", repo.MirrorName)
	if err := manager.Add(repo.MirrorName); err != nil {
		return err
	}
	commitMsg := payload.Release.Name
	log.Infof("Committing changes in mirror %s with message `%s`", repo.MirrorName, commitMsg)
	if err := manager.Commit(repo.MirrorName, commitMsg); err != nil {
		return err
	}
	log.Infof("Pushing changes to mirror %s", repo.MirrorName)
	if err := manager.Push(repo.MirrorName); err != nil {
		return err
	}
	return nil
}
