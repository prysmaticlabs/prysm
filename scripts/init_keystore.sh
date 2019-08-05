#!/bin/bash

PASSWORD="password"
UNAME=$(echo `uname` | tr '[A-Z]' '[a-z]')

bazel build //validator

START_INDEX=1
END_INDEX=64
KEYSTORE_PATH=~/Desktop/eth2keystore2

while test $# -gt 0; do
    case "$1" in
      --end-index)
          shift
          END_INDEX=$1
          shift
          ;;
      --start-index)
          shift
          START_INDEX=$1
          shift
          ;;
      --keystore-path)
          shift
          KEYSTORE_PATH=$1
          shift
          ;;
      *)
          echo "$1 is not a recognized flag!"
          exit 1;
          ;;
    esac
done

for i in `seq $START_INDEX $END_INDEX`;
do
  echo "Generating validator key $i"

  ACCOUNTCMD="bazel-bin/validator/${UNAME}_amd64_pure_stripped/validator accounts create --password ${PASSWORD} --keystore-path ${KEYSTORE_PATH}"

  echo $ACCOUNTCMD

  $ACCOUNTCMD
done
