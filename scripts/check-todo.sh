#!/bin/bash

declare -a VIOLATION 
RETURN=0

# Continuous integration script to check that TODOs are in the correct format
while IFS=: read file line rest;
do
    git blame -L $line,$line $file
    RETURN=1
done <<<$(grep -PrinH '(?<!context\.)todo(?!\(#{0,1}\d+\))' --include \*.go *)

if [[ $RETURN ]];
then
    echo "Invalid TODOs found. Failing." >&2
fi


exit $RETURN
