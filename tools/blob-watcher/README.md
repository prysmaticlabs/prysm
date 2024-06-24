# BlobWatcher

Blobwatcher is a tool to monitor your execution client's mempool for blob transactions
and determine how long it takes for them to get included. Data tracked by the tool:
- BaseFee monitoring for both blobs and the network
- Users propagating blob transactions along with the appropriate labelling for popular rollups.
- Builder Monitoring by blob transactions included
- Transaction Pool Monitoring For Blobs

This tool currently only works using a websocket endpoint.

```
 bazel run //tools/blob-watcher:blob-watcher -- --execution-endpoint ws://localhost:8546 --metrics-endpoint localhost:8080

```

Flags:
```
  -execution-endpoint string
        Path to webscocket endpoint for execution client. (default "ws://localhost:8546")
  -metrics-endpoint string
        Path for our metrics server. (default "localhost:8080")
  -origin-secret string
        Origin string for websocket connection

```