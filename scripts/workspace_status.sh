#!/bin/bash

echo STABLE_GIT_COMMIT $(git rev-parse HEAD)
echo DATE $(date --rfc-3339=seconds --utc)

# Use DOCKER_TAG from environment, if exists.
if [ $(printenv DOCKER_TAG) ]
then
    echo DOCKER_TAG $(printenv DOCKER_TAG)
else
    echo DOCKER_TAG $(git rev-parse --abbrev-ref HEAD)-$(git rev-parse --short=6 HEAD)
fi
