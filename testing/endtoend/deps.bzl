load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")  # gazelle:keep

lighthouse_version = "v2.1.2"
lighthouse_archive_name = "lighthouse-%s-x86_64-unknown-linux-gnu-portable.tar.gz" % lighthouse_version

def e2e_deps():
    http_archive(
        name = "web3signer",
        urls = ["https://artifacts.consensys.net/public/web3signer/raw/names/web3signer.tar.gz/versions/21.10.6/web3signer-21.10.6.tar.gz"],
        sha256 = "3f08a262da442e50eef31bb3d6c2e70d9457da8c4e4a8647ae310a17295b886b",
        build_file = "@prysm//testing/endtoend:web3signer.BUILD",
        strip_prefix = "web3signer-21.10.6",
    )

    http_archive(
        name = "lighthouse",
        sha256 = "8a83ba0f7c24cc4e5b588e7a09bb4e5a1f919346ccf7000d3409a3690a85b221",
        build_file = "@prysm//testing/endtoend:lighthouse.BUILD",
        url = ("https://github.com/sigp/lighthouse/releases/download/%s/" + lighthouse_archive_name) % lighthouse_version,
    )
