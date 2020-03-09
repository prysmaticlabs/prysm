package(default_visibility = ["//visibility:public"])

MCL_OPTS = [
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
]

cc_library(
    name = "fp",
    srcs = [
        "src/fp.cpp",
        "src/asm/x86-64.s",
    ],
    includes = [
        "include",
    ],
    hdrs = glob([
        "src/xbyak/*.h",
        "include/cybozu/*.hpp",
    ]) + [
        "include/mcl/op.hpp",
        "include/mcl/gmp_util.hpp",
        "include/mcl/vint.hpp",
        "include/mcl/randgen.hpp",
        "include/mcl/array.hpp",
        "include/mcl/util.hpp",
        "include/mcl/fp_tower.hpp",
        "include/mcl/fp.hpp",
        "include/mcl/conversion.hpp",
        "src/low_func.hpp",
        "src/fp_generator.hpp",
        "src/proto.hpp",
        "src/low_func_llvm.hpp",
    ],
    copts = MCL_OPTS,
)

cc_library(
    name = "bn",
    srcs = [
        "src/bn_c384_256.cpp",
    ],
    deps = [":fp"],
    hdrs = [
        "include/mcl/bn.h",
        "include/mcl/curve_type.h",
        "include/mcl/impl/bn_c_impl.hpp",
        "include/mcl/bls12_381.hpp",
        "include/mcl/bn_c384_256.h",
        "include/mcl/ec.hpp",
        "include/mcl/mapto_wb19.hpp",
        "include/mcl/ecparam.hpp",
        "include/mcl/lagrange.hpp",
        "include/mcl/bn.hpp",
        "include/mcl/operator.hpp",
    ],
    includes = ["include"],
    copts = MCL_OPTS,
)

# src_gen is a tool to generate some llvm assembly language file.
cc_binary(
    name = "src_gen",
    srcs = [
        "src/gen.cpp",
        "src/llvm_gen.hpp",
    ] + glob([
        "include/cybozu/*.hpp",
        "include/mcl/*.hpp",
    ]),
    includes = ["include"],
)
