#!/bin/bash
set -e

# Continuous integration script to check that TODOs are in the correct format
OUTPUT="$(grep -PrinH '(?<!context\.)todo(?!\(#{0,1}\d+\))' --include \*.go *)";
if [[ "$OUTPUT" -ne "" ]] ;
then 
    echo "Invalid TODOs found. Failing." >&2;
    echo $OUTPUT >&2;
    exit 1;
fi
