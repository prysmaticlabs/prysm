#!/bin/sh
# Copyright (c) 2017 Arista Networks, Inc.
# Use of this source code is governed by the Apache License 2.0
# that can be found in the COPYING file.

# egrep that comes with our Linux distro doesn't like \d, so use [0-9]
notice='Copyright \(c\) 20[0-9][0-9] Arista Networks, Inc.'
files=`git diff-tree --no-commit-id --name-only --diff-filter=ACMR -r HEAD | \
	egrep '\.(go|proto|py|sh)$' | grep -v '\.pb\.go$'`
status=0

for file in $files; do
	if ! egrep -q "$notice" $file; then
		echo "$file: missing or incorrect copyright notice"
		status=1
	fi
done

exit $status
