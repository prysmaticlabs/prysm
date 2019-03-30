#!/bin/sh

PRIVATE_KEY_PATH=~/priv

CMD="bazel run //contracts/deposit-contract/deployContract -- --httpPath=https://goerli.prylabs.net"
CMD+=" --privKey=$(cat $PRIVATE_KEY_PATH) --chainStart=8 --minDeposit=100000 --maxDeposit=3200000 --customChainstartDelay 120"

$CMD