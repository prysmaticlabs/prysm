### Added

- Prototype EIP-8411 (fast execution payload broadcast) in place of the Gloas `execution_payload` gossip topic for prysm-with-prysm devnet interop: `ExecutionPayloadBid` commits to the chunked envelope via `payload_chunks_root`, envelopes travel as 64 `PayloadChunk` messages on `execution_payload_chunks` with per-chunk Merkle inclusion proofs, and nodes reconstruct, strictly decode, and import the envelope through the existing Gloas pipeline.
- Metrics `sync_payload_chunk_arrival_delay_seconds`, `sync_payload_chunk_reconstruction_seconds`, `sync_payload_chunk_received_total`, and `sync_payload_chunk_reconstruction_total`.

### Changed

- `ExecutionPayloadEnvelope` no longer carries `beacon_block_root` (EIP-8411); the binding travels out of band (wrapper attachment, `GenericSignedExecutionPayloadEnvelope.beacon_block_root`, REST `beacon_block_root` field) and stays in the storage-only blinded container.
- Envelopes reconstructed from verified chunks are authenticated by the bid commitment and carry a zero sentinel signature; envelope signature checks are skipped for them.

### Removed

- The `execution_payload` gossip topic, its validator, and its subscriber.
