#!/bin/sh

DEPOSIT_CONTRACT="0xf430ce6768e5D63Af3a451994d1aeaaF7718a600"

DATA_DIR=/tmp/beacon
rm -rf $DATA_DIR
mkdir -p $DATA_DIR

CMD="bazel run //beacon-chain -- --web3provider wss://goerli.prylabs.net/websocket"
CMD+=" --datadir $DATA_DIR --deposit-contract $DEPOSIT_CONTRACT --demo-config --enable-tracing"

$CMD