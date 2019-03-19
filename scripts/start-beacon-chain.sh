#!/bin/sh

DEPOSIT_CONTRACT=DEPOSITCONTRACTHERE

DATA_DIR=/tmp/beacon
rm -rf $DATA_DIR
mkdir -p $DATA_DIR

CMD="bazel run //beacon-chain -- --web3provider wss://goerli.prylabs.net/websocket"
CMD+=" --datadir $DATA_DIR --deposit-contract $DEPOSIT_CONTRACT --demo-config"

$CMD