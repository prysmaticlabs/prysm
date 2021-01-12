#!/bin/bash

./bazel.sh --bazelrc=.buildkite-bazelrc coverage --config=remote-cache --features=norace --test_tag_filters="-race_on" --nocache_test_results -k  //...
find $(./bazel.sh --bazelrc=.buildkite-bazelrc info bazel-testlogs) -iname coverage.dat | xargs -t -rd '\n' ./bazel.sh --bazelrc=.buildkite-bazelrc run //tools/gocovmerge:gocovmerge -- > /tmp/cover.out
curl https://deepsource.io/cli | sh
./bin/deepsource report --analyzer test-coverage --key go --value-file /tmp/cover.out
bash <(curl -s https://codecov.io/bash) -s $(./bazel.sh info bazel-testlogs) -f '**/coverage.dat'
