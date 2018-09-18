#!/bin/bash

# Continuous integration script to check that TODOs are in the correct format
if grep -Priq '(?<!context\.)todo(?!\(#{0,1}\d+\))' --include \*.go *;
then 
    echo "Invalid TODOs found. Failing." >&2
    exit 1;
fi
