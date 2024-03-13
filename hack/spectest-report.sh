#!/bin/bash

set -xe

# Run spectests
bazel test //testing/spectest/... --flaky_test_attempts=3

# Constants
PROJECT_ROOT=$(pwd)
PRYSM_DIR="${PROJECT_ROOT%/hack}/testing/spectest"
BAZEL_DIR=$(bazel info bazel-testlogs)/spectest
SPEC_REPO="git@github.com:ethereum/consensus-spec-tests.git"
SPEC_DIR="tmp/consensus-spec"

# Ensure the SPEC_DIR exists and is a git repository
if [ -d "$SPEC_DIR/.git" ]; then
    echo "Repository already exists. Pulling latest changes."
    (cd "$SPEC_DIR" && git pull) || exit 1
else
    echo "Cloning the GitHub repository."
    git clone "$SPEC_REPO" "$SPEC_DIR" || exit 1
fi

# Extracting tests from outputs.zip and storing in tests.txt
find "$BAZEL_DIR" -name 'outputs.zip' -exec unzip -p {} \; > "$PRYSM_DIR/tests.txt"

# Generating spec.txt
(cd "$SPEC_DIR" && find tests -maxdepth 3 -mindepth 3 -type d > "$PRYSM_DIR/spec.txt") || exit 1

# Comparing spec.txt with tests.txt and generating report.txt
while IFS= read -r line; do
    if grep -q "$line" "$PRYSM_DIR/tests.txt"; then
        echo "found $line"
    else
        echo "missing $line"
    fi
done < "$PRYSM_DIR/spec.txt" > "$PRYSM_DIR/report.txt"

# Formatting report.txt
{
    echo "Prysm Spectest Report"
    echo ""
    grep '^missing' "$PRYSM_DIR/report.txt"
    echo ""
    grep '^found' "$PRYSM_DIR/report.txt"
} > "$PRYSM_DIR/report_temp.txt" && mv "$PRYSM_DIR/report_temp.txt" "$PRYSM_DIR/report.txt"

# Clean up
rm -f "$PRYSM_DIR/tests.txt" "$PRYSM_DIR/spec.txt"
