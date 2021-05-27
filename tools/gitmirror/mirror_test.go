package main

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

type mockGitManager struct{}

func Test_mirrorChanges(t *testing.T) {
	t.Run("repo_not_found_in_config", func(t *testing.T) {
		config := &Config{
			CloneBasePath: "tmp",
			Repositories: []ConfigRepo{
				{
					RemoteUrl:         "git/ethereumapis",
					RemoteName:        "ethereumapis",
					MirrorUrl:         "git/ethereumapis-mirror",
					MirrorName:        "ethereumapis-mirror",
					MirrorDirectories: []string{"/"},
				},
			},
		}
		payload := ReleasePayload{
			Action: "release",
			Release: Release{
				TagName:         "v1",
				TargetCommitish: "",
				URL:             "",
				Name:            "",
			},
			Repository: Repository{
				Name: "",
			},
		}
		manager := &mockGitManager{}
		err := mirrorChanges(config, manager, payload)
		require.ErrorContains(t, "could not find repo", err)
	})
}

func (m *mockGitManager) Fetch(name string) error {
	return nil
}

func (m *mockGitManager) Add(name string) error {
	return nil
}

func (m *mockGitManager) Commit(name, msg string) error {
	return nil
}

func (m *mockGitManager) Checkout(name, branch string) error {
	return nil
}

func (m *mockGitManager) Push(name string) error {
	return nil
}

func (m *mockGitManager) CopyDir(sourceRepo, targetRepo, dir string) error {
	return nil
}
