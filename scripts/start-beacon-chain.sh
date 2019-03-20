#!/bin/sh

DEPOSIT_CONTRACT=DEPOSITCONTRACTHERE

CMD="bazel run //beacon-chain -- --deposit-contract $DEPOSIT_CONTRACT"

$CMD
