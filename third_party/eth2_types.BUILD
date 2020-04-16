load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "bls.go",
        "blsprivatekey.go",
        "blspublickey.go",
        "blssignature.go",
        "domain.go",
        "errors.go",
        "privatekey.go",
        "publickey.go",
        "signature.go",
    ],
    importpath = "github.com/wealdtech/go-eth2-types/v2",
    importpath_aliases = ["github.com/wealdtech/go-eth2-types"],
    visibility = ["//visibility:public"],
    deps = [
        "@herumi_bls_eth_go_binary//:go_default_library",
        "@com_github_pkg_errors//:go_default_library",
        "@com_github_prysmaticlabs_go_ssz//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "blsprivatekey_test.go",
        "blspublickey_test.go",
        "blssignature_test.go",
        "domain_test.go",
    ],
    embed = [":go_default_library"],
    deps = [
        "@com_github_stretchr_testify//assert:go_default_library",
        "@com_github_stretchr_testify//require:go_default_library",
    ],
)
