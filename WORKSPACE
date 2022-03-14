workspace(name = "prysm")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

http_archive(
    name = "bazel_toolchains",
    sha256 = "8e0633dfb59f704594f19ae996a35650747adc621ada5e8b9fb588f808c89cb0",
    strip_prefix = "bazel-toolchains-3.7.0",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-toolchains/releases/download/3.7.0/bazel-toolchains-3.7.0.tar.gz",
        "https://github.com/bazelbuild/bazel-toolchains/releases/download/3.7.0/bazel-toolchains-3.7.0.tar.gz",
    ],
)

http_archive(
    name = "com_grail_bazel_toolchain",
    sha256 = "040b9d00b8a03e8a28e38159ad0f2d0e0de625d93f453a9f226971a8c47e757b",
    strip_prefix = "bazel-toolchain-5f82830f9d6a1941c3eb29683c1864ccf2862454",
    urls = ["https://github.com/grailbio/bazel-toolchain/archive/5f82830f9d6a1941c3eb29683c1864ccf2862454.tar.gz"],
)

load("@com_grail_bazel_toolchain//toolchain:deps.bzl", "bazel_toolchain_dependencies")

bazel_toolchain_dependencies()

load("@com_grail_bazel_toolchain//toolchain:rules.bzl", "llvm_toolchain")

llvm_toolchain(
    name = "llvm_toolchain",
    llvm_version = "10.0.0",
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
    sha256 = "1c531376ac7e5a180e0237938a2536de0c54d93f5c278634818e0efc952dd56c",
    urls = [
        "https://github.com/bazelbuild/bazel-skylib/releases/download/1.0.3/bazel-skylib-1.0.3.tar.gz",
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.0.3/bazel-skylib-1.0.3.tar.gz",
    ],
)

load("@bazel_skylib//:workspace.bzl", "bazel_skylib_workspace")

bazel_skylib_workspace()

http_archive(
    name = "bazel_gazelle",
    sha256 = "de69a09dc70417580aabf20a28619bb3ef60d038470c7cf8442fafcf627c21cb",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.24.0/bazel-gazelle-v0.24.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.24.0/bazel-gazelle-v0.24.0.tar.gz",
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
    sha256 = "1f4e59843b61981a96835dc4ac377ad4da9f8c334ebe5e0bb3f58f80c09735f4",
    strip_prefix = "rules_docker-0.19.0",
    urls = ["https://github.com/bazelbuild/rules_docker/releases/download/v0.19.0/rules_docker-v0.19.0.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_go",
    patch_args = ["-p1"],
    patches = [
        # Expose internals of go_test for custom build transitions.
        "//third_party:io_bazel_rules_go_test.patch",
    ],
    sha256 = "2b1641428dff9018f9e85c0384f03ec6c10660d935b750e3fa1492a281a53b0f",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.29.0/rules_go-v0.29.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.29.0/rules_go-v0.29.0.zip",
    ],
)

# Override default import in rules_go with special patch until
# https://github.com/gogo/protobuf/pull/582 is merged.
git_repository(
    name = "com_github_gogo_protobuf",
    commit = "b03c65ea87cdc3521ede29f62fe3ce239267c1bc",
    patch_args = ["-p1"],
    patches = [
        "@io_bazel_rules_go//third_party:com_github_gogo_protobuf-gazelle.patch",
        "//third_party:com_github_gogo_protobuf-equal.patch",
    ],
    remote = "https://github.com/gogo/protobuf",
    shallow_since = "1610265707 +0000",
    # gazelle args: -go_prefix github.com/gogo/protobuf -proto legacy
)

http_archive(
    name = "fuzzit_linux",
    build_file_content = "exports_files([\"fuzzit\"])",
    sha256 = "9ca76ac1c22d9360936006efddf992977ebf8e4788ded8e5f9d511285c9ac774",
    urls = ["https://github.com/fuzzitdev/fuzzit/releases/download/v2.4.76/fuzzit_Linux_x86_64.zip"],
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
    name = "cc_image_base",
    digest = "sha256:2c4bb6b7236db0a55ec54ba8845e4031f5db2be957ac61867872bf42e56c4deb",
    registry = "gcr.io",
    repository = "distroless/cc",
)

container_pull(
    name = "cc_debug_image_base",
    digest = "sha256:3680c61e81f68fc00bfb5e1ec65e8e678aaafa7c5f056bc2681c29527ebbb30c",
    registry = "gcr.io",
    repository = "distroless/cc",
)

container_pull(
    name = "go_image_base",
    digest = "sha256:ba7a315f86771332e76fa9c3d423ecfdbb8265879c6f1c264d6fff7d4fa460a4",
    registry = "gcr.io",
    repository = "distroless/base",
)

container_pull(
    name = "go_debug_image_base",
    digest = "sha256:efd8711717d9e9b5d0dbb20ea10876dab0609c923bc05321b912f9239090ca80",
    registry = "gcr.io",
    repository = "distroless/base",
)

container_pull(
    name = "alpine_cc_linux_amd64",
    digest = "sha256:752aa0c9a88461ffc50c5267bb7497ef03a303e38b2c8f7f2ded9bebe5f1f00e",
    registry = "index.docker.io",
    repository = "pinglamb/alpine-glibc",
)

container_pull(
    name = "fuzzit_base",
    digest = "sha256:24a39a4360b07b8f0121eb55674a2e757ab09f0baff5569332fefd227ee4338f",
    registry = "gcr.io",
    repository = "fuzzit-public/stretch-llvm8",
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(
    go_version = "1.17.6",
    nogo = "@//:nogo",
)

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
    name = "eip3076_spec_tests",
    build_file_content = """
filegroup(
    name = "test_data",
    srcs = glob([
        "**/*.json",
    ]),
    visibility = ["//visibility:public"],
)
    """,
    sha256 = "91434d5fd5e1c6eb7b0174fed2afe25e09bddf00e1e4c431db931b2cee4e7773",
    url = "https://github.com/eth-clients/slashing-protection-interchange-tests/archive/b8413ca42dc92308019d0d4db52c87e9e125c4e9.tar.gz",
)

consensus_spec_version = "v1.1.10"

bls_test_version = "v0.1.1"

http_archive(
    name = "consensus_spec_tests_general",
    build_file_content = """
filegroup(
    name = "test_data",
    srcs = glob([
        "**/*.ssz_snappy",
        "**/*.yaml",
    ]),
    visibility = ["//visibility:public"],
)
    """,
    sha256 = "28043009cc2f6fc9804e73c8c1fc2cb27062f1591e6884f3015ae1dd7a276883",
    url = "https://github.com/ethereum/consensus-spec-tests/releases/download/%s/general.tar.gz" % consensus_spec_version,
)

http_archive(
    name = "consensus_spec_tests_minimal",
    build_file_content = """
filegroup(
    name = "test_data",
    srcs = glob([
        "**/*.ssz_snappy",
        "**/*.yaml",
    ]),
    visibility = ["//visibility:public"],
)
    """,
    sha256 = "bc1a283ca068f310f04d70c4f6a8eaa0b8f7e9318073a8bdc2ee233111b4e339",
    url = "https://github.com/ethereum/consensus-spec-tests/releases/download/%s/minimal.tar.gz" % consensus_spec_version,
)

http_archive(
    name = "consensus_spec_tests_mainnet",
    build_file_content = """
filegroup(
    name = "test_data",
    srcs = glob([
        "**/*.ssz_snappy",
        "**/*.yaml",
    ]),
    visibility = ["//visibility:public"],
)
    """,
    sha256 = "bbabb482c229ff9d4e2c7b77c992edb452f9d0af7c6d8dd4f922f06a7b101e81",
    url = "https://github.com/ethereum/consensus-spec-tests/releases/download/%s/mainnet.tar.gz" % consensus_spec_version,
)

http_archive(
    name = "consensus_spec",
    build_file_content = """
filegroup(
    name = "spec_data",
    srcs = glob([
        "**/*.yaml",
    ]),
    visibility = ["//visibility:public"],
)
    """,
    sha256 = "408a5524548ad3fcf387f65ac7ec52781d9ee899499720bb12451b48a15818d4",
    strip_prefix = "consensus-specs-" + consensus_spec_version[1:],
    url = "https://github.com/ethereum/consensus-specs/archive/refs/tags/%s.tar.gz" % consensus_spec_version,
)

http_archive(
    name = "bls_spec_tests",
    build_file_content = """
filegroup(
    name = "test_data",
    srcs = glob([
        "**/*.yaml",
    ]),
    visibility = ["//visibility:public"],
)
    """,
    sha256 = "93c7d006e7c5b882cbd11dc9ec6c5d0e07f4a8c6b27a32f964eb17cf2db9763a",
    url = "https://github.com/ethereum/bls12-381-tests/releases/download/%s/bls_tests_yaml.tar.gz" % bls_test_version,
)

http_archive(
    name = "eth2_networks",
    build_file_content = """
filegroup(
    name = "configs",
    srcs = glob([
        "shared/**/config.yaml",
    ]),
    visibility = ["//visibility:public"],
)
    """,
    sha256 = "4e8a18b21d056c4032605621b1a6632198eabab57cb90c61e273f344c287f1b2",
    strip_prefix = "eth2-networks-791a5369c5981e829698b17fbcdcdacbdaba97c8",
    url = "https://github.com/eth-clients/eth2-networks/archive/791a5369c5981e829698b17fbcdcdacbdaba97c8.tar.gz",
)

http_archive(
    name = "com_github_bazelbuild_buildtools",
    sha256 = "7a182df18df1debabd9e36ae07c8edfa1378b8424a04561b674d933b965372b3",
    strip_prefix = "buildtools-f2aed9ee205d62d45c55cfabbfd26342f8526862",
    url = "https://github.com/bazelbuild/buildtools/archive/f2aed9ee205d62d45c55cfabbfd26342f8526862.zip",
)

git_repository(
    name = "com_google_protobuf",
    commit = "436bd7880e458532901c58f4d9d1ea23fa7edd52",
    remote = "https://github.com/protocolbuffers/protobuf",
    shallow_since = "1617835118 -0700",
)

# Group the sources of the library so that CMake rule have access to it
all_content = """filegroup(name = "all", srcs = glob(["**"]), visibility = ["//visibility:public"])"""

# External dependencies

http_archive(
    name = "prysm_web_ui",
    build_file_content = """
filegroup(
    name = "site",
    srcs = glob(["**/*"]),
    visibility = ["//visibility:public"],
)
""",
    sha256 = "4797a7e594a5b1f4c1c8080701613f3ee451b01ec0861499ea7d9b60877a6b23",
    urls = [
        "https://github.com/prysmaticlabs/prysm-web-ui/releases/download/v1.0.3/prysm-web-ui.tar.gz",
    ],
)

load("//:deps.bzl", "prysm_deps")

# gazelle:repository_macro deps.bzl%prysm_deps
prysm_deps()

load("@prysm//third_party/herumi:herumi.bzl", "bls_dependencies")

bls_dependencies()

load("@prysm//testing/endtoend:deps.bzl", "e2e_deps")

e2e_deps()

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

load("@io_bazel_rules_go//extras:embed_data_deps.bzl", "go_embed_data_dependencies")

go_embed_data_dependencies()

load("@com_github_atlassian_bazel_tools//gometalinter:deps.bzl", "gometalinter_dependencies")

gometalinter_dependencies()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")

buildifier_dependencies()

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

# Do NOT add new go dependencies here! Refer to DEPENDENCIES.md!
