### Ignored

- Revert the spectest-only stub pubkey aggregation (`SetStubPubkeyAggregation`): consensus-spec-tests `>= v1.7.0-alpha.13` compute real aggregate pubkeys with BLS disabled (consensus-specs#5489), so the stub breaks `next_sync_committee.aggregate_pubkey` on newer vectors.
