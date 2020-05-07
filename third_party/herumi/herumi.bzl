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
        strip_prefix = "bls-eth-go-binary-524312f42ec6769c557990175baede559f4c43b9",
        urls = [
            "https://github.com/herumi/bls-eth-go-binary/archive/524312f42ec6769c557990175baede559f4c43b9.tar.gz",
        ],
        sha256 = "c5d7e014059a8d95a8eb76817a53ef6e67b95b4cf6e8ad763fedf96bd2405add",
        build_file = "@prysm//third_party/herumi:bls_eth_go_binary.BUILD",
    )
    _maybe(
        http_archive,
        name = "herumi_mcl",
        strip_prefix = "mcl-95e80e3c7b3d3ba0e56ae4d8fdc75f80318fab6a",
        urls = [
            "https://github.com/herumi/mcl/archive/95e80e3c7b3d3ba0e56ae4d8fdc75f80318fab6a.tar.gz",
        ],
        sha256 = "4679383dd32a1a2c35680e4364250309c302c5473ce156991ce496bfcf1affaa",
        build_file = "@prysm//third_party/herumi:mcl.BUILD",
    )
    _maybe(
        http_archive,
        name = "herumi_bls",
        strip_prefix = "bls-f9af288f71b74e0eb9df366d5510210d70eb92e9",
        urls = [
            "https://github.com/herumi/bls/archive/f9af288f71b74e0eb9df366d5510210d70eb92e9.tar.gz",
        ],
        sha256 = "97f8cdb9f1610753acac8e81ec7308245b9eb02a4bfac03a88aa298325520f30",
        build_file = "@prysm//third_party/herumi:bls.BUILD",
    )

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)
