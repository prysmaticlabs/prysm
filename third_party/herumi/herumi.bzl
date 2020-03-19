load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

"""
Herumi's BLS library for go depends on
- herumi/mcl
- herumi/bls
- herumi/bls-eth-go-binary
"""

def bls_dependencies():
    # TODO(4804): Update herumi_bls_eth_go_binary and herumi_bls to latest supporting v0.10.0.
    _maybe(
        http_archive,
        name = "herumi_bls_eth_go_binary",
        strip_prefix = "bls-eth-go-binary-147ed25f233ed0b211e711ed8271606540c58064",
        urls = [
            "https://github.com/herumi/bls-eth-go-binary/archive/147ed25f233ed0b211e711ed8271606540c58064.tar.gz",
        ],
        sha256 = "bbd04f3354f12982e4ef32c62eb13ceb183303ada1ee69e2869553ed35134321",
        build_file = "@prysm//third_party/herumi:bls_eth_go_binary.BUILD",
        # TODO(4804): Delete this patch after updating this archive to commit 381c62473c28af84f424cfb1521c97e48289174a or later.
        patches = [
            "@prysm//third_party/herumi:bls_eth_go_binary_serialization_alloc_fix.patch",  # Integrates changes from PR #5.
        ],
        patch_args = ["-p1"],
    )
    _maybe(
        http_archive,
        name = "herumi_mcl",
        strip_prefix = "mcl-1b043ade54bf7e30b8edc29eb01410746ba92d3d",
        urls = [
            "https://github.com/herumi/mcl/archive/1b043ade54bf7e30b8edc29eb01410746ba92d3d.tar.gz",
        ],
        sha256 = "306bf22b747db174390bbe43de503131b0b5b75bbe586d44f3465c16bda8d28a",
        build_file = "@prysm//third_party/herumi:mcl.BUILD",
    )
    _maybe(
        http_archive,
        name = "herumi_bls",
        strip_prefix = "bls-b0e010004293a7ffd2a626edc2062950abd09938",
        urls = [
            "https://github.com/herumi/bls/archive/b0e010004293a7ffd2a626edc2062950abd09938.tar.gz",
        ],
        sha256 = "c7300970c8a639cbbe7465d10f412d6c6ab162b15f2e184b191c9763c2241da4",
        build_file = "@prysm//third_party/herumi:bls.BUILD",
    )

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)
