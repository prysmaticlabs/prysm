### Fixed

- Reject a voluntary exit in `POST /eth/v1/beacon/pool/voluntary_exits` when its epoch is more than 5 epochs ahead of the wall clock epoch.
