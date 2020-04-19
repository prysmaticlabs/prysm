load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "hid_disabled.go",
        "raw_disabled.go",
        "usb.go",
        "usb_disabled.go",
    ],
     importpath = "github.com/karalabe/usb",
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = [
        "usb_test.go",
    ],
    embed = [":go_default_library"],
)
