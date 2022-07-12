load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")  # gazelle:keep

lighthouse_version = "v2.3.0"
lighthouse_archive_name = "lighthouse-%s-x86_64-unknown-linux-gnu-portable.tar.gz" % lighthouse_version

def e2e_deps():
    http_archive(
        name = "web3signer",
        urls = ["https://output.circle-artifacts.com/output/job/4ea56fc6-d9a2-4f4b-9d6a-f3c04ebb4ba7/artifacts/0/distributions/web3signer-develop.tar.gz"],
        sha256 = "7a3268316f5416c779453cead9656980066cdd52fb909a2102154eb75691a425",
        build_file = "@prysm//testing/endtoend:web3signer.BUILD",
        strip_prefix = "web3signer-develop",
    )

    http_archive(
        name = "lighthouse",
        sha256 = "6029acd211f269bcf41f86fd72fe540703a167ff96dfd952ff95b1693e3b5495",
        build_file = "@prysm//testing/endtoend:lighthouse.BUILD",
        url = ("https://github.com/sigp/lighthouse/releases/download/%s/" + lighthouse_archive_name) % lighthouse_version,
    )
