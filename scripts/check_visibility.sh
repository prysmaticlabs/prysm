#!/bin/bash

# Continous Integration script to check that BUILD.bazel files have the correct
# visibility.

# Protected packages are:
#   //beacon-chain/...
#   //validator/...

# Duplicate redirect 5 to stdout so that it can be captured, but still printed
# nicely.
exec 5>&1

# Run gazelle while piping a copy of the output to stdout via 5.
changes=$(
bazel --batch --bazelrc=.buildkite-bazelrc query 'visible(//... except (//beacon-chain/... union //validator/...), (//beacon-chain/... union //validator/...))' | tee >(cat - >&5)
)

# If the captured stdout is not empty then targets are exposed!
if [ -z "$changes" ]
then
  echo "OK: Visibility is good."
  exit 0
else
  echo "FAIL: The above targets belong to protected packages and the targets \
are visible outside of their package!"
  echo "Please reduce the target visibility."
  exit 1
fi
