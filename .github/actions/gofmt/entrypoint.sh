#!/bin/sh -l
set -e

cd $GITHUB_WORKSPACE

# Check if any files are not formatted.
nonformatted="$(gofmt -l $1 2>&1)"

# Return if `go fmt` passes.
[ -z "$nonformatted" ] && exit 0

# Notify of issues with formatting.
echo "Following files need to be properly formatted:"
echo "$nonformatted"
exit 1
