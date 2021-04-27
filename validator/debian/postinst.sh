#!/bin/bash

set -e

chown prysm-validator:prysm-validator /etc/prysm/validator.yaml
chmod -R 600 /etc/prysm/validator.yaml