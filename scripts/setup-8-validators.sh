#!/bin/sh

PRIVATE_KEY_PATH=~/priv

echo "clearing data"
DATA_PATH=/tmp/data
rm -rf $DATA_PATH
mkdir -p $DATA_PATH

CONTRACT="0x067a1C3455aa798911156ae06DC5C8dc64a6649D"
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
  ACCOUNTCMD+="_amd64_pure_stripped/validator accounts create --password $PASSWORD_PATH --keystore-path $KEYSTORE"

  $ACCOUNTCMD
done

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
  DEPOSITCMD+=" --keystoreUTCPath=$KEYSTORE"
  DEPOSITCMD+=" --depositAmount=3200000"

  $DEPOSITCMD
done