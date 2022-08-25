#! /usr/bin/env bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
BUILDKITE_TAG=$(git rev-parse --verify HEAD)
OUTDIR=/tmp/$BUILDKITE_TAG
mkdir -p $OUTDIR

declare -a configs=("--config=linux_amd64 --config=llvm" "--config=release --config=linux_amd64 --define=blst_modern=true" "--config=release --config=linux_amd64 --define=blst_modern=true" "--config=osx_amd64_docker" "--config=osx_arm64_docker" "--config=windows_amd64_docker")
declare -a targetsuffix=("$BUILDKITE_TAG-linux-amd64" "$BUILDKITE_TAG-modern-linux-amd64" "$BUILDKITE_TAG-linux-arm64" "$BUILDKITE_TAG-darwin-amd64" "$BUILDKITE_TAG-darwin-arm64" "$BUILDKITE_TAG-windows-amd64.exe")

bazel query 'kind(rule, //cmd/...:*)' --output label_kind --logging=0 2>/dev/null | grep go_binary | tr -s ' ' | cut -d' ' -f3 | while read target ; do
	for i in "${!configs[@]}"
	do
		bname=$(echo $target | cut -d':' -f2)
		cfg="${configs[$i]}"
		suff="${targetsuffix[$i]}"
		echo "bazel build --config=release $cfg $target"
		output=$(bazel cquery $target  --output starlark --starlark:file=tools/cquery/format-out/output.cquery 2>/dev/null)
		fname=$OUTDIR/$bname-$suff
		echo "cp $output $fname"
		pushd $OUTDIR > /dev/null
			echo "sha256sum $fname > $fname.sha256"
			echo "gpg -o $fname.sig --sign --detach-sig $fname"
		popd > /dev/null
		echo "$SCRIPT_DIR/../hack/upload-github-release-asset.sh github_api_token=$TOKEN owner=prysmaticlabs repo=prysm tag=$BUILDKITE_TAG filename=$fname"
	done
done
echo "gsutil -m cp -a public-read /tmp/validator-$BUILDKITE_TAG-* gs://prysmaticlabs.com/releases/"
echo "gsutil -m cp -a public-read /tmp/beacon-chain-$BUILDKITE_TAG-* gs://prysmaticlabs.com/releases/"
echo "gsutil -m cp -a public-read /tmp/client-stats-$BUILDKITE_TAG-* gs://prysmaticlabs.com/releases/"
echo $BUILDKITE_TAG > /tmp/latest
echo "gsutil -h "Cache-Control:no-cache,max-age=0" -h "Content-Type:text/html;charset=UTF-8" cp -a public-read /tmp/latest gs://prysmaticlabs.com/releases/latest"
echo "gsutil -m acl ch -u AllUsers:R gs://prysmaticlabs.com/releases/*"
echo "./hack/tag-versioned-docker-images.sh "
echo "./hack/tag-versioned-docker-images.sh -s"

