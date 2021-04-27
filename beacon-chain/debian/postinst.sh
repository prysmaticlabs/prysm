#!/bin/bash

set -e

chown prysm-beacon:prysm-beacon /etc/prysm/beacon-chain.yaml
chmod -R 600 /etc/prysm/beacon-chain.yaml