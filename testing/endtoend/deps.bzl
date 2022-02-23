load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")  # gazelle:keep

lighthouse_version = "v2.1.2"
lighthouse_archive_name = "lighthouse-%s-x86_64-unknown-linux-gnu-portable.tar.gz" % lighthouse_version

def e2e_deps():
    http_archive(
        name = "web3signer",
        # Built from commit 196462 part of an in progress PR:
        # https://github.com/ConsenSys/web3signer/pull/515
	    urls = ["https://prysmaticlabs.com/uploads/web3signer-196462.tar.gz"],
	    sha256 = "c7af370044afaf8cb096c852f600600c674669da3b0547900cf4d38a3aca52bb",
        build_file = "@prysm//testing/endtoend:web3signer.BUILD",
	    strip_prefix = "web3signer-develop",
    )

    http_archive(
        name = "lighthouse",
        sha256 = "8a83ba0f7c24cc4e5b588e7a09bb4e5a1f919346ccf7000d3409a3690a85b221",
        build_file = "@prysm//testing/endtoend:lighthouse.BUILD",
        url = ("https://github.com/sigp/lighthouse/releases/download/%s/" + lighthouse_archive_name) % lighthouse_version,
    )
