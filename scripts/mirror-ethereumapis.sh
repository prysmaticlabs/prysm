#!/usr/bin/env bash
#
# This script accepts the following parameters:
#
# * owner
# * repo
# * tag
# * filename
# * github_api_token
#
# Script to upload a release asset using the GitHub API v3.
#
# Example:
#
# mirror-ethereumapis.sh github_api_token=TOKEN owner=prysmaticlabs repo=playground tag=v1.3.9
#

# Check dependencies.
set -e
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
GH_REPO="$GH_API/repos/$owner/$repo"
GH_TAGS="$GH_REPO/releases/tags/$tag"
AUTH="Authorization: token $github_api_token"
# skipcq: SH-2034
export WGET_ARGS="--content-disposition --auth-no-challenge --no-cookie"
# skipcq: SH-2034
export CURL_ARGS="-LJO#"

if [[ "$tag" == 'LATEST' ]]; then
  GH_TAGS="$GH_REPO/releases/latest"
fi

# Validate token.
curl -o /dev/null -sH "$AUTH" "$GH_REPO" || { echo "Error: Invalid repo, token or network issue!";  exit 1; }

# Read asset tags.
response=$(curl -sH "$AUTH" "$GH_TAGS")

echo "$response"

# Get ID of the asset based on given filename.
#eval "$(echo "$response" | grep -m 1 "id.:" | grep -w id | tr : = | tr -cd '[[:alnum:]]=')"
#[ "$id" ] || { echo "Error: Failed to get release id for tag: $tag"; echo "$response" | awk 'length($0)<100' >&2; exit 1; }
#
## Upload asset
#echo "Uploading asset... "
#
## Construct url
#GH_ASSET="https://uploads.github.com/repos/$owner/$repo/releases/$id/assets?name=$(basename "$filename")"
#
#echo "$GH_ASSET"
#
#curl "$GITHUB_OAUTH_BASIC" --data-binary @"$filename" -H "Authorization: token $github_api_token" -H "Content-Type: application/octet-stream" "$GH_ASSET"
