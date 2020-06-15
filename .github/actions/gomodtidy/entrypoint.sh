#!/bin/sh -l
set -e

cd $GITHUB_WORKSPACE

/usr/local/go/bin/go mod tidy

echo "Git status:"
/usr/bin/git status
/usr/bin/git diff
/usr/bin/git diff go.sum

if [ "$(git status | grep -c 'nothing to commit, working tree clean')" = 1 ]; then
  echo "go.mod and go.sum are up to date"
  exit 0
fi

# Notify of any issues.
echo "The go.sum is not up to date: run 'go mod tidy' to update"
exit 1
