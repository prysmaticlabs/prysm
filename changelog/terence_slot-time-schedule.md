### Added

- Quick slots (EIP-8198) in Heze: the slot duration changes to `SLOT_DURATION_MS_HEZE` at the Heze fork epoch, intra-slot deadlines follow the new duration, and rewards, penalties, churn and data column retention (`MIN_BLOB_DATA_RETENTION_MS`) are scaled relative to `SLOT_DURATION_MS` to keep their wall-clock behavior. Networking, sync, RPC and validator client timeouts, pollers and cache lifetimes follow the slot duration schedule.
