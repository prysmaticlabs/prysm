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
        strip_prefix = "bls-eth-go-binary-e81c3e745d31cfee089456774aa707bc98c76523",
        urls = [
            "https://github.com/herumi/bls-eth-go-binary/archive/e81c3e745d31cfee089456774aa707bc98c76523.tar.gz",
        ],
        sha256 = "9d794d0856bfab78953798b5f446148ed5413c5e5695857c34e2282a80439f2c",
        build_file = "@prysm//third_party/herumi:bls_eth_go_binary.BUILD",
    )
    _maybe(
        http_archive,
        name = "herumi_mcl",
        strip_prefix = "mcl-c08437c973004cf64895da197eb7076d44354aff",
        urls = [
            "https://github.com/herumi/mcl/archive/c08437c973004cf64895da197eb7076d44354aff.tar.gz",
        ],
        sha256 = "4118dfdcf86d98cdc78349cf9e51a0b58f4ecfea4b974a3d2df9ea11dd2cb6ad",
        build_file = "@prysm//third_party/herumi:mcl.BUILD",
    )
    _maybe(
        http_archive,
        name = "herumi_bls",
        strip_prefix = "bls-4ae022a6bb71dc518d81f22141d71d2a1f767ab3",
        urls = [
            "https://github.com/herumi/bls/archive/4ae022a6bb71dc518d81f22141d71d2a1f767ab3.tar.gz",
        ],
        sha256 = "9f07c0ce33502e3bd2a3018814ed509093a0865ce16fa16cefff7ba11fb4537e",
        build_file = "@prysm//third_party/herumi:bls.BUILD",
    )

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)
