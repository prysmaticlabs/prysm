# Bootnode query tool

To query the test network

```
bazel run //tools/bootnode-query -- /ip4/35.224.249.2/tcp/30001/p2p/QmQEe7o6hKJdGdSkJRh7WJzS6xrex5f4w2SPR6oWbJNriw
```

This will dial the testnet bootnode and attempt to connect to all of its peers.
