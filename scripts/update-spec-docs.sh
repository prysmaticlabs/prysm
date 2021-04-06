#!/bin/bash

declare -a files=("phase0/beacon-chain.md"
  "phase0/deposit-contract.md"
  "phase0/fork-choice.md"
  "phase0/p2p-interface.md"
  "phase0/validator.md"
  "phase0/weak-subjectivity.md"
)

BASE_URL="https://raw.githubusercontent.com/ethereum/eth2.0-specs/dev/specs"
OUTPUT_DIR="tools/analyzers/specdocs/data"

for file in "${files[@]}"; do
  echo "downloading $file"
  wget -q -O $OUTPUT_DIR/"$file" --no-check-certificate --content-disposition $BASE_URL/"$file"
done
