### Fixed

- `--suggested-gas-limit` now sets the default gas limit when `--proposer-settings-file` or `--proposer-settings-url` loads a version 2 settings file. Before, a v2 source dropped the flag and the validator client used the network default gas limit. Per-key gas limits still take precedence over the flag.
