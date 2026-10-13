### Changed

- Web3Signer key files are read only at startup. Manual edits to the file no longer update the running validator; use the Keymanager API with `--rpc` (or `--web`) for runtime changes, or restart after editing. API updates can overwrite manual edits made while running.

### Fixed

- Serialize Web3Signer API key changes and persist the complete key file atomically before publishing account updates. Publish updates only when the public-key set changes.
- Fix a read-lock leak in the Web3Signer remote key delete handler when no keys were loaded.
- Match Web3Signer remote key deletes case-insensitively on the public key hex.
