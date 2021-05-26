package main

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/prysmaticlabs/prysm/shared/fileutil"
)

const (
	reposFolder = "tmp"
)

type GitManager interface {
	Add(name string) error
	Commit(name, msg string) error
	Checkout(name, branch string) error
	Push(name string) error
	CopyDir(sourceRepo, targetRepo, dir string) error
}

type gitCLI struct {
	reposBasePath string
}

func newGitCLI(reposBasePath string) *gitCLI {
	return &gitCLI{reposBasePath: reposBasePath}
}

func (g *gitCLI) Clone(remoteName, remoteUrl string) error {
	repoPath := filepath.Join(g.reposBasePath, remoteName)
	cmd := exec.Command("git", "clone", remoteUrl, repoPath)
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
	stdoutStr := fmt.Sprintf("%s", data)
	var alreadyExists bool
	if strings.Contains(stdoutStr, "already exists") {
		alreadyExists = true
	}
	if err := cmd.Wait(); err != nil && !alreadyExists {
		return err
	}
	return nil
}

func (g *gitCLI) Add(name string) error {
	repoPath := filepath.Join(g.reposBasePath, name)
	cmd := exec.Command("git", "add", "--all")
	cmd.Dir = repoPath
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	slurp, _ := io.ReadAll(stdout)
	fmt.Printf("%s\n", slurp)
	slurp2, _ := io.ReadAll(stderr)
	fmt.Printf("%s\n", slurp2)
	return cmd.Wait()
}

func (g *gitCLI) Commit(name, msg string) error {
	repoPath := filepath.Join(g.reposBasePath, name)
	cmd := exec.Command("git", "commit", "-m", fmt.Sprintf(`"%s"`, msg))
	cmd.Dir = repoPath
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	slurp, _ := io.ReadAll(stdout)
	fmt.Printf("%s\n", slurp)
	slurp2, _ := io.ReadAll(stderr)
	fmt.Printf("%s\n", slurp2)
	return cmd.Wait()
}

func (g *gitCLI) Checkout(repoName, branch string) error {
	repoPath := filepath.Join(g.reposBasePath, repoName)
	cmd := exec.Command("git", "checkout", branch)
	cmd.Dir = repoPath
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	slurp, _ := io.ReadAll(stdout)
	fmt.Printf("%s\n", slurp)
	slurp2, _ := io.ReadAll(stderr)
	fmt.Printf("%s\n", slurp2)
	return cmd.Wait()
}

func (g *gitCLI) Push(name string) error {
	repoPath := filepath.Join(g.reposBasePath, name)
	cmd := exec.Command("git", "push", "origin", "master")
	cmd.Dir = repoPath
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	slurp, _ := io.ReadAll(stdout)
	fmt.Printf("%s\n", slurp)
	slurp2, _ := io.ReadAll(stderr)
	fmt.Printf("%s\n", slurp2)
	return cmd.Wait()
}

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

// Navigate to the repository that just had a release
// Checkout the release tag in the repo
// Copy the directories we desire (specified by some configmap per repo) to a target repo's folder (config as well)
// Git add all, git commit, and git push to master branch of the target repo
func mirrorChanges(config *Config, manager GitManager, payload ReleasePayloadMinimal) error {
	repoName := payload.Repository.Name
	if err := manager.Checkout(repoName, payload.Release.TagName); err != nil {
		return err
	}
	var repo Repo
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
