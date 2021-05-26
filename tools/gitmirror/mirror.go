package main

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"

	"github.com/prysmaticlabs/prysm/shared/fileutil"
)

// CopyDir from a github repository to a target repository.
func (g *gitCLI) CopyDir(sourceRepo, targetRepo, dir string) error {
	dirPath := filepath.Join(g.reposBasePath, sourceRepo, dir)
	targetPath := filepath.Join(g.reposBasePath, targetRepo, dir)
	ok, err := fileutil.HasDir(dirPath)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if err := fileutil.MkdirAll(targetPath); err != nil {
		return err
	}
	cmd := exec.Command("cp", "-R", dirPath, targetPath)
	fmt.Println(cmd.String())
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
	fmt.Printf("%s\n", data)
	return cmd.Wait()
}

// Mirror changes from specified source repositories to target repositories upon
// receiving a ReleasePayload via Github webhooks. The function does the following actions.
// - Checkout the release tag in the source repo
// - Copy the directories we desire to a mirror repo's folder
// - Git add all, git commit, and git push to master branch of the mirror repo
func mirrorChanges(config *Config, manager GitManager, payload ReleasePayload) error {
	repoName := payload.Repository.Name
	if err := manager.Checkout(repoName, payload.Release.TagName); err != nil {
		return err
	}
	var repo ConfigRepo
	for _, repository := range config.Repositories {
		if repository.RemoteName == repoName {
			repo = repository
		}
	}
	for _, dir := range repo.MirrorDirectories {
		if err := manager.CopyDir(repoName, repo.MirrorName, dir); err != nil {
			return err
		}
	}
	if err := manager.Add(repo.MirrorName); err != nil {
		return err
	}
	commitMsg := payload.Release.Name
	if err := manager.Commit(repo.MirrorName, commitMsg); err != nil {
		return err
	}
	if err := manager.Push(repo.MirrorName); err != nil {
		return err
	}
	return nil
}
