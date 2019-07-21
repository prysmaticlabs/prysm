#!/bin/bash
while test $# -gt 0; do
    case "$1" in
      --deposit-contract)
          shift
          DEPOSIT_CONTRACT=$1
          shift
          ;;
      *)
          echo "$1 is not a recognized flag!"
          exit 1;
          ;;
    esac
done

bazel run //beacon-chain -- --clear-db --deposit-contract $DEPOSIT_CONTRACT --web3provider=wss://goerli.infura.io/ws/v3/be3fb7ed377c418087602876a40affa1