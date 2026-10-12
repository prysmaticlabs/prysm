### Added

- Add `HEZE_FORK_VERSION` and `HEZE_FORK_EPOCH` config placeholders for the Heze fork, set to far-future on all networks, so upcoming Heze features (EIP-7805, EIP-8015, EIP-8365) can be gated on the fork epoch.
- Implement EIP-8365 (disallow new 0x00 validators): from the Heze fork, pending deposits that would create a new validator with BLS (`0x00`) withdrawal credentials are skipped before signature verification. Top-ups to existing validators are unaffected. The rule is gated on `HEZE_FORK_EPOCH` and is inactive on all networks.
