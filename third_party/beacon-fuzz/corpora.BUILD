package(default_visibility = ["//visibility:public"])

filegroup(
    name = "all",
    srcs = glob(["**"]),
)

# See: https://github.com/sigp/beacon-fuzz-corpora

current_version = "0_11_0"

alias(
    name = "current_mainnet_attestation",
    actual = ":" + current_version + "_mainnet_attestation",
)

alias(
    name = "current_mainnet_attester_slashing",
    actual = ":" + current_version + "_mainnet_attester_slashing",
)

alias(
    name = "current_mainnet_block_header",
    actual = ":" + current_version + "_mainnet_block_header",
)

alias(
    name = "current_mainnet_beaconstate",
    actual = ":" + current_version + "_mainnet_beaconstate",
)

alias(
    name = "current_mainnet_proposer_slashing",
    actual = ":" + current_version + "_mainnet_proposer_slashing",
)

filegroup(
    name = "0_11_0_mainnet_attestation",
    srcs = glob(["0-11-0/mainnet/attestation/*"]),
)

filegroup(
    name = "0_11_0_mainnet_attester_slashing",
    srcs = glob(["0-11-0/mainnet/attester_slashing/*"]),
)

filegroup(
    name = "0_11_0_mainnet_block_header",
    srcs = glob(["0-11-0/mainnet/block_header/*"]),
)

filegroup(
    name = "0_11_0_mainnet_beaconstate",
    srcs = glob(["0-11-0/mainnet/beaconstate/*"]),
)

filegroup(
    name = "0_11_0_mainnet_proposer_slashing",
    srcs = glob(["0-11-0/mainnet/proposer_slashing/*"]),
)
