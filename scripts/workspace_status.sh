#!/bin/bash

echo STABLE_GIT_COMMIT $(git rev-parse HEAD)
echo DATE $(date --rfc-3339=seconds --utc)
echo DOCKER_TAG $(git rev-parse --abbrev-ref HEAD)-$(git rev-parse --short=6 HEAD)
