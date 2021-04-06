#!/bin/bash

# This script will pull the latest specs from https://github.com/ethereum/eth2.0-specs repo, extract code blocks
# and save them for reference in "all-defs.md" file (in separate directories for phase0, altair etc).

declare -a files=("phase0/beacon-chain.md"
  "phase0/deposit-contract.md"
  "phase0/fork-choice.md"
  "phase0/p2p-interface.md"
  "phase0/validator.md"
  "phase0/weak-subjectivity.md"
)

BASE_URL="https://raw.githubusercontent.com/ethereum/eth2.0-specs/dev/specs"
OUTPUT_DIR="tools/analyzers/specdocs/data"

# Trunc all-defs files (they will contain extracted python code blocks).
echo -n >$OUTPUT_DIR/phase0/all-defs.md

for file in "${files[@]}"; do
  OUTPUT_PATH=$OUTPUT_DIR/$file
  echo "$file"
  echo "- downloading"
  wget -q -O "$OUTPUT_PATH" --no-check-certificate --content-disposition $BASE_URL/"$file"
  echo "- extracting all code blocks"
  sed -n '/^```/,/^```/ p' <"$OUTPUT_PATH" >>"${OUTPUT_PATH%/*}"/all-defs.md
  echo "- removing raw file"
  rm "$OUTPUT_PATH"
done
