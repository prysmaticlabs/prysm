#!/bin/sh

PRIVATE_KEY_PATH=PUTPRIVKEYPATHHERE

echo "clearing data"
DATA_PATH=/tmp/data
rm -rf $DATA_PATH
mkdir -p $DATA_PATH

CONTRACT=PUTDEPOSITCONTRACTHERE
PASSWORD="password"
PASSWORD_PATH=$DATA_PATH/password.txt

UNAME=$(echo `uname` | tr '[A-Z]' '[a-z]')

echo $PASSWORD > $PASSWORD_PATH

bazel build //validator
bazel build //contracts/deposit-contract/sendDepositTx:sendDepositTx

for i in `seq 1 8`;
do
  echo "Generating validator $i"

  KEYSTORE=$DATA_PATH/keystore$i

  ACCOUNTCMD="bazel-bin/validator/$UNAME"
  ACCOUNTCMD+="_amd64_pure_stripped/validator accounts create --password $(cat $PASSWORD_PATH) --keystore-path $KEYSTORE"

  echo $ACCOUNTCMD

  $ACCOUNTCMD
done

for i in `seq 1 8`;
do
  KEYSTORE=$DATA_PATH/keystore$i

  CMD="bazel-bin/validator/"
  CMD+=$UNAME
  CMD+="_amd64_pure_stripped/validator --demo-config --password $(cat $PASSWORD_PATH) --keystore-path $KEYSTORE"

  echo $CMD

  nohup $CMD $> /tmp/validator$i.log &
done

echo "Started 8 validators"

for i in `seq 1 8`;
do
  echo "Sending TX for validator $i"

  KEYSTORE=$DATA_PATH/keystore$i

  DEPOSITCMD="bazel-bin/contracts/deposit-contract/sendDepositTx/$UNAME"
  DEPOSITCMD+="_amd64_stripped/sendDepositTx"
  DEPOSITCMD+=" --httpPath=https://goerli.prylabs.net"
  DEPOSITCMD+=" --passwordFile=$PASSWORD_PATH"
  DEPOSITCMD+=" --depositContract=$CONTRACT"
  DEPOSITCMD+=" --numberOfDeposits=1"
  DEPOSITCMD+=" --privKey=$(cat $PRIVATE_KEY_PATH)"
  DEPOSITCMD+=" --prysm-keystore=$KEYSTORE"
  DEPOSITCMD+=" --depositAmount=3200000"

  $DEPOSITCMD

  echo $DEPOSITCMD
done

echo "8 validators are running in the background. You can follow their logs at /tmp/validator#.log where # is replaced by the validator index of 1 through 8."

echo "To stop the processes, use 'pkill validator'"