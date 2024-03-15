#!/bin/bash

set -xe

# Constants
PROJECT_ROOT=$(pwd)
PRYSM_DIR="${PROJECT_ROOT%/hack}/testing/spectest"
EXCLUSION_LIST="$PRYSM_DIR/exclusions.txt"
BAZEL_DIR="/tmp/spectest_report"
SPEC_REPO="git@github.com:ethereum/consensus-spec-tests.git"
SPEC_DIR="/tmp/consensus-spec"

# Run spectests
bazel test //testing/spectest/... --test_env=SPEC_TEST_REPORT_OUTPUT_DIR="$BAZEL_DIR"

# Ensure the SPEC_DIR exists and is a git repository
if [ -d "$SPEC_DIR/.git" ]; then
    echo "Repository already exists. Pulling latest changes."
    (cd "$SPEC_DIR" && git pull) || exit 1
else
    echo "Cloning the GitHub repository."
    git clone "$SPEC_REPO" "$SPEC_DIR" || exit 1
fi

# Finding all *_tests.txt files in BAZEL_DIR and concatenating them into tests.txt
find "$BAZEL_DIR" -type f -name '*_tests.txt' -exec cat {} + > "$PRYSM_DIR/tests.txt"

# Generating spec.txt
(cd "$SPEC_DIR" && find tests -maxdepth 3 -mindepth 3 -type d > "$PRYSM_DIR/spec.txt") || exit 1

# Comparing spec.txt with tests.txt and generating report.txt
while IFS= read -r line; do
    if grep -Fxq "$line" "$EXCLUSION_LIST"; then
        echo "Skipping excluded item: $line"
        continue
    fi
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
    echo "Tests Missing"
    grep '^missing' "$PRYSM_DIR/report.txt"
    echo ""
    echo "Tests Found"
    grep '^found' "$PRYSM_DIR/report.txt"
} > "$PRYSM_DIR/report_temp.txt" && mv "$PRYSM_DIR/report_temp.txt" "$PRYSM_DIR/report.txt"

# Clean up
rm -f "$PRYSM_DIR/tests.txt" "$PRYSM_DIR/spec.txt"
