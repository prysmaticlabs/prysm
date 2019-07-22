#!/bin/bash

# Script to copy pb.go files from bazel build folder to appropriate location.
# Bazel builds to bazel-bin/... folder, script copies them back to original folder where .proto is.

bazel build //proto/...

# Get locations of pb.go files.

file_list=()
while IFS= read -d $'\0' -r file ; do
    file_list=("${file_list[@]}" "$file")
done < <(find -L $(bazel info bazel-bin)/proto -type f -name "*pb.go" -print0)
 
arraylength=${#file_list[@]}
searchstring="prysm/"

# Copy pb.go files from bazel-bin to original folder where .proto is.

for (( i=0; i<${arraylength}; i++ ));
do
  destination=${file_list[i]#*$searchstring}
  cp -R -L "${file_list[i]}" "$destination"
done

# Run goimports on newly generated protos until gogo protobuf's proto-gen-go 
# formats imports properly.
# https://github.com/gogo/protobuf/issues/554
goimports -w proto/**/*.pb.go
