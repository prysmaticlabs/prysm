#!/bin/sh

PRIVATE_KEY_PATH=~/priv
DEPOSIT_CONTRACT="0x067a1C3455aa798911156ae06DC5C8dc64a6649D"

DATA_DIR=/tmp/beacon
rm -rf $DATA_DIR
mkdir -p $DATA_DIR

CMD="bazel run //beacon-chain -- --web3provider wss://goerli.prylabs.net/websocket"
CMD+=" --datadir $DATA_DIR --deposit-contract $DEPOSIT_CONTRACT --demo-config --enable-tracing"

$CMD