package main

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/prysmaticlabs/prysm/shared/fileutil"
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
	return execCommandAndCaptureOutput(cmd)
}

// Add files to a github repository.
func (g *gitCLI) Add(remoteName string) error {
	repoPath := filepath.Join(g.reposBasePath, remoteName)
	cmd := exec.Command("git", "add", "--all")
	cmd.Dir = repoPath
	return execCommandAndCaptureOutput(cmd)
}

// Commit to a github repository with a commit message.
func (g *gitCLI) Commit(remoteName, msg string) error {
	repoPath := filepath.Join(g.reposBasePath, remoteName)
	cmd := exec.Command("git", "commit", "-m", fmt.Sprintf(`"%s"`, msg))
	cmd.Dir = repoPath
	return execCommandAndCaptureOutput(cmd)
}

// Checkout a repository's branch.
func (g *gitCLI) Checkout(remoteName, branch string) error {
	repoPath := filepath.Join(g.reposBasePath, remoteName)
	cmd := exec.Command("git", "checkout", branch)
	cmd.Dir = repoPath
	return execCommandAndCaptureOutput(cmd)
}

// Push to a github repository remote master branch.
func (g *gitCLI) Push(remoteName string) error {
	repoPath := filepath.Join(g.reposBasePath, remoteName)
	cmd := exec.Command("git", "push", "origin", "master")
	cmd.Dir = repoPath
	return execCommandAndCaptureOutput(cmd)
}

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
	log.Infof("Making target path %s in mirror %s", targetPath, targetRepo)
	if err := fileutil.MkdirAll(targetPath); err != nil {
		return err
	}
	log.Info("Copying folders...")
	cmd := exec.Command("cp", "-R", dirPath, targetPath)
	return execCommandAndCaptureOutput(cmd)
}

func execCommandAndCaptureOutput(cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	outBytes, err := io.ReadAll(stdout)
	if err != nil {
		return err
	}
	if len(outBytes) > 0 {
		fmt.Printf("%s\n", outBytes)
	}
	errBytes, err := io.ReadAll(stderr)
	if err != nil {
		return err
	}
	if len(errBytes) > 0 {
		fmt.Printf("%s\n", errBytes)
	}
	return cmd.Wait()
}
