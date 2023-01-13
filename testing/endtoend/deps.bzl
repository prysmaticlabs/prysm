load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")  # gazelle:keep

lighthouse_version = "v3.1.2"
lighthouse_archive_name = "lighthouse-%s-x86_64-unknown-linux-gnu-portable.tar.gz" % lighthouse_version

def e2e_deps():
    http_archive(
        name = "web3signer",
        urls = ["https://artifacts.consensys.net/public/web3signer/raw/names/web3signer.tar.gz/versions/23.1.0/web3signer-23.1.0.tar.gz"],
        sha256 = "47b6ff266d0c99185e4a4a595bc2e0578f0f722fb08baa80df462938f2f8fe74",
        build_file = "@prysm//testing/endtoend:web3signer.BUILD",
        strip_prefix = "web3signer-23.1.0",
    )

    http_archive(
        name = "lighthouse",
        sha256 = "172bb132d5fdc5bd257d5a66e98d0799498f08cb60502f93a5f4437a70d9c5e0",
        build_file = "@prysm//testing/endtoend:lighthouse.BUILD",
        url = ("https://github.com/sigp/lighthouse/releases/download/%s/" + lighthouse_archive_name) % lighthouse_version,
    )
