# Ethereum Serenity Protocol Buffers

This package defines common protobuf messages and services used by Ethereum Serenity clients. Following the structure of:

```
proto/
  beacon/
    p2p/
      v1/
    rpc/
      v1/
  sharding/
    p2p/
      v1/
  testing/
```

We specify messages available for p2p communication common to beacon chain nodes and sharding clients.

For now, we are checking in all generated code to support native go dependency
management. The generated pb.go files can be derived from bazel's bin 
directory.

For example, when we build the testing go proto library 
`bazel build //proto/testing:ethereum_testing_go_proto` there is a pb.go 
generated at 
`bazel-bin/proto/testing/linux_amd64_stripped/ethereum_testing_go_proto\~/github.com/prysmaticlabs/prysm/proto/testing/test.pb.go`.
This generated file can be copied, or you can use you protoc locally if you
prefer.
