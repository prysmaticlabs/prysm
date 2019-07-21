#!/bin/bash

PRIVATE_KEY_PATH=~/priv

DATA_PATH=/tmp/data

PASSWORD="password"
PASSWORD_PATH=$DATA_PATH/password.txt

UNAME=$(echo `uname` | tr '[A-Z]' '[a-z]')

echo $PASSWORD > $PASSWORD_PATH

INDEX=9

while test $# -gt 0; do
    case "$1" in
      --deposit-contract)
          shift
          DEPOSIT_CONTRACT=$1
          shift
          ;;
      --index)
          shift
          INDEX=$1
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

KEYSTORE=$DATA_PATH/keystore$INDEX

echo "Generating validator $INDEX"

ACCOUNTCMD="bazel-bin/validator/${UNAME}_amd64_pure_stripped/validator accounts create --password $(cat $PASSWORD_PATH) --keystore-path $KEYSTORE"
$ACCOUNTCMD


echo "Sending TX for validator $INDEX"

HTTPFLAG="--httpPath=https://goerli.infura.io/v3/be3fb7ed377c418087602876a40affa1"
PASSFLAG="--passwordFile=$PASSWORD_PATH"
CONTRACTFLAG="--depositContract=$DEPOSIT_CONTRACT"
PRIVFLAG="--privKey=$(cat $PRIVATE_KEY_PATH)"
KEYFLAG="--prysm-keystore=$KEYSTORE"
AMOUNTFLAG="--depositAmount=3200000"

CMD="bazel-bin/contracts/deposit-contract/sendDepositTx/${UNAME}_amd64_stripped/sendDepositTx"

DEPOSITCMD="$CMD $HTTPFLAG $PASSFLAG $CONTRACTFLAG $PRIVFLAG $KEYFLAG $AMOUNTFLAG"

$DEPOSITCMD

echo "Started validator $INDEX"

CMD="bazel-bin/validator/${UNAME}_amd64_pure_stripped/validator --beacon-rpc-provider localhost:4545 --password $(cat $PASSWORD_PATH) --keystore-path $KEYSTORE"
$CMD
