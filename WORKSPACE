workspace(name = "prysm")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

http_archive(
    name = "rules_pkg",
    sha256 = "8c20f74bca25d2d442b327ae26768c02cf3c99e93fad0381f32be9aab1967675",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_pkg/releases/download/0.8.1/rules_pkg-0.8.1.tar.gz",
        "https://github.com/bazelbuild/rules_pkg/releases/download/0.8.1/rules_pkg-0.8.1.tar.gz",
    ],
)

load("@rules_pkg//:deps.bzl", "rules_pkg_dependencies")

rules_pkg_dependencies()

http_archive(
    name = "hermetic_cc_toolchain",
    sha256 = "973ab22945b921ef45b8e1d6ce01ca7ce1b8a462167449a36e297438c4ec2755",
    strip_prefix = "hermetic_cc_toolchain-5098046bccc15d2962f3cc8e7e53d6a2a26072dc",
    urls = [
        "https://github.com/uber/hermetic_cc_toolchain/archive/5098046bccc15d2962f3cc8e7e53d6a2a26072dc.tar.gz",  # 2023-06-28
    ],
)

load("@hermetic_cc_toolchain//toolchain:defs.bzl", zig_toolchains = "toolchains")

zig_toolchains()

# Register zig sdk toolchains with support for Ubuntu 20.04 (Focal Fossa) which has an EOL date of April, 2025.
# For ubuntu glibc support, see https://launchpad.net/ubuntu/+source/glibc
register_toolchains(
    "@zig_sdk//toolchain:linux_amd64_gnu.2.31",
    "@zig_sdk//toolchain:linux_arm64_gnu.2.31",
    # Hermetic cc toolchain is not yet supported on darwin. Sysroot needs to be provided.
    # See https://github.com/uber/hermetic_cc_toolchain#osx-sysroot
    #    "@zig_sdk//toolchain:darwin_amd64",
    #    "@zig_sdk//toolchain:darwin_arm64",
    # Windows builds are not supported yet.
    #    "@zig_sdk//toolchain:windows_amd64",
)

load("@prysm//tools/cross-toolchain:darwin_cc_hack.bzl", "configure_nonhermetic_darwin")

configure_nonhermetic_darwin()

load("@prysm//tools/cross-toolchain:prysm_toolchains.bzl", "configure_prysm_toolchains")

configure_prysm_toolchains()

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
    sha256 = "29d5dafc2a5582995488c6735115d1d366fcd6a0fc2e2a153f02988706349825",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.31.0/bazel-gazelle-v0.31.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.31.0/bazel-gazelle-v0.31.0.tar.gz",
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
    sha256 = "b1e80761a8a8243d03ebca8845e9cc1ba6c82ce7c5179ce2b295cd36f7e394bf",
    urls = ["https://github.com/bazelbuild/rules_docker/releases/download/v0.25.0/rules_docker-v0.25.0.tar.gz"],
)

http_archive(
    name = "rules_oci",
    sha256 = "c71c25ed333a4909d2dd77e0b16c39e9912525a98c7fa85144282be8d04ef54c",
    strip_prefix = "rules_oci-1.3.4",
    url = "https://github.com/bazel-contrib/rules_oci/releases/download/v1.3.4/rules_oci-v1.3.4.tar.gz",
)

load("@rules_oci//oci:dependencies.bzl", "rules_oci_dependencies")

rules_oci_dependencies()

load("@rules_oci//oci:repositories.bzl", "LATEST_CRANE_VERSION", "oci_register_toolchains")

oci_register_toolchains(
    name = "oci",
    crane_version = LATEST_CRANE_VERSION,
)

http_archive(
    name = "io_bazel_rules_go",
    patch_args = ["-p1"],
    patches = [
        # Expose internals of go_test for custom build transitions.
        "//third_party:io_bazel_rules_go_test.patch",
    ],
    sha256 = "bfc5ce70b9d1634ae54f4e7b495657a18a04e0d596785f672d35d5f505ab491a",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.40.0/rules_go-v0.40.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.40.0/rules_go-v0.40.0.zip",
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

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_pull",
)

# Pulled gcr.io/distroless/cc-debian11:latest on 2022-02-23
container_pull(
    name = "cc_image_base_amd64",
    digest = "sha256:2a0daf90a7deb78465bfca3ef2eee6e91ce0a5706059f05d79d799a51d339523",
    registry = "gcr.io",
    repository = "distroless/cc-debian11",
)

# Pulled gcr.io/distroless/cc-debian11:debug on 2022-02-23
container_pull(
    name = "cc_debug_image_base_amd64",
    digest = "sha256:7bd596f5f200588f13a69c268eea6ce428b222b67cd7428d6a7fef95e75c052a",
    registry = "gcr.io",
    repository = "distroless/cc-debian11",
)

# Pulled from gcr.io/distroless/base-debian11:latest on 2022-02-23
container_pull(
    name = "go_image_base_amd64",
    digest = "sha256:34e682800774ecbd0954b1663d90238505f1ba5543692dbc75feef7dd4839e90",
    registry = "gcr.io",
    repository = "distroless/base-debian11",
)

# Pulled from gcr.io/distroless/base-debian11:debug on 2022-02-23
container_pull(
    name = "go_debug_image_base_amd64",
    digest = "sha256:0f503c6bfd207793bc416f20a35bf6b75d769a903c48f180ad73f60f7b60d7bd",
    registry = "gcr.io",
    repository = "distroless/base-debian11",
)

container_pull(
    name = "alpine_cc_linux_amd64",
    digest = "sha256:752aa0c9a88461ffc50c5267bb7497ef03a303e38b2c8f7f2ded9bebe5f1f00e",
    registry = "index.docker.io",
    repository = "pinglamb/alpine-glibc",
)

load("@rules_oci//oci:pull.bzl", "oci_pull")

# A multi-arch base image
oci_pull(
    name = "linux_debian11_multiarch_base",  # Debian bullseye
    digest = "sha256:9b8e0854865dcaf49470b4ec305df45957020fbcf17b71eeb50ffd3bc5bf885d",  # 2023-05-17
    image = "gcr.io/distroless/cc-debian11",
    platforms = [
        "linux/amd64",
        "linux/arm64",
    ],
    reproducible = True,
)

load("@prysm//tools:image_deps.bzl", "prysm_image_deps")

prysm_image_deps()

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(
    go_version = "1.20.7",
    nogo = "@//:nogo",
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

http_archive(
    name = "eip4881_spec_tests",
    build_file_content = """
filegroup(
    name = "test_data",
    srcs = glob([
        "**/*.yaml",
    ]),
    visibility = ["//visibility:public"],
)
    """,
    sha256 = "89cb659498c0d196fc9f957f8b849b2e1a5c041c3b2b3ae5432ac5c26944297e",
    url = "https://github.com/ethereum/EIPs/archive/5480440fe51742ed23342b68cf106cefd427e39d.tar.gz",
)

consensus_spec_test_version = "v1.4.0-beta.2-hotfix"

consensus_spec_version = "v1.4.0-beta.2"

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
    sha256 = "99770a001189f66204a4ef79161c8002bcbbcbd8236f1c6479bd5b83a3c68d42",
    url = "https://github.com/ethereum/consensus-spec-tests/releases/download/%s/general.tar.gz" % consensus_spec_test_version,
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
    sha256 = "56763f6492ee137108271007d62feef60d8e3f1698e53dee4bc4b07e55f7326b",
    url = "https://github.com/ethereum/consensus-spec-tests/releases/download/%s/minimal.tar.gz" % consensus_spec_test_version,
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
    sha256 = "bc1cac1a991cdc7426efea14385dcf215df85ed3f0572b824ad6a1d7ca0c89ad",
    url = "https://github.com/ethereum/consensus-spec-tests/releases/download/%s/mainnet.tar.gz" % consensus_spec_test_version,
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
    sha256 = "c5898001aaab2a5bb38a39ff9d17a52f1f9befcc26e63752cbf556040f0c884e",
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
    sha256 = "2701e1e1a3ec10c673fe7dbdbbe6f02c8ae8c922aebbf6e720d8c72d5458aafe",
    strip_prefix = "eth2-networks-7b4897888cebef23801540236f73123e21774954",
    url = "https://github.com/eth-clients/eth2-networks/archive/7b4897888cebef23801540236f73123e21774954.tar.gz",
)

http_archive(
    name = "holesky_testnet",
    build_file_content = """
filegroup(
    name = "configs",
    srcs = [
        "custom_config_data/config.yaml",
    ],
    visibility = ["//visibility:public"],
)
""",
    sha256 = "9f66d8d5644982d3d0d2e3d2b9ebe77a5f96638a5d7fcd715599c32818195cb3",
    strip_prefix = "holesky-ea39b9006210848e13f28d92e12a30548cecd41d",
    url = "https://github.com/eth-clients/holesky/archive/ea39b9006210848e13f28d92e12a30548cecd41d.tar.gz",  # 2023-09-21
)

http_archive(
    name = "com_google_protobuf",
    sha256 = "4e176116949be52b0408dfd24f8925d1eb674a781ae242a75296b17a1c721395",
    strip_prefix = "protobuf-23.3",
    urls = [
        "https://github.com/protocolbuffers/protobuf/archive/v23.3.tar.gz",
    ],
)

# External dependencies
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

load("@com_github_atlassian_bazel_tools//gometalinter:deps.bzl", "gometalinter_dependencies")

gometalinter_dependencies()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

# Do NOT add new go dependencies here! Refer to DEPENDENCIES.md!
