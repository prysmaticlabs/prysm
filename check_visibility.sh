#!/bin/bash

# Continous Integration script to check that BUILD.bazel files have the correct
# visibility.

# Protected packages are:
#   //beacon-chain/...
#   //client/...

# Duplicate redirect 5 to stdout so that it can be captured, but still printed
# nicely.
exec 5>&1

# Run gazelle while piping a copy of the output to stdout via 5.
changes=$(
bazel query 'visible(//... except (//beacon-chain/... union //client/...), (//beacon-chain/... union //client/...))' | tee >(cat - >&5)
)

# If the captured stdout is not empty then targets are exposed!
if [ -z "$changes" ]
then
  echo "OK: Visibility is good."
  exit 0
else
  echo "FAIL: The above targets belong to protected packages and the targets \
are visibile outside of their package!"
  echo "Please run reduce the target visibility."
  exit 1
fi
