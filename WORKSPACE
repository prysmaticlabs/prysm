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
    sha256 = "b210fc8e58782ef171f428bfc850ed7179bdd805543ebd1aa144b9c93489134f",
    strip_prefix = "bazel-toolchain-83e69ba9e4b4fdad0d1d057fcb87addf77c281c9",
    urls = ["https://github.com/grailbio/bazel-toolchain/archive/83e69ba9e4b4fdad0d1d057fcb87addf77c281c9.tar.gz"],
)

load("@com_grail_bazel_toolchain//toolchain:deps.bzl", "bazel_toolchain_dependencies")

bazel_toolchain_dependencies()

load("@com_grail_bazel_toolchain//toolchain:rules.bzl", "llvm_toolchain")

llvm_toolchain(
    name = "llvm_toolchain",
    llvm_version = "13.0.1",
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
    sha256 = "5982e5463f171da99e3bdaeff8c0f48283a7a5f396ec5282910b9e8a49c0dd7e",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.25.0/bazel-gazelle-v0.25.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.25.0/bazel-gazelle-v0.25.0.tar.gz",
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
    name = "io_bazel_rules_go",
    patch_args = ["-p1"],
    patches = [
        # Expose internals of go_test for custom build transitions.
        "//third_party:io_bazel_rules_go_test.patch",
    ],
    sha256 = "ae013bf35bd23234d1dea46b079f1e05ba74ac0321423830119d3e787ec73483",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.36.0/rules_go-v0.36.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.36.0/rules_go-v0.36.0.zip",
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

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(
    go_version = "1.19.4",
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

consensus_spec_version = "v1.3.0-rc.3"

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
    sha256 = "c56ea4e8fb2fea4b0b23d2191a87bc7e676738dd8a623b44ac847bfeaae5fe64",
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
    sha256 = "c97730b372b81e9ee698c4f09eafaec3fb4be177fab323b32e5ef78d0873fb5c",
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
    sha256 = "bdd8255c8a536fa83366144aa32fcc3df92a4997401f88f7cda934d13e90d11c",
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
    sha256 = "f6539e19b8e4e45f45b80da39c87dfe7c61d9d7cb51fa7a3a36bdaca11a89693",
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
    sha256 = "5006614c33e358699b4e072c649cd4c3866f7d41a691449d5156f6c6e07a4c60",
    urls = [
        "https://github.com/prysmaticlabs/prysm-web-ui/releases/download/v2.0.3/prysm-web-ui.tar.gz",
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
