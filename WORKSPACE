workspace(name = "prysm")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

http_archive(
    name = "bazel_toolchains",
    sha256 = "a802b753e127a6f73f3f300db5dd83fb618cd798bc880b6a87db9a8777b7939f",
    strip_prefix = "bazel-toolchains-3.3.0",
    urls = [
        "https://github.com/bazelbuild/bazel-toolchains/releases/download/3.3.0/bazel-toolchains-3.3.0.tar.gz",
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-toolchains/archive/3.3.0.tar.gz",
    ],
)

http_archive(
    name = "com_grail_bazel_toolchain",
    sha256 = "0bec89e35d8a141c87f28cfc506d6d344785c8eb2ff3a453140a1fe972ada79d",
    strip_prefix = "bazel-toolchain-77a87103145f86f03f90475d19c2c8854398a444",
    urls = ["https://github.com/grailbio/bazel-toolchain/archive/77a87103145f86f03f90475d19c2c8854398a444.tar.gz"],
)

load("@com_grail_bazel_toolchain//toolchain:deps.bzl", "bazel_toolchain_dependencies")

bazel_toolchain_dependencies()

load("@com_grail_bazel_toolchain//toolchain:rules.bzl", "llvm_toolchain")

llvm_toolchain(
    name = "llvm_toolchain",
    llvm_version = "9.0.0",
)

load("@llvm_toolchain//:toolchains.bzl", "llvm_register_toolchains")

llvm_register_toolchains()

load("@prysm//tools/cross-toolchain:prysm_toolchains.bzl", "configure_prysm_toolchains")

configure_prysm_toolchains()

load("@prysm//tools/cross-toolchain:rbe_toolchains_config.bzl", "rbe_toolchains_config")

rbe_toolchains_config()

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "bazel_skylib",
    sha256 = "97e70364e9249702246c0e9444bccdc4b847bed1eb03c5a3ece4f83dfe6abc44",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.0.2/bazel-skylib-1.0.2.tar.gz",
        "https://github.com/bazelbuild/bazel-skylib/releases/download/1.0.2/bazel-skylib-1.0.2.tar.gz",
    ],
)

load("@bazel_skylib//:workspace.bzl", "bazel_skylib_workspace")

bazel_skylib_workspace()

http_archive(
    name = "bazel_gazelle",
    sha256 = "cdb02a887a7187ea4d5a27452311a75ed8637379a1287d8eeb952138ea485f7d",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/bazel-gazelle/releases/download/v0.21.1/bazel-gazelle-v0.21.1.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.21.1/bazel-gazelle-v0.21.1.tar.gz",
    ],
)

http_archive(
    name = "com_github_atlassian_bazel_tools",
    sha256 = "60821f298a7399450b51b9020394904bbad477c18718d2ad6c789f231e5b8b45",
    strip_prefix = "bazel-tools-a2138311856f55add11cd7009a5abc8d4fd6f163",
    urls = ["https://github.com/atlassian/bazel-tools/archive/a2138311856f55add11cd7009a5abc8d4fd6f163.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "6287241e033d247e9da5ff705dd6ef526bac39ae82f3d17de1b69f8cb313f9cd",
    strip_prefix = "rules_docker-0.14.3",
    url = "https://github.com/bazelbuild/rules_docker/archive/v0.14.3.tar.gz",
)

http_archive(
    name = "io_bazel_rules_go",
    patch_args = ["-p1"],
    patches = [
        # Required until https://github.com/bazelbuild/rules_go/pull/2450 merges otherwise nilness
        # nogo check fails for certain third_party dependencies.
        "//third_party:io_bazel_rules_go.patch",
    ],
    sha256 = "a8d6b1b354d371a646d2f7927319974e0f9e52f73a2452d2b3877118169eb6bb",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.23.3/rules_go-v0.23.3.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.23.3/rules_go-v0.23.3.tar.gz",
    ],
)

# Override default import in rules_go with special patch until
# https://github.com/gogo/protobuf/pull/582 is merged.
git_repository(
    name = "com_github_gogo_protobuf",
    commit = "5628607bb4c51c3157aacc3a50f0ab707582b805",
    patch_args = ["-p1"],
    patches = [
        "@io_bazel_rules_go//third_party:com_github_gogo_protobuf-gazelle.patch",
        "//third_party:com_github_gogo_protobuf-equal.patch",
    ],
    remote = "https://github.com/gogo/protobuf",
    shallow_since = "1571033717 +0200",
    # gazelle args: -go_prefix github.com/gogo/protobuf -proto legacy
)

http_archive(
    name = "fuzzit_linux",
    build_file_content = "exports_files([\"fuzzit\"])",
    sha256 = "1fdfe9af3f1db1881fd3f544975697a97219fae14c46e913f556e5ad97bba673",
    urls = ["https://github.com/fuzzitdev/fuzzit/releases/download/v2.4.77/fuzzit_Linux_x86_64.zip"],
)

git_repository(
    name = "graknlabs_bazel_distribution",
    commit = "962f3a7e56942430c0ec120c24f9e9f2a9c2ce1a",
    remote = "https://github.com/graknlabs/bazel-distribution",
    shallow_since = "1569509514 +0300",
)

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_pull",
)

container_pull(
    name = "alpine_cc_linux_amd64",
    digest = "sha256:3f7f4dfcb6dceac3a902f36609cc232262e49f5656a6dc4bb3da89e35fecc8a5",
    registry = "index.docker.io",
    repository = "fasibio/alpine-libgcc",
)

container_pull(
    name = "fuzzit_base",
    digest = "sha256:24a39a4360b07b8f0121eb55674a2e757ab09f0baff5569332fefd227ee4338f",
    registry = "gcr.io",
    repository = "fuzzit-public/stretch-llvm8",
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(nogo = "@//:nogo")

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

load("@com_github_atlassian_bazel_tools//gometalinter:deps.bzl", "gometalinter_dependencies")

gometalinter_dependencies()

load(
    "@io_bazel_rules_docker//go:image.bzl",
    _go_image_repos = "repositories",
)

# Golang images
# This is using gcr.io/distroless/base
_go_image_repos()

# CC images
# This is using gcr.io/distroless/base
load(
    "@io_bazel_rules_docker//cc:image.bzl",
    _cc_image_repos = "repositories",
)

_cc_image_repos()

http_archive(
    name = "prysm_testnet_site",
    build_file_content = """
proto_library(
  name = "faucet_proto",
  srcs = ["src/proto/faucet.proto"],
  visibility = ["//visibility:public"],
)""",
    sha256 = "29742136ff9faf47343073c4569a7cf21b8ed138f726929e09e3c38ab83544f7",
    strip_prefix = "prysm-testnet-site-5c711600f0a77fc553b18cf37b880eaffef4afdb",
    url = "https://github.com/prestonvanloon/prysm-testnet-site/archive/5c711600f0a77fc553b18cf37b880eaffef4afdb.tar.gz",
)

http_archive(
    name = "io_kubernetes_build",
    sha256 = "b84fbd1173acee9d02a7d3698ad269fdf4f7aa081e9cecd40e012ad0ad8cfa2a",
    strip_prefix = "repo-infra-6537f2101fb432b679f3d103ee729dd8ac5d30a0",
    url = "https://github.com/kubernetes/repo-infra/archive/6537f2101fb432b679f3d103ee729dd8ac5d30a0.tar.gz",
)

http_archive(
    name = "eth2_spec_tests_general",
    build_file_content = """
filegroup(
    name = "test_data",
    srcs = glob([
        "**/*.ssz",
        "**/*.yaml",
    ]),
    visibility = ["//visibility:public"],
)
    """,
    sha256 = "678c50336ce39bef19b2a0dc69e20a7bda37a673ae07dc0577386e9876e0a525",
    url = "https://github.com/ethereum/eth2.0-spec-tests/releases/download/v0.12.1/general.tar.gz",
)

http_archive(
    name = "eth2_spec_tests_minimal",
    build_file_content = """
filegroup(
    name = "test_data",
    srcs = glob([
        "**/*.ssz",
        "**/*.yaml",
    ]),
    visibility = ["//visibility:public"],
)
    """,
    sha256 = "d0ce95a3ca0d30df24f96a1b5cfad1f7e6fcc07ad84ca221d92480add051af3e",
    url = "https://github.com/ethereum/eth2.0-spec-tests/releases/download/v0.12.1/minimal.tar.gz",
)

http_archive(
    name = "eth2_spec_tests_mainnet",
    build_file_content = """
filegroup(
    name = "test_data",
    srcs = glob([
        "**/*.ssz",
        "**/*.yaml",
    ]),
    visibility = ["//visibility:public"],
)
    """,
    sha256 = "1dfa1ae6822912508dbf6d1fe7608169372daa3ad1e53a3ed0867cb2d6e0ccb0",
    url = "https://github.com/ethereum/eth2.0-spec-tests/releases/download/v0.12.1/mainnet.tar.gz",
)

http_archive(
    name = "com_github_bazelbuild_buildtools",
    sha256 = "b5d7dbc6832f11b6468328a376de05959a1a9e4e9f5622499d3bab509c26b46a",
    strip_prefix = "buildtools-bf564b4925ab5876a3f64d8b90fab7f769013d42",
    url = "https://github.com/bazelbuild/buildtools/archive/bf564b4925ab5876a3f64d8b90fab7f769013d42.zip",
)

load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")

buildifier_dependencies()

git_repository(
    name = "com_google_protobuf",
    commit = "4059c61f27eb1b06c4ee979546a238be792df0a4",
    remote = "https://github.com/protocolbuffers/protobuf",
    shallow_since = "1558721209 -0700",
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

# Group the sources of the library so that CMake rule have access to it
all_content = """filegroup(name = "all", srcs = glob(["**"]), visibility = ["//visibility:public"])"""

http_archive(
    name = "rules_foreign_cc",
    sha256 = "b85ce66a3410f7370d1a9a61dfe3a29c7532b7637caeb2877d8d0dfd41d77abb",
    strip_prefix = "rules_foreign_cc-3515b20a2417c4dd51c8a4a8cac1f6ecf3c6d934",
    url = "https://github.com/bazelbuild/rules_foreign_cc/archive/3515b20a2417c4dd51c8a4a8cac1f6ecf3c6d934.zip",
)

load("@rules_foreign_cc//:workspace_definitions.bzl", "rules_foreign_cc_dependencies")

rules_foreign_cc_dependencies([
    "@prysm//:built_cmake_toolchain",
])

http_archive(
    name = "librdkafka",
    build_file_content = all_content,
    sha256 = "0984ffbe17b9e04599fb9eceb16cfa189f525a042bef02474cd1bbfe1ea68416",
    strip_prefix = "librdkafka-1.4.4",
    urls = ["https://github.com/edenhill/librdkafka/archive/v1.4.4.tar.gz"],
)

http_archive(
    name = "sigp_beacon_fuzz_corpora",
    build_file = "//third_party:beacon-fuzz/corpora.BUILD",
    sha256 = "42993d0901a316afda45b4ba6d53c7c21f30c551dcec290a4ca131c24453d1ef",
    strip_prefix = "beacon-fuzz-corpora-bac24ad78d45cc3664c0172241feac969c1ac29b",
    urls = [
        "https://github.com/sigp/beacon-fuzz-corpora/archive/bac24ad78d45cc3664c0172241feac969c1ac29b.tar.gz",
    ],
)

# External dependencies

http_archive(
    name = "sszgen",  # Hack because we don't want to build this binary with libfuzzer, but need it to build.
    build_file_content = """
load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_binary")

go_library(
    name = "go_default_library",
    srcs = [
        "sszgen/main.go",
        "sszgen/marshal.go",
        "sszgen/size.go",
        "sszgen/unmarshal.go",
    ],
    importpath = "github.com/ferranbt/fastssz/sszgen",
    visibility = ["//visibility:private"],
)

go_binary(
    name = "sszgen",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)
    """,
    strip_prefix = "fastssz-06015a5d84f9e4eefe2c21377ca678fa8f1a1b09",
    urls = ["https://github.com/ferranbt/fastssz/archive/06015a5d84f9e4eefe2c21377ca678fa8f1a1b09.tar.gz"],
)

load("//:deps.bzl", "prysm_deps")

# gazelle:repository_macro deps.bzl%prysm_deps
prysm_deps()

load("@com_github_prysmaticlabs_go_ssz//:deps.bzl", "go_ssz_dependencies")

go_ssz_dependencies()

load("@prysm//third_party/herumi:herumi.bzl", "bls_dependencies")

bls_dependencies()

load("@com_github_ethereum_go_ethereum//:deps.bzl", "geth_dependencies")

geth_dependencies()

# Do NOT add new go dependencies here! Refer to DEPENDENCIES.md!
