#!/bin/bash

echo STABLE_GIT_COMMIT $(git rev-parse HEAD)
echo DATE $(date --rfc-3339=seconds --utc)
