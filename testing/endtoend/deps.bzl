load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")  # gazelle:keep

lighthouse_version = "v4.0.1"
lighthouse_archive_name = "lighthouse-%s-x86_64-unknown-linux-gnu-portable.tar.gz" % lighthouse_version

def e2e_deps():
    http_archive(
        name = "web3signer",
        urls = ["https://artifacts.consensys.net/public/web3signer/raw/names/web3signer.tar.gz/versions/23.9.0/web3signer-23.9.0.tar.gz"],
        sha256 = "7af5cd0589f6105f2267b6c9e6eedda077d597e6410975e1687a6a20e7f1518c",
        build_file = "@prysm//testing/endtoend:web3signer.BUILD",
        strip_prefix = "web3signer-23.3.1",
    )

    http_archive(
        name = "lighthouse",
        sha256 = "bb41eaa2f01b1231c1a8b24f1b6296c134c654ecc2b24c7f2c877f97420503f1",
        build_file = "@prysm//testing/endtoend:lighthouse.BUILD",
        url = ("https://github.com/sigp/lighthouse/releases/download/%s/" + lighthouse_archive_name) % lighthouse_version,
    )
