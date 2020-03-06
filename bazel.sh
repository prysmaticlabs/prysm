#!/bin/bash

env -i \
 PATH=/usr/bin:/bin \
 HOME=$HOME \
 bazel "$@"
