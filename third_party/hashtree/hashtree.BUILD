load("@prysm//tools/go:def.bzl", "go_library")
load("@io_bazel_rules_go//go:def.bzl", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "bindings.go",
	"sha256_1_generic.go",
	"bindings_amd64.go",
	"bindings_arm64.go",
        "wrapper_arm64.s",
        "wrapper_linux_amd64.s",
        "wrapper_windows_amd64.s",
    ],
  deps = select({
    "@io_bazel_rules_go//go/platform:amd64": ["@com_github_klauspost_cpuid_v2//:cpuid"], 
   "@io_bazel_rules_go//go/platform:arm64": ["@com_github_klauspost_cpuid_v2//:cpuid"], 
   "//conditions:default": []
  }),
    cgo = True,
    copts = [
        "-g -Wall -Werror -fpic",
        "-O2",
	"-Isrc",
    ],
    cdeps = [":hashtree"],
    importpath = "github.com/prysmaticlabs/hashtree",
    visibility = ["//visibility:public"],
)

go_test(
	name = "go_default_test",
	srcs = [
		"bindings_test.go",
	],
	embed = [":go_default_library"],
)

cc_library(
    name = "hashtree",
    srcs = [
        "src/hashtree.c",
        "src/sha256_sse_x1.S",
        "src/sha256_avx_x1.S",
        "src/sha256_avx_x4.S",
        "src/sha256_avx_x8.S",
        "src/sha256_avx_x16.S",
        "src/sha256_shani.S",
        "src/sha256_armv8_crypto.S",
        "src/sha256_armv8_neon_x1.S",
        "src/sha256_armv8_neon_x4.S",
    ],
    hdrs = [
        "src/hashtree.h",
    ],
    copts = [
        "-g -Wall -Werror -fpic",
        "-O2",
	"-Isrc",
        "-fno-integrated-as",
    ],
    visibility = ["//visibility:public"],
    linkstatic = True,
)
