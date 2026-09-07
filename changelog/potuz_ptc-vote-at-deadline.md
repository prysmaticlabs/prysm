### Fixed

- PTC members now request the payload attestation data again at the payload attestation deadline when the first request comes back before it, so a payload revealed between `PAYLOAD_DUE_BPS` and `PAYLOAD_ATTESTATION_DUE_BPS` yields a `payload_present=false` vote instead of no vote at all. Previously only the REST validator client recovered from this, leaving the default gRPC client silent.
- A `204 No Content` response to an SSZ read is surfaced as a typed error carrying the status instead of an empty success body, and payload attestation data failures are now classified by HTTP status as well as gRPC code, so a withheld or block-less response is logged and counted as a skip rather than an error.
