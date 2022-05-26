load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")  # gazelle:keep

lighthouse_version = "v2.1.4"
lighthouse_archive_name = "lighthouse-%s-x86_64-unknown-linux-gnu-portable.tar.gz" % lighthouse_version

def e2e_deps():
    http_archive(
        name = "web3signer",
        urls = ["https://artifacts.consensys.net/public/web3signer/raw/names/web3signer.tar.gz/versions/22.5.0/web3signer-22.5.0.tar.gz"],
        sha256 = "3a954f5302e424b34acb4bb024f275caf722d8b116c639617f7a2e0f9c9ddc78",
        build_file = "@prysm//testing/endtoend:web3signer.BUILD",
        strip_prefix = "web3signer-22.5.0",
    )

    http_archive(
        name = "lighthouse",
        sha256 = "236883a4827037d96636aa259eef8cf3abc54c795adc18c4c2880842e09c743c",
        build_file = "@prysm//testing/endtoend:lighthouse.BUILD",
        #   url = ("https://github.com/sigp/lighthouse/releases/download/%s/" + lighthouse_archive_name) % lighthouse_version,
        # This is a compiled version of lighthouse from their `unstable` branch at this commit
        # https://github.com/sigp/lighthouse/commit/99bb55472c278a1050f7679b2e018546ad3a28bf. Lighthouse does not have support
        # for all the merge features as of their latest release, so this is a temporary compromise to allow multiclient test
        # runs till their official release includes the required merge features in.
        url = "https://prysmaticlabs.com/uploads/misc/lighthouse-99bb5547.tar.xz",
    )
