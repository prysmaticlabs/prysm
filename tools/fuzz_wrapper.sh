#!/bin/bash

set -e

$1 $TEST_UNDECLARED_OUTPUTS_DIR ${@:2} -artifact_prefix=$TEST_UNDECLARED_OUTPUTS_DIR
