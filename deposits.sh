#!/bin/bash
for i in {1..5}
do
   bazel run //validator -- accounts create --keystore-path /Users/terence/altona --password "12345"
done
