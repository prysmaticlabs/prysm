#!/bin/sh

DATA_PATH=/tmp/data
PASSWORD_PATH=$DATA_PATH/password.txt
PASSWORD="password"

echo $PASSWORD > $PASSWORD_PATH

bazel build //validator

for i in `seq 1 8`;
do
  KEYSTORE=$DATA_PATH/keystore$i

  UNAME=$(echo `uname` | tr '[A-Z]' '[a-z]')
  CMD="bazel-bin/validator/"
  CMD+=$UNAME
  CMD+="_amd64_pure_stripped/validator --demo-config --password $PASSWORD_PATH --keystore-path $KEYSTORE"

  nohup $CMD $> /tmp/validator$i.log &
done

echo "8 validators are running in the background. You can follow their logs at /tmp/validator#.log where # is replaced by the validator index of 1 through 8."

echo "To stop the processes, use 'pkill validator'"