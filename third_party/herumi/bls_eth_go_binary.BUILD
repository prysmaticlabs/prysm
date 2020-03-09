load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

OPTS = [
    "-DMCL_USE_VINT",
    "-DMCL_DONT_USE_OPENSSL",
    "-DMCL_LLVM_BMI2=0",
    "-DMCL_USE_LLVM=1",
    "-DMCL_VINT_FIXED_BUFFER",
    "-DMCL_SIZEOF_UNIT=8",
    "-DMCL_MAX_BIT_SIZE=384",
    "-DCYBOZU_DONT_USE_EXCEPTION",
    "-DCYBOZU_DONT_USE_STRING",
    "-std=c++03 ",
    "-DBLS_SWAP_G",
    "-DBLS_ETH",
]

genrule(
    name = "base64_ll",
    outs = ["src/base64.ll"],
    tools = [
        "@herumi_mcl//:src_gen",
    ],
    # TODO: func.list is different based on LOW_ASM_SRC.
    cmd = "touch func.list && $(location @herumi_mcl//:src_gen) -u 64 -f func.list > $@",
)

genrule(
    name = "base64_o",
    srcs = [
        "src/base64.ll",
    ],
    outs = ["base64.o"],
    # TODO: Should use toolchain to provide clang++.
    cmd = "clang++ -c -o $@ $(location src/base64.ll) -std=c++03 -O3 -DNDEBUG -DMCL_DONT_USE_OPENSSL -DMCL_LLVM_BMI2=0 -DMCL_USE_LLVM=1 -DMCL_USE_VINT -DMCL_SIZEOF_UNIT=8 -DMCL_VINT_FIXED_BUFFER -DMCL_MAX_BIT_SIZE=384 -DCYBOZU_DONT_USE_EXCEPTION -DCYBOZU_DONT_USE_STRING",
    tools = ["@bazel_tools//tools/cpp:current_cc_toolchain"],
)

cc_library(
    name = "lib",
    srcs = [
        "@herumi_mcl//:src/fp.cpp",
        "@herumi_bls//:src/bls_c384_256.cpp",
        "@herumi_bls//:src/bls_c_impl.hpp",
        ":base64_o",
    ],
    deps = ["@herumi_mcl//:bn"],
    includes = [
        "bls/include",
    ],
    hdrs = [
        "bls/include/bls/bls.h",
        "bls/include/bls/bls384_256.h",
        "bls/include/mcl/bn.h",
        "bls/include/mcl/bn_c384_256.h",
        "@herumi_mcl//:include/mcl/curve_type.h",
    ],
    copts = OPTS,
)

# TODO: integrate @nisdas patch for better serialization alloc.
go_library(
    name = "go_default_library",
    importpath = "github.com/herumi/bls-eth-go-binary/bls",
    srcs = [
        "bls/bls.go",
        "bls/callback.go",
        "bls/cast.go",
        "bls/mcl.go",
    ],
    cdeps = [
        # TODO: Select pre-compiled archives when clang is not available.
        ":lib",
    ],
    copts = OPTS,
    cgo = True,
    visibility = ["//visibility:public"],
)
