load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")  # gazelle:keep

lighthouse_version = "v2.1.1"
lighthouse_archive_name = "lighthouse-%s-x86_64-unknown-linux-gnu-portable.tar.gz" % lighthouse_version

def e2e_deps():
    http_archive(
        name = "web3signer",
        urls = ["https://artifacts.consensys.net/public/web3signer/raw/names/web3signer.tar.gz/versions/21.10.5/web3signer-21.10.5.tar.gz"],
        sha256 = "d122429f6a310bc555d1281e0b3f4e3ac43a7beec5e5dcf0a0d2416a5984f461",
        build_file = "@prysm//testing/endtoend:web3signer.BUILD",
        strip_prefix = "web3signer-21.10.5",
    )

    http_archive(
        name = "lighthouse",
        sha256 = "7f34d87a312d03346ea36942586b8d4beeec544a1988b43adbdd1b5f50eacf29",
        build_file = "@prysm//testing/endtoend:lighthouse.BUILD",
        url = ("https://github.com/sigp/lighthouse/releases/download/%s/" + lighthouse_archive_name) % lighthouse_version,
    )
