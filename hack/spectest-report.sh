#!/bin/bash

set -x

# Prysm spectest directory to start from
start_directory="/prysmaticLabs/prysm/testing/spectest"
# Can be found with os.Getenv("TEST_UNDECLARED_OUTPUTS_DIR")
bazel_output_dir="<path to bazel>/testlogs/testing/spectest"

# URL of the Ethereum Spec Tests GitHub repo
github_repo_url="git@github.com:ethereum/consensus-spec-tests.git"
# Temporary directory to clone the GitHub repository
temp_dir=$(mktemp -d)

# Clone the GitHub repository
git clone "$github_repo_url" "$temp_dir"

tests=$(find "$bazel_output_dir" -name 'outputs.zip' -follow)
for i in $tests; do unzip -p "$i";
echo;
done > $start_directory/tests.txt

cd "$temp_dir" && find tests -maxdepth 3 -mindepth 3 -type d > "$start_directory/spec.txt"

for i in $(cat "$start_directory/spec.txt"); do if grep $i "$start_directory/tests.txt" > /dev/null; then echo "found $i"; else echo "missing $i";
fi; done > "$start_directory/report.txt"

{
    echo "Prysm Spectest Report"
    echo ""
    grep '^missing' $start_directory/report.txt
    echo ""
    grep '^found' $start_directory/report.txt
} > $start_directory/temp_report.txt && mv $start_directory/temp_report.txt $start_directory/report.txt

# Clean up
rm -rf "$temp_dir" $start_directory/tests.txt $start_directory/spec.txt
