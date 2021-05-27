package main

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
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
			Repository: Repository{
				Name: "non-existentrepo",
			},
		}
		manager := &mockGitManager{}
		err := mirrorChanges(config, manager, payload)
		require.ErrorContains(t, "could not find repo", err)
	})

	t.Run("copies_directories_for_mirrors", func(t *testing.T) {
		hook := logTest.NewGlobal()
		config := &Config{
			CloneBasePath: "tmp",
			Repositories: []ConfigRepo{
				{
					RemoteUrl:         "git/ethereumapis",
					RemoteName:        "ethereumapis",
					MirrorUrl:         "git/ethereumapis-mirror",
					MirrorName:        "ethereumapis-mirror",
					MirrorDirectories: []string{"proto", "config"},
				},
			},
		}
		payload := ReleasePayload{
			Repository: Repository{
				Name: "ethereumapis",
			},
		}
		manager := &mockGitManager{}
		err := mirrorChanges(config, manager, payload)
		require.NoError(t, err)
		require.LogsContain(
			t,
			hook,
			"Copying directory proto from source ethereumapis to mirror repo ethereumapis-mirror",
		)
		require.LogsContain(
			t,
			hook,
			"Copying directory config from source ethereumapis to mirror repo ethereumapis-mirror",
		)
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
