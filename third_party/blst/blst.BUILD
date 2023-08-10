load("@prysm//tools/go:def.bzl", "go_library")
load("@io_bazel_rules_go//go:def.bzl", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "bindings/go/blst.go",
        "bindings/go/cgo_server.c",
    ],
    cgo = True,
    copts = [
        "-D__BLST_CGO__",
        "-D__BLST_PORTABLE__",
        "-Ibindings",
        "-Isrc",
        "-O2",
    ] + select({
        "@io_bazel_rules_go//go/platform:amd64": [
            "-mno-avx",
        ],
        "//conditions:default": [],
    }),
    cdeps = [":blst"],
    importpath = "github.com/supranational/blst/bindings/go",
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = [
        "bindings/go/blst_htoc_test.go",
        "bindings/go/blst_minpk_test.go",
        "bindings/go/blst_minsig_test.go",
    ],
    embed = [":go_default_library"],
    data = glob([
        "bindings/go/hash_to_curve/*.json",
    ]),
)

cc_library(
    name = "blst",
    srcs = [
        "bindings/blst.h",
        "bindings/blst_aux.h",
    ],
    hdrs = [
        "bindings/blst.h",
        "bindings/blst_aux.h",
    ],
    deps = [
        ":src",
        ":asm",
    ],
    strip_include_prefix = "bindings",
    visibility = ["//visibility:public"],
)

cc_library(
    name = "asm_hdrs",
    hdrs = glob([
        "build/**/*.s",
        "build/**/*.S",
    ], exclude = ["build/assembly.s"]),
)

cc_library(
    name = "asm",
    srcs = [
        "build/assembly.S",
    ],
    copts = [
            "-D__BLST_PORTABLE__",
            "-O2",
    ] + select({
        "@io_bazel_rules_go//go/platform:amd64": [
            "-mno-avx",
        ],
        "//conditions:default": [],
    }),
    deps = [":asm_hdrs"],
    linkstatic = True,
)

cc_library(
    name = "hdrs",
    hdrs = glob(
        [
            "src/*.c",
            "src/*.h",
        ],
        exclude = [
            "src/client_*.c",
        ],
    ),
    strip_include_prefix = "src",
)

cc_library(
    name = "src",
    srcs = [
        "src/server.c",
    ],
    deps = [
        ":hdrs",
    ],
)
