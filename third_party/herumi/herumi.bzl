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
        strip_prefix = "bls-eth-go-binary-71567a52ad652c61b67c7d5bc64848b036f6caca",
        urls = [
            "https://github.com/herumi/bls-eth-go-binary/archive/71567a52ad652c61b67c7d5bc64848b036f6caca.tar.gz",
        ],
        sha256 = "25700f60b68dbd10ef7d29f91b91a3fa2b055231d46399a5d760989cb5d60eca",
        build_file = "@prysm//third_party/herumi:bls_eth_go_binary.BUILD",
    )
    _maybe(
        http_archive,
        name = "herumi_mcl",
        strip_prefix = "mcl-a3cb8ff42172cd730e834a8ad56c2d35e5f45c9d",
        urls = [
            "https://github.com/herumi/mcl/archive/a3cb8ff42172cd730e834a8ad56c2d35e5f45c9d.tar.gz",
        ],
        sha256 = "7652f198b17b30da5497561c1957656e59a90ad9cf85a01a8803324d5baa79fb",
        build_file = "@prysm//third_party/herumi:mcl.BUILD",
    )
    _maybe(
        http_archive,
        name = "herumi_bls",
        strip_prefix = "bls-1b48de51f4f76deb204d108f6126c1507623f739",
        urls = [
            "https://github.com/herumi/bls/archive/1b48de51f4f76deb204d108f6126c1507623f739.tar.gz",
        ],
        sha256 = "d9102e3d0948bbcc83a4632415bf29694b00d0ce30a98f05174a2fa89e374264",
        build_file = "@prysm//third_party/herumi:bls.BUILD",
    )

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)
