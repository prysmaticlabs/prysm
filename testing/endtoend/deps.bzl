load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")  # gazelle:keep

lighthouse_version = "v4.0.1"
lighthouse_archive_name = "lighthouse-%s-x86_64-unknown-linux-gnu-portable.tar.gz" % lighthouse_version

mev_rs_version = "v0.3.0"
mev_rs_archive_name = "mev-rs-%s-x86_64-unknown-linux-gnu.tar.gz" % mev_rs_version

def e2e_deps():
    http_archive(
        name = "web3signer",
        urls = ["https://artifacts.consensys.net/public/web3signer/raw/names/web3signer.tar.gz/versions/23.3.1/web3signer-23.3.1.tar.gz"],
        sha256 = "32dfbfd8d5900f19aa426d3519724dd696e6529b7ec2f99e0cb1690dae52b3d6",
        build_file = "@prysm//testing/endtoend:web3signer.BUILD",
        strip_prefix = "web3signer-23.3.1",
    )

    http_archive(
        name = "lighthouse",
        sha256 = "bb41eaa2f01b1231c1a8b24f1b6296c134c654ecc2b24c7f2c877f97420503f1",
        build_file = "@prysm//testing/endtoend:lighthouse.BUILD",
        url = ("https://github.com/sigp/lighthouse/releases/download/%s/" + lighthouse_archive_name) % lighthouse_version,
    )

    http_archive(
        name = "mev_rs",
        sha256 = "db11a2ea984693c409d8283dcfb981bbbc6104d415eb0ee5e2bb133474cc4f4f",
        build_file = "@prysm//testing/endtoend:mev_rs.BUILD",
        #   url = ("https://github.com/ralexstokes/mev-rs/releases/download/%s/" + mev_rs_archive_name) % mev_rs_version,
        # This is a compiled version with an older version of libc so that it can run on our infrastructure.
        url = "https://prysmaticlabs.com/uploads/misc/mev-rs-9cf220.tar.gz",
    )
