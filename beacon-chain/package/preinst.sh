#!/bin/bash

set -e

useradd -s /bin/false --no-create-home --system --user-group prysm-beacon || true

mkdir -p /etc/prysm
mkdir -p /var/lib/prysm/beacon-chain
chown prysm-beacon:prysm-beacon /var/lib/prysm/beacon-chain
chmod 700 /var/lib/prysm/beacon-chain