#!/bin/bash

PRIVATE_KEY_PATH=~/priv

echo "clearing data"
DATA_PATH=/tmp/data
rm -rf $DATA_PATH
mkdir -p $DATA_PATH

PASSWORD="password"
PASSWORD_PATH=$DATA_PATH/password.txt

UNAME=$(echo `uname` | tr '[A-Z]' '[a-z]')

echo $PASSWORD > $PASSWORD_PATH

bazel build //validator
bazel build //contracts/deposit-contract/sendDepositTx

START_INDEX=1
END_INDEX=8

while test $# -gt 0; do
    case "$1" in
      --deposit-contract)
          shift
          DEPOSIT_CONTRACT=$1
          shift
          ;;
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

for i in `seq $START_INDEX $END_INDEX`;
do
  echo "Generating validator $i"

  KEYSTORE=$DATA_PATH/keystore$i

  ACCOUNTCMD="bazel-bin/validator/${UNAME}_amd64_pure_stripped/validator accounts create --password $(cat $PASSWORD_PATH) --keystore-path $KEYSTORE"

  echo $ACCOUNTCMD

  $ACCOUNTCMD
done

for i in `seq $START_INDEX $END_INDEX`;
do
  KEYSTORE=$DATA_PATH/keystore$i

  CMD="bazel-bin/validator/${UNAME}_amd64_pure_stripped/validator --password $(cat $PASSWORD_PATH) --keystore-path $KEYSTORE"

  echo $CMD

  nohup $CMD $> /tmp/validator$i.log &
done

echo "Started $END_INDEX validators"

for i in `seq $START_INDEX $END_INDEX`;
do
  echo "Sending TX for validator $i"

  KEYSTORE=$DATA_PATH/keystore$i

  HTTPFLAG="--httpPath=https://goerli.prylabs.net"
  PASSFLAG="--passwordFile=$PASSWORD_PATH"
  CONTRACTFLAG="--depositContract=$DEPOSIT_CONTRACT"
  PRIVFLAG="--privKey=$(cat $PRIVATE_KEY_PATH)"
  KEYFLAG="--prysm-keystore=$KEYSTORE"
  AMOUNTFLAG="--depositAmount=3200000"

  CMD="bazel-bin/contracts/deposit-contract/sendDepositTx/${UNAME}_amd64_stripped/sendDepositTx"

  DEPOSITCMD="$CMD $HTTPFLAG $PASSFLAG $CONTRACTFLAG $PRIVFLAG $KEYFLAG $AMOUNTFLAG"

  $DEPOSITCMD
done

echo "$END_INDEX validators are running in the background. You can follow their logs at /tmp/validator#.log where # is replaced by the validator index of $START_INDEX through $END_INDEX."

echo "To stop the processes, use 'pkill validator'"