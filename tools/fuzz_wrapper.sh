#!/bin/bash

set -e

# A wrapper for libfuzz tests that sets test undeclared outputs directory as the first corpus
# which libfuzz will write to and the artifact prefix to write any crashes.

$1 "$TEST_UNDECLARED_OUTPUTS_DIR" "${@:2}" -artifact_prefix="$TEST_UNDECLARED_OUTPUTS_DIR"/
