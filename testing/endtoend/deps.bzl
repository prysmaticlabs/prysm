load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")  # gazelle:keep

lighthouse_version = "v2.1.2"
lighthouse_archive_name = "lighthouse-%s-x86_64-unknown-linux-gnu-portable.tar.gz" % lighthouse_version

def e2e_deps():
    http_archive(
        name = "web3signer",
        # Built from commit 196462 part of an in progress PR:
        # https://github.com/ConsenSys/web3signer/pull/515
        urls = ["https://prysmaticlabs.com/uploads/web3signer-17d253b.tar.gz"],
        sha256 = "bf450a59a0845c1ce8100b3192c7fec021b565efe8b1ab46bed9f71cb994a6d7",
        build_file = "@prysm//testing/endtoend:web3signer.BUILD",
        strip_prefix = "web3signer-develop",
    )

    http_archive(
        name = "lighthouse",
        sha256 = "8a83ba0f7c24cc4e5b588e7a09bb4e5a1f919346ccf7000d3409a3690a85b221",
        build_file = "@prysm//testing/endtoend:lighthouse.BUILD",
        url = ("https://github.com/sigp/lighthouse/releases/download/%s/" + lighthouse_archive_name) % lighthouse_version,
    )
