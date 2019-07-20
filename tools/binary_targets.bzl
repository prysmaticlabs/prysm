# Supported pairs of system and architecture. (GOOS, GOARCH).
binary_targets = [
    ("linux", "amd64"),
    ("linux", "arm64"),
    # TODO(2849): Enable after cross OS compilation is supported.
    # https://github.com/prysmaticlabs/prysm/issues/2849
    #    ("darwin", "amd64"),
    #    ("windows", "amd64"),
]

# Determine binary targets from a supported pair. These targets are part of the
# distributable bundle.
def determine_targets(pair, common_files):
    targets = {
        "//beacon-chain:beacon-chain-{}-{}".format(
            pair[0],
            pair[1],
        ): "beacon-chain",
        "//validator:validator-{}-{}".format(
            pair[0],
            pair[1],
        ): "validator",
    }
    targets.update(common_files)
    return targets
