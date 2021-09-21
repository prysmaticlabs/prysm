#!/bin/bash

# Note: The STABLE_ prefix will force a relink when the value changes when using rules_go x_defs.

echo STABLE_GIT_COMMIT "continuous-integration"
echo DATE "now"
echo DATE_UNIX "0"
echo DOCKER_TAG "ci-foo"
echo STABLE_GIT_TAG "c1000deadbeef"
