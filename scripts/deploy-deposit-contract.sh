#!/bin/sh

PRIVATE_KEY_PATH=~/priv

while test $# -gt 0; do
    case "$1" in
      --privkey-path)
          shift
          PRIVATE_KEY_PATH=$1
          shift
          ;;
      *)
          echo "$1 is not a recognized flag!"
          exit 1;
          ;;
    esac
done

CMD="bazel run //contracts/deposit-contract/deployContract --"

HTTPFLAG="--httpPath=https://goerli.prylabs.net"
PRIVFLAG="--privKey=$(cat $PRIVATE_KEY_PATH)"
CONFIGFLAGS="--chainStart=8 --minDeposit=100000 --maxDeposit=3200000 --customChainstartDelay 120"

CMD="$CMD $HTTPFLAG $PRIVFLAG $CONFIGFLAGS"

$CMD