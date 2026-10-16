### Fixed

- Validate RPC message lengths against the expected SSZ type's maximum size.
- Bound beacon blocks-by-root requests by the post-Deneb `MAX_REQUEST_BLOCKS_DENEB` limit instead of the pre-Deneb `MAX_REQUEST_BLOCKS`.
