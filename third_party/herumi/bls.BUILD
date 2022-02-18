package(default_visibility = ["//visibility:public"])

cc_library(
    name = "bls_c384_256",
    srcs = [
        "src/bls_c384_256.cpp",
    ],
    deps = [
        "@herumi_mcl//:bn",
    ],
    hdrs = [
        "include/bls/bls.h",
        "src/bls_c_impl.hpp",
        "src/qcoeff-bn254.hpp",
    ],
    includes = [
        "include",
    ],
    copts = [
        "-DBLS_SWAP_G",
        "-DBLS_ETH",
        "-std=c++03",
    ],
)
