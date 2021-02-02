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
        strip_prefix = "bls-eth-go-binary-57372fb273715552ba3f0f2e9c2806e9db940200",
        urls = [
            "https://github.com/herumi/bls-eth-go-binary/archive/57372fb273715552ba3f0f2e9c2806e9db940200.tar.gz",
        ],
        sha256 = "52dcc1776e6a219af980f5c6f70f8d95d46720f8398bd9978b605074ea4a48f1",
        build_file = "@prysm//third_party/herumi:bls_eth_go_binary.BUILD",
    )
    _maybe(
        http_archive,
        name = "herumi_mcl",
        strip_prefix = "mcl-c1bcf317a15868ee4a2192c8ad50e387253e1e64",
        urls = [
            "https://github.com/herumi/mcl/archive/c1bcf317a15868ee4a2192c8ad50e387253e1e64.tar.gz",
        ],
        sha256 = "05886c21fe6fe869a3b426fd9b56561dfc49df114b858a95c276ef5b99f54388",
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
