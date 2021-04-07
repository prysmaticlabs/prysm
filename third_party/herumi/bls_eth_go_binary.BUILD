load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

config_setting(
    name = "use_gmp",
    values = {"define": "BLS_USE_GMP=true"},
)

config_setting(
    name = "use_openssl",
    values = {"define": "BLS_USE_OPENSSL=true"},
)

OPTS = [
    "-DMCL_LLVM_BMI2=0",
    "-DMCL_USE_LLVM=1",
    "-DMCL_VINT_FIXED_BUFFER",
    "-DMCL_SIZEOF_UNIT=8",
    "-DMCL_MAX_BIT_SIZE=384",
    "-DCYBOZU_DONT_USE_EXCEPTION",
    "-DCYBOZU_DONT_USE_STRING",
    "-DBLS_SWAP_G",
    "-DBLS_ETH",
] + select({
    ":use_gmp": [],
    "//conditions:default": [
        "-DMCL_USE_VINT",
    ],
}) + select({
    ":use_openssl": [],
    "//conditions:default": [
        "-DMCL_DONT_USE_OPENSSL",
    ],
})

cc_library(
    name = "precompiled",
    srcs = select({
        "@io_bazel_rules_go//go/platform:android_arm": [
            "bls/lib/android/armeabi-v7a/libbls384_256.a",
        ],
        "@io_bazel_rules_go//go/platform:linux_arm64": [
            "bls/lib/linux/arm64/libbls384_256.a",
        ],
        "@io_bazel_rules_go//go/platform:android_arm64": [
            "bls/lib/android/arm64-v8a/libbls384_256.a",
        ],
        "@io_bazel_rules_go//go/platform:darwin_amd64": [
            "bls/lib/darwin/amd64/libbls384_256.a",
        ],
        "@io_bazel_rules_go//go/platform:linux_amd64": [
            "bls/lib/linux/amd64/libbls384_256.a",
        ],
        "@io_bazel_rules_go//go/platform:windows_amd64": [
            "bls/lib/windows/amd64/libbls384_256.a",
        ],
        "//conditions:default": [],
    }),
    hdrs = [
        "bls/include/bls/bls.h",
        "bls/include/bls/bls384_256.h",
        "bls/include/mcl/bn.h",
        "bls/include/mcl/bn_c384_256.h",
        "bls/include/mcl/curve_type.h",
    ],
    includes = [
        "bls/include",
    ],
    deprecation = "Using precompiled BLS archives. To build BLS from source with llvm, use --config=llvm.",
)

go_library(
    name = "go_default_library",
    importpath = "github.com/herumi/bls-eth-go-binary/bls",
    srcs = [
        "bls/bls.go",
        "bls/callback.go",
        "bls/cast.go",
        "bls/mcl.go",
    ],
    cdeps = select({
        "//conditions:default": [":precompiled"],
    }),
    cgo = True,
    copts = OPTS,
    visibility = [
        # Additional access will require security approval.
        "@com_github_wealdtech_go_eth2_types_v2//:__pkg__",
    ],
    clinkopts = select({
        "@prysm//fuzz:fuzzing_enabled": ["-Wl,--unresolved-symbols=ignore-all", "-fsanitize=address"],
        "//conditions:default": [],
    }),
)
