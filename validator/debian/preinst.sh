#!/bin/bash

set -e

useradd -s /bin/false --no-create-home --system --user-group prysm-validator || true

mkdir -p /etc/prysm
mkdir -p /var/lib/prysm/validator
chown prysm-validator:prysm-validator /var/lib/prysm/validator
chmod 700 /var/lib/prysm/validator