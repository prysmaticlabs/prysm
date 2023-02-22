load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")  # gazelle:keep

lighthouse_version = "v3.1.2"
lighthouse_archive_name = "lighthouse-%s-x86_64-unknown-linux-gnu-portable.tar.gz" % lighthouse_version

def e2e_deps():
    http_archive(
        name = "web3signer",
        urls = ["https://artifacts.consensys.net/public/web3signer/raw/names/web3signer.tar.gz/versions/22.8.1/web3signer-22.8.1.tar.gz"],
        sha256 = "ec888222484c4d1b6203bd6d248890adf713f8bf47fb362fb36e8d47a98cb401",
        build_file = "@prysm//testing/endtoend:web3signer.BUILD",
        strip_prefix = "web3signer-22.8.1",
    )

    http_archive(
        name = "lighthouse",
        sha256 = "469d800ca8ed1e82af288d730d0e9f3e1e054fe1fe7262ab0964d315d1a15020",
        build_file = "@prysm//testing/endtoend:lighthouse.BUILD",
        #   url = ("https://github.com/sigp/lighthouse/releases/download/%s/" + lighthouse_archive_name) % lighthouse_version,
        # This is a compiled version of lighthouse from their `capella` branch at this commit
        # https://github.com/sigp/lighthouse/commit/10d32ee04c416200205a051724daafb76ae2bc50. Lighthouse does not have support
        # for all the capella features as of their latest release, so this is a temporary compromise to allow multiclient test
        # runs till their official release includes the required capella features in.
        url = "https://prysmaticlabs.com/uploads/misc/lighthouse-10d32e.tar.gz",
    )
