load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

"""
Herumi's BLS library for go depends on
- herumi/bls-eth-go-binary
"""

def bls_dependencies():
    _maybe(
        http_archive,
        name = "herumi_bls_eth_go_binary",
        strip_prefix = "bls-eth-go-binary-e81c3e745d31cfee089456774aa707bc98c76523",
        urls = [
            "https://github.com/herumi/bls-eth-go-binary/archive/e81c3e745d31cfee089456774aa707bc98c76523.tar.gz",
        ],
        sha256 = "9d794d0856bfab78953798b5f446148ed5413c5e5695857c34e2282a80439f2c",
        build_file = "@prysm//third_party/herumi:bls_eth_go_binary.BUILD",
    )

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)
