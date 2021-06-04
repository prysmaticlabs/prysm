#!/usr/bin/env bash
#
# This script accepts the following parameters:
#
# * tag
# * github_api_token
#
# Script to mirror a tag from Prysm into EthereumAPIs protocol buffers
#
# Example:
#
# mirror-ethereumapis.sh github_api_token=TOKEN tag=v1.3.0
#
set -e

# Check dependencies.
# skipcq: SH-2034
export xargs=$(which gxargs || which xargs)

# Validate settings.
[ "$TRACE" ] && set -x

CONFIG=$*

for line in $CONFIG; do
  eval "$line"
done

# Define variables.
GH_API="https://api.github.com"
GH_REPO="$GH_API/repos/prysmaticlabs/ethereumapis"

AUTH="Authorization: token $github_api_token"
# skipcq: SH-2034
export WGET_ARGS="--content-disposition --auth-no-challenge --no-cookie"
# skipcq: SH-2034
export CURL_ARGS="-LJO#"

# Validate token.
curl -o /dev/null -sH "$AUTH" "$GH_REPO" || { echo "Error: Invalid repo, token or network issue!";  exit 1; }

git config --global user.email contact@prysmaticlabs.com
git config --global user.name prylabsbot
git config --global url."https://git:'$github_api_token'@github.com/".insteadOf "git@github.com/"

# Clone ethereumapis and prysm
git clone https://github.com/prysmaticlabs/prysm /tmp/prysm/
git clone https://github.com/prysmaticlabs/ethereumapis /tmp/ethereumapis/

# Checkout the release tag in prysm and copy over protos
cd /tmp/prysm && git checkout "$tag"
cp -Rf /tmp/prysm/proto/eth /tmp/ethereumapis
cd /tmp/ethereumapis || exit

# Replace imports in go files and proto files as needed
find ./eth -name '*.go' -print0 |
    while IFS= read -r -d '' line; do
        sed -i 's/prysm\/proto\/eth/ethereumapis\/eth/g' "$line"
    done

find ./eth -name '*.proto' -print0 |
    while IFS= read -r -d '' line; do
        sed -i 's/"proto\/eth/"eth/g' "$line"
    done

# Push to the mirror repository
git add --all
git commit -am "'$tag'"
git push origin master
