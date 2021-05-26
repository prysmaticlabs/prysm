package main

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitManager defines a struct which can deal with git repositories' common actions.
type GitManager interface {
	Fetch(name string) error
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

// Clone a github repository from a remote url.
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
	stdoutStr := fmt.Sprintf("Got error from clone %s\n", data)
	fmt.Println(stdoutStr)
	var alreadyExists bool
	if strings.Contains(stdoutStr, "already exists") {
		alreadyExists = true
	}
	if err := cmd.Wait(); err != nil && !alreadyExists {
		return err
	}
	return nil
}

// Fetch from github repository.
func (g *gitCLI) Fetch(remoteName string) error {
	repoPath := filepath.Join(g.reposBasePath, remoteName)
	cmd := exec.Command("git", "fetch")
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

// Add files to a github repository.
func (g *gitCLI) Add(remoteName string) error {
	repoPath := filepath.Join(g.reposBasePath, remoteName)
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

// Commit to a github repository with a commit message.
func (g *gitCLI) Commit(remoteName, msg string) error {
	repoPath := filepath.Join(g.reposBasePath, remoteName)
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

// Checkout a repository's branch.
func (g *gitCLI) Checkout(remoteName, branch string) error {
	repoPath := filepath.Join(g.reposBasePath, remoteName)
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

// Push to a github repository remote master branch.
func (g *gitCLI) Push(remoteName string) error {
	repoPath := filepath.Join(g.reposBasePath, remoteName)
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
