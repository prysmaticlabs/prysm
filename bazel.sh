#!/bin/bash

env -i \
 PATH=/usr/bin:/bin \
 bazel "$@"
