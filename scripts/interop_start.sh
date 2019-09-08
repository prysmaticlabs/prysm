#!/bin/bash

"""
This file is intended for dockerfile deployment.
"""

# Flags
IDENTITY="" # P2P private key
PEERS="" # Comma separated list of peers
NUM_VALIDATORS="3" # Positive number of validators to operate.
GEN_STATE="" # filepath to ssz encoded state.
PORT="8000"

# Constants
BEACON_LOG_FILE="/tmp/beacon.log"
VALIDATOR_LOG_FILE="/tmp/validator.log"
YAML_KEY_FILE="/launch/keys.yaml"

usage() {
    echo "--identity=<identity>"
    echo "--peer=<peer>"
    echo "--num-validators=<number>"
    echo "--gen-state=<file path>"
    port "--port=<port number>"
}

while [ "$1" != "" ];
do
    PARAM=`echo $1 | awk -F= '{print $1}'`
    VALUE=`echo $1 | sed 's/^[^=]*=//g'`

    case $PARAM in
        --identity)
            IDENTITY=$VALUE
            ;;
        --peers)
            PEERS+=" $VALUE"
            ;;
        --num-validators)
            NUM_VALIDATORS=$VALUE
            ;;
        --gen-state)
            GEN_STATE=$VALUE
            ;;
        --port)
            PORT=$VALUE
            ;;
        --help)
            usage
            exit
            ;;
        *)
            echo "ERROR: unknown parameter \"$PARAM\""
            usage
            exit 1
            ;;
    esac
    shift
done


echo "Converting hex yaml keys to a format that Prysm understands"

# Expect YAML keys in hex encoded format. Convert this into the format the the validator already understands.
bazel run //tools/interop/convert-keys -- $YAML_KEY_FILE /tmp/keys.json

echo "Building beacon chain and validator binaries."

# Build both binaries prior to launching. The binary must be built using the build defined variable ssz=minimal
# to template protobuf constants.

BUILD_FLAGS="--define ssz=minimal"

bazel build $BUILD_FLAGS //beacon-chain //validator

echo "Starting beacon chain and logging to $BEACON_LOG_FILE"

BEACON_FLAGS="--bootstrap-node= \
  --deposit-contract=0xD775140349E6A5D12524C6ccc3d6A1d4519D4029 \
  --p2p-port=$PORT \
  --peer=$PEERS \
  --interop-load-genesis-state=$GEN_STATE \
  --log-file=$BEACON_LOG_FILE"

bazel run $BUILD_FLAG //beacon-chain -- $BEACON_FLAGS &

echo "Starting validator client with $NUM_VALIDATORS keys and logging to $BEACON_LOG_FILE"

VALIDATOR_FLAGS="--monitoring-port=9091 \
  --unencrypted-keys /tmp/keys.json \
  --log-file=$BEACON_LOG_FILE"

bazel run $BUILD_FLAG //validator -- $VALIDATOR_FLAGS &