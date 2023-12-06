load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")  # gazelle:keep

lighthouse_version = "v4.5.0"
lighthouse_archive_name = "lighthouse-%s-x86_64-unknown-linux-gnu-portable.tar.gz" % lighthouse_version

def e2e_deps():
    http_archive(
        name = "web3signer",
        urls = ["https://artifacts.consensys.net/public/web3signer/raw/names/web3signer.tar.gz/versions/23.9.1/web3signer-23.9.1.tar.gz"],
        sha256 = "aec9dc745cb25fd8d7b38b06e435e3138972c2cf842dd6f851d50be7bf081629",
        build_file = "@prysm//testing/endtoend:web3signer.BUILD",
        strip_prefix = "web3signer-23.9.1",
    )

    http_archive(
        name = "lighthouse",
        sha256 = "1e91ecab827649ac8ea0cfbb11ee2fb159cecd6ac5125e56dd27004225b128c9",
        build_file = "@prysm//testing/endtoend:lighthouse.BUILD",
        url = ("https://github.com/sigp/lighthouse/releases/download/%s/" + lighthouse_archive_name) % lighthouse_version,
    )
