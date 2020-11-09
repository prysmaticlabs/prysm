load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

"""
Herumi's BLS library for go depends on
- herumi/mcl
- herumi/bls
- herumi/bls-eth-go-binary
"""

def bls_dependencies():
    _maybe(
        http_archive,
        name = "herumi_bls_eth_go_binary",
        strip_prefix = "bls-eth-go-binary-4b463a10c225efa46f6d7525ef44822e4b5b7b05",
        urls = [
            "https://github.com/herumi/bls-eth-go-binary/archive/4b463a10c225efa46f6d7525ef44822e4b5b7b05.tar.gz",
        ],
        sha256 = "23de2840e53818f650bc656bbd9e9a128806f686bcfec56918605868f05b5ac3",
        build_file = "@prysm//third_party/herumi:bls_eth_go_binary.BUILD",
    )
    _maybe(
        http_archive,
        name = "herumi_mcl",
        strip_prefix = "mcl-bd5a3686924d4ef38f994bb400f87a684ee65fe8",
        urls = [
            "https://github.com/herumi/mcl/archive/bd5a3686924d4ef38f994bb400f87a684ee65fe8.tar.gz",
        ],
        sha256 = "db90a9c671f4bb59183e51ae2ae3bfe0d5c99b64877c0fca2e7b047fd88b2a41",
        build_file = "@prysm//third_party/herumi:mcl.BUILD",
    )
    _maybe(
        http_archive,
        name = "herumi_bls",
        strip_prefix = "bls-f53dadd5a51900f94b7aecff0063feada2f4bb30",
        urls = [
            "https://github.com/herumi/bls/archive/f53dadd5a51900f94b7aecff0063feada2f4bb30.tar.gz",
        ],
        sha256 = "201e779894ee22cfa9fbe07ccbf9fe4ba7ba2c6fb868c280bf765908c437c8d3",
        build_file = "@prysm//third_party/herumi:bls.BUILD",
    )

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)
