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
        strip_prefix = "bls-eth-go-binary-4d3c66ab099d05706ccefcc3c1d1e0f70ff5b2bf",
        urls = [
            "https://github.com/herumi/bls-eth-go-binary/archive/4d3c66ab099d05706ccefcc3c1d1e0f70ff5b2bf.tar.gz",
        ],
	sha256 = "584a95d4965aaa95445d8f4816a00be88d3ec2c30052dad331d9f5d4fcad9ffc",
        build_file = "@prysm//third_party/herumi:bls_eth_go_binary.BUILD",
    )
    _maybe(
        http_archive,
        name = "herumi_mcl",
        strip_prefix = "mcl-7b4eb83d5bf0940504bfe891f70335d41f5a6037",
        urls = [
            "https://github.com/herumi/mcl/archive/7b4eb83d5bf0940504bfe891f70335d41f5a6037.tar.gz",
        ],
        sha256 = "ded0d4611d7115d949d9396cc45e111e8c4e58c4041975b2a0ab994b4de8fc9b",
        build_file = "@prysm//third_party/herumi:mcl.BUILD",
    )
    _maybe(
        http_archive,
        name = "herumi_bls",
        strip_prefix = "bls-a1c04f30892d6d65b53d1d216baebaa26da695f1",
        urls = [
            "https://github.com/herumi/bls/archive/a1c04f30892d6d65b53d1d216baebaa26da695f1.tar.gz",
        ],
        sha256 = "9ba743d0fc704ce38b084cbc036a709fb15e440b36f2efeed6a8c70e52bed82d",
        build_file = "@prysm//third_party/herumi:bls.BUILD",
    )

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)
