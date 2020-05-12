workspace(name = "prysm")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

http_archive(
    name = "bazel_toolchains",
    sha256 = "144290c4166bd67e76a54f96cd504ed86416ca3ca82030282760f0823c10be48",
    strip_prefix = "bazel-toolchains-3.1.1",
    urls = [
        "https://github.com/bazelbuild/bazel-toolchains/releases/download/3.1.1/bazel-toolchains-3.1.1.tar.gz",
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-toolchains/archive/3.1.1.tar.gz",
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
    sha256 = "d8c45ee70ec39a57e7a05e5027c32b1576cc7f16d9dd37135b0eddde45cf1b10",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/bazel-gazelle/releases/download/v0.20.0/bazel-gazelle-v0.20.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.20.0/bazel-gazelle-v0.20.0.tar.gz",
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
    sha256 = "dc97fccceacd4c6be14e800b2a00693d5e8d07f69ee187babfd04a80a9f8e250",
    strip_prefix = "rules_docker-0.14.1",
    url = "https://github.com/bazelbuild/rules_docker/archive/v0.14.1.tar.gz",
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "7b9bbe3ea1fccb46dcfa6c3f3e29ba7ec740d8733370e21cdc8937467b4a4349",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.22.4/rules_go-v0.22.4.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.22.4/rules_go-v0.22.4.tar.gz",
    ],
)

http_archive(
    name = "fuzzit_linux",
    build_file_content = "exports_files([\"fuzzit\"])",
    sha256 = "9ca76ac1c22d9360936006efddf992977ebf8e4788ded8e5f9d511285c9ac774",
    urls = ["https://github.com/fuzzitdev/fuzzit/releases/download/v2.4.76/fuzzit_Linux_x86_64.zip"],
)

git_repository(
    name = "graknlabs_bazel_distribution",
    commit = "962f3a7e56942430c0ec120c24f9e9f2a9c2ce1a",
    remote = "https://github.com/graknlabs/bazel-distribution",
    shallow_since = "1569509514 +0300",
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

load("@prysm//third_party/herumi:herumi.bzl", "bls_dependencies")

bls_dependencies()

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(nogo = "@//:nogo")

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
load("@prysm//tools/go:def.bzl", "go_repository")

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
    sha256 = "e31f491682a7b5bcf48250efefd150e6dcf0b2d7658d0ec5407c053fb84cbad0",
    url = "https://github.com/ethereum/eth2.0-spec-tests/releases/download/v0.11.2/general.tar.gz",
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
    sha256 = "1d978d482cd680f99dfc808ccfd59c88e33089450ffc1e283041cbf6dc8982db",
    url = "https://github.com/ethereum/eth2.0-spec-tests/releases/download/v0.11.2/minimal.tar.gz",
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
    sha256 = "ac68dcd5c3a78a7a51e86b61131f71683d844c81b79547029cc6dfa259c3c6dd",
    url = "https://github.com/ethereum/eth2.0-spec-tests/releases/download/v0.11.2/mainnet.tar.gz",
)

http_archive(
    name = "com_github_bazelbuild_buildtools",
    sha256 = "b5d7dbc6832f11b6468328a376de05959a1a9e4e9f5622499d3bab509c26b46a",
    strip_prefix = "buildtools-bf564b4925ab5876a3f64d8b90fab7f769013d42",
    url = "https://github.com/bazelbuild/buildtools/archive/bf564b4925ab5876a3f64d8b90fab7f769013d42.zip",
)

load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")

buildifier_dependencies()

go_repository(
    name = "com_github_golang_mock",
    commit = "d74b93584564161b2de771089ee697f07d8bd5b5",  # v1.3.1
    importpath = "github.com/golang/mock",
)

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
    sha256 = "f6be27772babfdacbbf2e4c5432ea46c57ef5b7d82e52a81b885e7b804781fd6",
    strip_prefix = "librdkafka-1.2.1",
    urls = ["https://github.com/edenhill/librdkafka/archive/v1.2.1.tar.gz"],
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

go_repository(
    name = "com_github_ethereum_go_ethereum",
    commit = "0beb54b2147b3473a4c55e5ce6f02643ce403b14",
    importpath = "github.com/ethereum/go-ethereum",
    # Note: go-ethereum is not bazel-friendly with regards to cgo. We have a
    # a fork that has resolved these issues by disabling HID/USB support and
    # some manual fixes for c imports in the crypto package. This is forked
    # branch should be updated from time to time with the latest go-ethereum
    # code.
    remote = "https://github.com/prysmaticlabs/bazel-go-ethereum",
    vcs = "git",
)

go_repository(
    name = "com_github_prysmaticlabs_go_ssz",
    commit = "e24db4d9e9637cf88ee9e4a779e339a1686a84ee",
    importpath = "github.com/prysmaticlabs/go-ssz",
    patch_args = ["-p1"],
    patches = [
        "//third_party:com_github_prysmaticlabs_go_ssz.patch",
    ],
)

go_repository(
    name = "com_github_go_yaml_yaml",
    commit = "f221b8435cfb71e54062f6c6e99e9ade30b124d5",  # v2.2.4
    importpath = "github.com/go-yaml/yaml",
)

go_repository(
    name = "com_github_x_cray_logrus_prefixed_formatter",
    commit = "bb2702d423886830dee131692131d35648c382e2",  # v0.5.2
    importpath = "github.com/x-cray/logrus-prefixed-formatter",
)

go_repository(
    name = "com_github_mgutz_ansi",
    commit = "9520e82c474b0a04dd04f8a40959027271bab992",
    importpath = "github.com/mgutz/ansi",
)

go_repository(
    name = "com_github_fjl_memsize",
    commit = "2a09253e352a56f419bd88effab0483f52da4c7d",
    importpath = "github.com/fjl/memsize",
)

go_repository(
    name = "com_github_libp2p_go_libp2p",
    commit = "76944c4fc848530530f6be36fb22b70431ca506c",  # v0.5.1
    importpath = "github.com/libp2p/go-libp2p",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_peer",
    commit = "62676d8fb785a8fc279878cbe8e03b878f005910",  # v0.2.0
    importpath = "github.com/libp2p/go-libp2p-peer",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_crypto",
    build_file_proto_mode = "disable_global",
    commit = "ddb6d72b5ad0ae81bf1ee77b628eac1d7237536a",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-crypto",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr",
    commit = "8c6cee15b340d7210c30a82a19231ee333b69b1d",  # v0.2.0
    importpath = "github.com/multiformats/go-multiaddr",
)

go_repository(
    name = "com_github_ipfs_go_log",
    commit = "91b837264c0f35dd4e2be341d711316b91d3573d",  # v0.0.1
    importpath = "github.com/ipfs/go-log",
)

go_repository(
    name = "com_github_multiformats_go_multihash",
    commit = "6b39927dce4869bc1726861b65ada415ee1f7fc7",  # v0.0.13
    importpath = "github.com/multiformats/go-multihash",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_swarm",
    commit = "4f59859086ea4bfd750cf40ff2598fe8e6256f78",  # v0.2.2
    importpath = "github.com/libp2p/go-libp2p-swarm",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_host",
    commit = "fb741ff65522f904e7d46f527c9a823f32346f83",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-host",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_peerstore",
    commit = "dee88d7532302c001604811fa3fbb5a7f83225e7",  # v0.1.4
    importpath = "github.com/libp2p/go-libp2p-peerstore",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_circuit",
    commit = "61af9db0dd78e01e53b9fb044be44dcc7255667e",  # v0.1.4
    importpath = "github.com/libp2p/go-libp2p-circuit",
)

go_repository(
    name = "com_github_coreos_go_semver",
    commit = "e214231b295a8ea9479f11b70b35d5acf3556d9b",  # v0.3.0
    importpath = "github.com/coreos/go-semver",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_interface_connmgr",
    commit = "ad0549099b57dc8a5f0fe2f596467960ed1ed66b",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-interface-connmgr",
)

go_repository(
    name = "com_github_libp2p_go_conn_security_multistream",
    commit = "09b4134a655b5fc883a5bdd62ea12db6e0a1b095",  # v0.1.0
    importpath = "github.com/libp2p/go-conn-security-multistream",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_metrics",
    commit = "2551ab4747111d6c216a06d963c575cebdfd5c9f",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-metrics",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_net",
    commit = "e8fc79d2ba74e10b386a79ba9176b88680f8acb0",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-net",
)

go_repository(
    name = "com_github_whyrusleeping_mafmt",
    commit = "7aa7fad2ede4e7157818e3e7af5061f866a9ae23",  # v1.2.8
    importpath = "github.com/whyrusleeping/mafmt",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_net",
    commit = "c9acf9f27c5020e78925937dc3de142d2d393cd1",  # v0.1.1
    importpath = "github.com/multiformats/go-multiaddr-net",
)

go_repository(
    name = "com_github_minio_blake2b_simd",
    commit = "3f5f724cb5b182a5c278d6d3d55b40e7f8c2efb4",
    importpath = "github.com/minio/blake2b-simd",
)

go_repository(
    name = "com_github_mattn_go_colorable",
    commit = "98ec13f34aabf44cc914c65a1cfb7b9bc815aef1",  # v0.1.4
    importpath = "github.com/mattn/go-colorable",
)

go_repository(
    name = "com_github_btcsuite_btcd",
    commit = "306aecffea325e97f513b3ff0cf7895a5310651d",
    importpath = "github.com/btcsuite/btcd",
)

go_repository(
    name = "com_github_minio_sha256_simd",
    commit = "6de4475307716de15b286880ff321c9547086fdd",  # v0.1.1
    importpath = "github.com/minio/sha256-simd",
)

go_repository(
    name = "com_github_mr_tron_base58",
    commit = "d504ab2e22d97cb9f10b1d146a1e6a063f4a5f43",  # v1.1.2
    importpath = "github.com/mr-tron/base58",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_secio",
    build_file_proto_mode = "disable_global",
    commit = "6f83420d5715a8b1c4082aaf9c5c7785923e702e",  # v0.2.1
    importpath = "github.com/libp2p/go-libp2p-secio",
)

go_repository(
    name = "com_github_libp2p_go_tcp_transport",
    commit = "4da01758afabe2347b015cc12d3478a384ebc909",  # v0.1.1
    importpath = "github.com/libp2p/go-tcp-transport",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_protocol",
    commit = "25288782ae7dd539248ffa7dc62d521027ea311b",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-protocol",
)

go_repository(
    name = "com_github_jbenet_goprocess",
    commit = "7f9d9ed286badffcf2122cfeb383ec37daf92508",
    importpath = "github.com/jbenet/goprocess",
)

go_repository(
    name = "com_github_multiformats_go_multistream",
    commit = "039807e4901c4b2041f40a0e4aa32d72939608aa",  # v0.1.0
    importpath = "github.com/multiformats/go-multistream",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_loggables",
    commit = "814642b01726ff6f9302e8ce9eeeb00d25409520",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-loggables",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_nat",
    commit = "873ef75f6ab6273821d77197660c1fb3af4cc02e",  # v0.0.5
    importpath = "github.com/libp2p/go-libp2p-nat",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_dns",
    commit = "aeb5743691b968cfa3365c9da59ef872a3133c87",  # v0.2.0
    importpath = "github.com/multiformats/go-multiaddr-dns",
)

go_repository(
    name = "com_github_whyrusleeping_go_logging",
    commit = "d89ec39241781fab261571aeddb2a4177bb57bf3",  # v0.0.1
    importpath = "github.com/whyrusleeping/go-logging",
)

go_repository(
    name = "com_github_mattn_go_isatty",
    commit = "7b513a986450394f7bbf1476909911b3aa3a55ce",
    importpath = "github.com/mattn/go-isatty",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_transport_upgrader",
    commit = "07ed92ccf9aba3a2e4b2fddc4c17ced060524922",  # v0.1.1
    importpath = "github.com/libp2p/go-libp2p-transport-upgrader",
)

go_repository(
    name = "com_github_libp2p_go_maddr_filter",
    commit = "4d5679194bce9c87a81d3b9948a4b5edd5ddc094",  # v0.0.5
    importpath = "github.com/libp2p/go-maddr-filter",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_transport",
    commit = "2406e91c260757c7cf63c70ad20073f5a7b29af4",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-transport",
)

go_repository(
    name = "com_github_libp2p_go_addr_util",
    commit = "4cd36c0f325f9e38f1e31ff7a10b9d94d53a11cf",  # v0.0.1
    importpath = "github.com/libp2p/go-addr-util",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_interface_pnet",
    commit = "1357b4bb4b863afcc688f7820c88564ad79818be",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-interface-pnet",
)

go_repository(
    name = "com_github_whyrusleeping_timecache",
    commit = "cfcb2f1abfee846c430233aef0b630a946e0a5a6",
    importpath = "github.com/whyrusleeping/timecache",
)

go_repository(
    name = "com_github_opentracing_opentracing_go",
    commit = "659c90643e714681897ec2521c60567dd21da733",  # v1.1.0
    importpath = "github.com/opentracing/opentracing-go",
)

go_repository(
    name = "com_github_libp2p_go_reuseport",
    commit = "3e6d618acfdfacbbeff71cb2bd70fc188f897a0f",  # v0.0.1
    importpath = "github.com/libp2p/go-reuseport",
)

go_repository(
    name = "com_github_huin_goupnp",
    commit = "656e61dfadd241c7cbdd22a023fa81ecb6860ea8",  # v1.0.0
    importpath = "github.com/huin/goupnp",
)

go_repository(
    name = "com_github_spaolacci_murmur3",
    commit = "f09979ecbc725b9e6d41a297405f65e7e8804acc",  # v1.1.0
    importpath = "github.com/spaolacci/murmur3",
)

go_repository(
    name = "com_github_jbenet_go_temp_err_catcher",
    commit = "aac704a3f4f27190b4ccc05f303a4931fd1241ff",
    importpath = "github.com/jbenet/go-temp-err-catcher",
)

go_repository(
    name = "com_github_sirupsen_logrus",
    commit = "839c75faf7f98a33d445d181f3018b5c3409a45e",  # v1.4.2
    importpath = "github.com/sirupsen/logrus",
)

go_repository(
    name = "org_golang_x_sys",
    commit = "fae7ac547cb717d141c433a2a173315e216b64c4",
    importpath = "golang.org/x/sys",
)

go_repository(
    name = "com_github_libp2p_go_flow_metrics",
    commit = "e5a6a4db89199d99b2a74b8da198277a826241d8",  # v0.0.3
    importpath = "github.com/libp2p/go-flow-metrics",
)

go_repository(
    name = "com_github_libp2p_go_msgio",
    commit = "9142103f7d8dc5a74a91116b8f927fe8d8bf4a96",  # v0.0.4
    importpath = "github.com/libp2p/go-msgio",
)

go_repository(
    name = "com_github_jackpal_gateway",
    commit = "cbcf4e3f3baee7952fc386c8b2534af4d267c875",  # v1.0.5
    importpath = "github.com/jackpal/gateway",
)

go_repository(
    name = "com_github_whyrusleeping_multiaddr_filter",
    commit = "e903e4adabd70b78bc9293b6ee4f359afb3f9f59",
    importpath = "github.com/whyrusleeping/multiaddr-filter",
)

go_repository(
    name = "com_github_libp2p_go_ws_transport",
    commit = "370d1a3a7420e27423417c37630cad3754ad5702",  # v0.2.0
    importpath = "github.com/libp2p/go-ws-transport",
)

go_repository(
    name = "org_golang_x_crypto",
    importpath = "golang.org/x/crypto",
    sum = "h1:1ZiEyfaQIg3Qh0EoqpwAakHVhecoE5wlSg5GjnafJGw=",
    version = "v0.0.0-20200221231518-2aa609cf4a9d",
)

go_repository(
    name = "com_github_jackpal_go_nat_pmp",
    commit = "d89d09f6f3329bc3c2479aa3cafd76a5aa93a35c",
    importpath = "github.com/jackpal/go-nat-pmp",
)

go_repository(
    name = "com_github_libp2p_go_reuseport_transport",
    commit = "c7583c88df654a2ecd621e863f661783d79b64d1",  # v0.0.2
    importpath = "github.com/libp2p/go-reuseport-transport",
)

go_repository(
    name = "com_github_whyrusleeping_go_notifier",
    commit = "097c5d47330ff6a823f67e3515faa13566a62c6f",
    importpath = "github.com/whyrusleeping/go-notifier",
)

go_repository(
    name = "com_github_gorilla_websocket",
    commit = "c3e18be99d19e6b3e8f1559eea2c161a665c4b6b",  # v1.4.1
    importpath = "github.com/gorilla/websocket",
)

go_repository(
    name = "com_github_syndtr_goleveldb",
    commit = "9d007e481048296f09f59bd19bb7ae584563cd95",  # v1.0.0
    importpath = "github.com/syndtr/goleveldb",
)

go_repository(
    name = "com_github_emicklei_dot",
    commit = "5810de2f2ab7aac98cd7bcbd59147a7ca6071768",
    importpath = "github.com/emicklei/dot",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_blankhost",
    commit = "da3b45205dfce3ef3926054ffd5dee76f5903382",  # v0.1.4
    importpath = "github.com/libp2p/go-libp2p-blankhost",
)

go_repository(
    name = "io_opencensus_go",
    importpath = "go.opencensus.io",
    sum = "h1:75k/FF0Q2YM8QYo07VPddOLBslDt1MZOdEslOHvmzAs=",
    version = "v0.22.2",
)

go_repository(
    name = "io_opencensus_go_contrib_exporter_jaeger",
    importpath = "contrib.go.opencensus.io/exporter/jaeger",
    sum = "h1:nhTv/Ry3lGmqbJ/JGvCjWxBl5ozRfqo86Ngz59UAlfk=",
    version = "v0.2.0",
)

go_repository(
    name = "org_golang_google_api",
    importpath = "google.golang.org/api",
    sum = "h1:uMf5uLi4eQMRrMKhCplNik4U4H8Z6C1br3zOtAa/aDE=",
    version = "v0.14.0",
)

go_repository(
    name = "org_golang_x_sync",
    commit = "e225da77a7e68af35c70ccbf71af2b83e6acac3c",
    importpath = "golang.org/x/sync",
)

go_repository(
    name = "com_github_golang_lint",
    commit = "5b3e6a55c961c61f4836ae6868c17b070744c590",
    importpath = "github.com/golang/lint",
)

go_repository(
    name = "org_golang_x_lint",
    commit = "5b3e6a55c961c61f4836ae6868c17b070744c590",
    importpath = "golang.org/x/lint",
)

go_repository(
    name = "com_github_aristanetworks_goarista",
    commit = "728bce664cf5dfb921941b240828f989a2c8f8e3",
    importpath = "github.com/aristanetworks/goarista",
)

go_repository(
    name = "com_github_prometheus_client_golang",
    commit = "4ab88e80c249ed361d3299e2930427d9ac43ef8d",  # v1.0.0
    importpath = "github.com/prometheus/client_golang",
)

go_repository(
    name = "com_github_prometheus_client_model",
    commit = "7bc5445566f0fe75b15de23e6b93886e982d7bf9",
    importpath = "github.com/prometheus/client_model",
)

go_repository(
    name = "com_github_prometheus_common",
    commit = "d978bcb1309602d68bb4ba69cf3f8ed900e07308",
    importpath = "github.com/prometheus/common",
)

go_repository(
    name = "com_github_prometheus_procfs",
    commit = "6d489fc7f1d9cd890a250f3ea3431b1744b9623f",
    importpath = "github.com/prometheus/procfs",
)

go_repository(
    name = "com_github_beorn7_perks",
    commit = "37c8de3658fcb183f997c4e13e8337516ab753e6",  # v1.0.1
    importpath = "github.com/beorn7/perks",
)

go_repository(
    name = "com_github_matttproud_golang_protobuf_extensions",
    commit = "c12348ce28de40eed0136aa2b644d0ee0650e56c",  # v1.0.1
    importpath = "github.com/matttproud/golang_protobuf_extensions",
)

go_repository(
    name = "com_github_pborman_uuid",
    commit = "8b1b92947f46224e3b97bb1a3a5b0382be00d31e",  # v1.2.0
    importpath = "github.com/pborman/uuid",
)

go_repository(
    name = "com_github_libp2p_go_buffer_pool",
    commit = "c4a5988a1e475884367015e1a2d0bd5fa4c491f4",  # v0.0.2
    importpath = "github.com/libp2p/go-buffer-pool",
)

go_repository(
    name = "com_github_libp2p_go_mplex",
    commit = "62fe9554facaec3f80333b61ea8d694fe615705f",  # v0.1.0
    importpath = "github.com/libp2p/go-mplex",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_pubsub",
    build_file_proto_mode = "disable_global",
    importpath = "github.com/libp2p/go-libp2p-pubsub",
    sum = "h1:+Iz8zeI1KO6HX8cexU9g98cCGjae52Vujeg087SkuME=",
    version = "v0.2.6-0.20191219233527-97846b574895",
)

go_repository(
    name = "com_github_ipfs_go_ipfs_util",
    commit = "a4bb5361e49427531f9a716ead2ce4bd9bdd7959",  # v0.0.1
    importpath = "github.com/ipfs/go-ipfs-util",
)

go_repository(
    name = "com_github_google_uuid",
    commit = "0cd6bf5da1e1c83f8b45653022c74f71af0538a4",  # v1.1.1
    importpath = "github.com/google/uuid",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_kad_dht",
    build_file_proto_mode = "disable_global",
    commit = "e216d3cf6cfadfc91b8c3bec6ac9492ea40908d0",  # v0.2.1
    importpath = "github.com/libp2p/go-libp2p-kad-dht",
)

go_repository(
    name = "com_github_ipfs_go_datastore",
    commit = "e7a498916ccca1b0b40fb08630659cd4d68a01e8",  # v0.3.1
    importpath = "github.com/ipfs/go-datastore",
)

go_repository(
    name = "com_github_whyrusleeping_base32",
    commit = "c30ac30633ccdabefe87eb12465113f06f1bab75",
    importpath = "github.com/whyrusleeping/base32",
)

go_repository(
    name = "com_github_ipfs_go_cid",
    commit = "3da5bbbe45260437a44f777e6b2e5effa2606901",  # v0.0.4
    importpath = "github.com/ipfs/go-cid",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_record",
    build_file_proto_mode = "disable_global",
    commit = "8ccbca30634f70a8f03d133ac64cbf245d079e1e",  # v0.1.2
    importpath = "github.com/libp2p/go-libp2p-record",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_routing",
    commit = "f4ece6c1baa8e77ee488b25014fcb1059955ed0f",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-routing",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_kbucket",
    commit = "a0cac6f63c491504b18eeba24be2ac0bbbfa0e5c",  # v0.2.3
    importpath = "github.com/libp2p/go-libp2p-kbucket",
)

go_repository(
    name = "com_github_ipfs_go_todocounter",
    commit = "742667602a47ab3a2b7f17d935019c3255719dce",  # v0.0.2
    importpath = "github.com/ipfs/go-todocounter",
)

go_repository(
    name = "com_github_whyrusleeping_go_keyspace",
    commit = "5b898ac5add1da7178a4a98e69cb7b9205c085ee",
    importpath = "github.com/whyrusleeping/go-keyspace",
)

go_repository(
    name = "com_github_multiformats_go_multibase",
    commit = "d63641945dc1749baa23686ad0564ad63fef0493",  # v0.0.1
    importpath = "github.com/multiformats/go-multibase",
)

go_repository(
    name = "com_github_hashicorp_golang_lru",
    commit = "14eae340515388ca95aa8e7b86f0de668e981f54",  # v0.5.4
    importpath = "github.com/hashicorp/golang-lru",
)

go_repository(
    name = "com_github_ipfs_go_ipfs_addr",
    commit = "ac4881d4db36effbbeebf93d9172fcb20ed04c15",  # v0.0.1
    importpath = "github.com/ipfs/go-ipfs-addr",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_discovery",
    importpath = "github.com/libp2p/go-libp2p-discovery",
    sum = "h1:1p3YSOq7VsgaL+xVHPi8XAmtGyas6D2J6rWBEfz/aiY=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_autonat",
    commit = "60bf479cf6bc73c939f4db97ad711756e949e522",  # v0.1.1
    importpath = "github.com/libp2p/go-libp2p-autonat",
)

go_repository(
    name = "com_github_konsorten_go_windows_terminal_sequences",
    commit = "f55edac94c9bbba5d6182a4be46d86a2c9b5b50e",  # v1.0.2
    importpath = "github.com/konsorten/go-windows-terminal-sequences",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_interface_conn",
    commit = "c7cda99284db0bea441058da8fd1f1373c763ed6",
    importpath = "github.com/libp2p/go-libp2p-interface-conn",
)

go_repository(
    name = "io_k8s_client_go",
    build_extra_args = ["-exclude=vendor"],
    commit = "c1ea390cb7f7ca6d6345b4d3bcfd5546028cee20",  # v12.0.0
    importpath = "k8s.io/client-go",
)

go_repository(
    name = "io_etcd_go_bbolt",
    importpath = "go.etcd.io/bbolt",
    sum = "h1:hi1bXHMVrlQh6WwxAy+qZCV/SYIlqo+Ushwdpa4tAKg=",
    version = "v1.3.4",
)

go_repository(
    name = "io_k8s_apimachinery",
    build_file_proto_mode = "disable_global",
    commit = "79c2a76c473a20cdc4ce59cae4b72529b5d9d16b",  # v0.17.2
    importpath = "k8s.io/apimachinery",
)

go_repository(
    name = "io_k8s_klog",
    commit = "2ca9ad30301bf30a8a6e0fa2110db6b8df699a91",  # v1.0.0
    importpath = "k8s.io/klog",
)

go_repository(
    name = "com_github_google_gofuzz",
    importpath = "github.com/google/gofuzz",
    sum = "h1:Hsa8mG0dQ46ij8Sl2AYJDUv1oA9/d6Vk+3LG99Oe02g=",
    version = "v1.1.0",
)

go_repository(
    name = "io_k8s_api",
    build_file_proto_mode = "disable_global",
    commit = "3043179095b6baa0087e8735d796bd6dfa881f8e",
    importpath = "k8s.io/api",
)

go_repository(
    name = "com_github_shyiko_kubesec",
    commit = "7718facdb5e5529cecff1fe42fc3aaa4cc837d5d",
    importpath = "github.com/shyiko/kubesec",
)

go_repository(
    name = "in_gopkg_yaml_v2",
    commit = "f221b8435cfb71e54062f6c6e99e9ade30b124d5",  # v2.2.4
    importpath = "gopkg.in/yaml.v2",
)

go_repository(
    name = "com_github_spf13_pflag",
    commit = "2e9d26c8c37aae03e3f9d4e90b7116f5accb7cab",  # v1.0.5
    importpath = "github.com/spf13/pflag",
)

go_repository(
    name = "com_github_spf13_cobra",
    commit = "f2b07da1e2c38d5f12845a4f607e2e1018cbb1f5",  # v0.0.5
    importpath = "github.com/spf13/cobra",
)

go_repository(
    name = "com_github_aws_aws_sdk_go",
    commit = "36cc7fd7051ac4707bd56c8774825df9e8de5918",
    importpath = "github.com/aws/aws-sdk-go",
)

go_repository(
    name = "com_github_posener_complete",
    commit = "699ede78373dfb0168f00170591b698042378437",
    importpath = "github.com/posener/complete",
    remote = "https://github.com/shyiko/complete",
    vcs = "git",
)

go_repository(
    name = "org_golang_x_oauth2",
    commit = "e64efc72b421e893cbf63f17ba2221e7d6d0b0f3",
    importpath = "golang.org/x/oauth2",
)

go_repository(
    name = "com_github_hashicorp_go_multierror",
    commit = "886a7fbe3eb1c874d46f623bfa70af45f425b3d1",  # v1.0.0
    importpath = "github.com/hashicorp/go-multierror",
)

go_repository(
    name = "com_github_hashicorp_errwrap",
    commit = "8a6fb523712970c966eefc6b39ed2c5e74880354",  # v1.0.0
    importpath = "github.com/hashicorp/errwrap",
)

go_repository(
    name = "com_google_cloud_go",
    commit = "6daa679260d92196ffca2362d652c924fdcb7a22",  # v0.52.0
    importpath = "cloud.google.com/go",
)

go_repository(
    name = "com_github_inconshreveable_mousetrap",
    commit = "76626ae9c91c4f2a10f34cad8ce83ea42c93bb75",  # v1.0.0
    importpath = "github.com/inconshreveable/mousetrap",
)

go_repository(
    name = "com_github_deckarep_golang_set",
    commit = "cbaa98ba5575e67703b32b4b19f73c91f3c4159e",  # v1.7.1
    importpath = "github.com/deckarep/golang-set",
)

go_repository(
    name = "com_github_go_stack_stack",
    commit = "2fee6af1a9795aafbe0253a0cfbdf668e1fb8a9a",  # v1.8.0
    importpath = "github.com/go-stack/stack",
)

go_repository(
    name = "com_github_rs_cors",
    commit = "db0fe48135e83b5812a5a31be0eea66984b1b521",  # v1.7.0
    importpath = "github.com/rs/cors",
)

go_repository(
    name = "com_github_golang_snappy",
    commit = "2a8bb927dd31d8daada140a5d09578521ce5c36a",  # v0.0.1
    importpath = "github.com/golang/snappy",
)

go_repository(
    name = "com_github_edsrzf_mmap_go",
    commit = "188cc3b666ba704534fa4f96e9e61f21f1e1ba7c",  # v1.0.0
    importpath = "github.com/edsrzf/mmap-go",
)

go_repository(
    name = "com_github_pkg_errors",
    commit = "614d223910a179a466c1767a985424175c39b465",  # v0.9.1
    importpath = "github.com/pkg/errors",
)

go_repository(
    name = "in_gopkg_natefinch_npipe_v2",
    commit = "c1b8fa8bdccecb0b8db834ee0b92fdbcfa606dd6",
    importpath = "gopkg.in/natefinch/npipe.v2",
)

go_repository(
    name = "com_github_prestonvanloon_go_recaptcha",
    commit = "0834cef6e8bd3a7ebdb3ac7def9440ee47d501a4",
    importpath = "github.com/prestonvanloon/go-recaptcha",
)

go_repository(
    name = "com_github_phoreproject_bls",
    commit = "da95d4798b09e9f45a29dc53124b2a0b4c1dfc13",
    importpath = "github.com/phoreproject/bls",
)

go_repository(
    name = "com_github_multiformats_go_base32",
    commit = "a9c2755c3d1672dbe6a7e4a5d182169fa30b6a8e",  # v0.0.3
    importpath = "github.com/multiformats/go-base32",
)

go_repository(
    name = "org_golang_x_xerrors",
    commit = "a5947ffaace3e882f334c1750858b4a6a7e52422",
    importpath = "golang.org/x/xerrors",
)

go_repository(
    name = "com_github_grpc_ecosystem_go_grpc_middleware",
    commit = "dd15ed025b6054e5253963e355991f3070d4e593",  # v1.1.0
    importpath = "github.com/grpc-ecosystem/go-grpc-middleware",
)

go_repository(
    name = "com_github_apache_thrift",
    commit = "cecee50308fc7e6f77f55b3fd906c1c6c471fa2f",  # v0.13.0
    importpath = "github.com/apache/thrift",
)

go_repository(
    name = "com_github_grpc_ecosystem_go_grpc_prometheus",
    commit = "c225b8c3b01faf2899099b768856a9e916e5087b",  # v1.2.0
    importpath = "github.com/grpc-ecosystem/go-grpc-prometheus",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_connmgr",
    commit = "273839464339f1885413b385feee35301c5cb76f",  # v0.2.1
    importpath = "github.com/libp2p/go-libp2p-connmgr",
)

go_repository(
    name = "com_github_joonix_log",
    commit = "13fe31bbdd7a6f706b9114e188cdb53856be4d64",
    importpath = "github.com/joonix/log",
)

go_repository(
    name = "grpc_ecosystem_grpc_gateway",
    importpath = "github.com/grpc-ecosystem/grpc-gateway",
    sum = "h1:IOPK2xMPP3aV6/NPt4jt//ELFo3Vv8sDVD8j3+tleDU=",
    version = "v1.14.4",
)

go_repository(
    name = "com_github_ghodss_yaml",
    commit = "0ca9ea5df5451ffdf184b4428c902747c2c11cd7",  # v1.0.0
    importpath = "github.com/ghodss/yaml",
)

go_repository(
    name = "org_uber_go_automaxprocs",
    commit = "946a8391268aea0a60a86403988ff3ab4b604a83",  # v1.2.0
    importpath = "go.uber.org/automaxprocs",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_core",
    build_file_proto_mode = "disable_global",
    commit = "f7f724862d85ec9f9ee7c58b0f79836abdee8cd9",  # v0.3.0
    importpath = "github.com/libp2p/go-libp2p-core",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_testing",
    commit = "82713a62880a5fe72d438bd58d737f0d3c4b7f36",  # v0.1.1
    importpath = "github.com/libp2p/go-libp2p-testing",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_yamux",
    commit = "a61e80cb5770aa0d9b1bafe94da1278f58baa2c5",  # v0.2.1
    importpath = "github.com/libp2p/go-libp2p-yamux",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_mplex",
    commit = "811729f15f0af13fe3f0d9e410c22f6a4bc5c686",  # v0.2.1
    importpath = "github.com/libp2p/go-libp2p-mplex",
)

go_repository(
    name = "com_github_libp2p_go_stream_muxer_multistream",
    commit = "2439b02deee2de8bb1fe24473d3d8333008a714a",  # v0.2.0
    importpath = "github.com/libp2p/go-stream-muxer-multistream",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_fmt",
    commit = "113ed87ed03cfff94f29fd95236be3ccd933fd36",  # v0.1.0
    importpath = "github.com/multiformats/go-multiaddr-fmt",
)

go_repository(
    name = "com_github_multiformats_go_varint",
    commit = "0aa688902217dff2cba0f678c7e4a0f547b4983e",
    importpath = "github.com/multiformats/go-varint",
)

go_repository(
    name = "com_github_libp2p_go_yamux",
    commit = "663972181d409e7263040f0b668462f87c85e1bd",  # v1.2.3
    importpath = "github.com/libp2p/go-yamux",
)

go_repository(
    name = "com_github_libp2p_go_nat",
    commit = "4b355d438085545df006ad9349686f30d8d37a27",  # v0.0.4
    importpath = "github.com/libp2p/go-nat",
)

go_repository(
    name = "com_github_koron_go_ssdp",
    commit = "4a0ed625a78b6858dc8d3a55fb7728968b712122",
    importpath = "github.com/koron/go-ssdp",
)

go_repository(
    name = "com_github_libp2p_go_eventbus",
    commit = "d34a18eba211bd65b32a4a7a06390fc441257cbd",  # v0.1.0
    importpath = "github.com/libp2p/go-eventbus",
)

go_repository(
    name = "in_gopkg_d4l3k_messagediff_v1",
    commit = "29f32d820d112dbd66e58492a6ffb7cc3106312b",  # v1.2.1
    importpath = "gopkg.in/d4l3k/messagediff.v1",
)

go_repository(
    name = "com_github_prysmaticlabs_go_bitfield",
    commit = "62c2aee7166951c456888f92237aee4303ba1b9d",
    importpath = "github.com/prysmaticlabs/go-bitfield",
)

load("@com_github_prysmaticlabs_go_ssz//:deps.bzl", "go_ssz_dependencies")

go_ssz_dependencies()

go_repository(
    name = "org_golang_google_grpc",
    build_file_proto_mode = "disable",
    importpath = "google.golang.org/grpc",
    sum = "h1:zvIju4sqAGvwKspUQOhwnpcqSbzi7/H6QomNNjTL4sk=",
    version = "v1.27.1",
)

go_repository(
    name = "org_golang_x_net",
    commit = "da137c7871d730100384dbcf36e6f8fa493aef5b",
    importpath = "golang.org/x/net",
)

go_repository(
    name = "org_golang_x_text",
    commit = "342b2e1fbaa52c93f31447ad2c6abc048c63e475",  # v0.3.2
    importpath = "golang.org/x/text",
)

go_repository(
    name = "com_github_golang_glog",
    commit = "23def4e6c14b4da8ac2ed8007337bc5eb5007998",
    importpath = "github.com/golang/glog",
)

go_repository(
    name = "org_golang_x_time",
    commit = "9d24e82272b4f38b78bc8cff74fa936d31ccd8ef",
    importpath = "golang.org/x/time",
)

go_repository(
    name = "com_github_googleapis_gnostic",
    commit = "896953e6749863beec38e27029c804e88c3144b8",  # v0.4.1
    importpath = "github.com/googleapis/gnostic",
)

go_repository(
    name = "in_gopkg_inf_v0",
    commit = "d2d2541c53f18d2a059457998ce2876cc8e67cbf",  # v0.9.1
    importpath = "gopkg.in/inf.v0",
)

go_repository(
    name = "com_github_davecgh_go_spew",
    commit = "8991bc29aa16c548c550c7ff78260e27b9ab7c73",  # v1.1.1
    importpath = "github.com/davecgh/go-spew",
)

go_repository(
    name = "io_k8s_sigs_yaml",
    commit = "fd68e9863619f6ec2fdd8625fe1f02e7c877e480",  # v1.1.0
    importpath = "sigs.k8s.io/yaml",
)

go_repository(
    name = "com_github_google_go_cmp",
    commit = "5a6f75716e1203a923a78c9efb94089d857df0f6",  # v0.4.0
    importpath = "github.com/google/go-cmp",
)

go_repository(
    name = "com_github_modern_go_reflect2",
    commit = "94122c33edd36123c84d5368cfb2b69df93a0ec8",  # v1.0.1
    importpath = "github.com/modern-go/reflect2",
)

go_repository(
    name = "com_github_json_iterator_go",
    commit = "4f2e55fcf87ba29ab80379002316db67620ff622",
    importpath = "github.com/json-iterator/go",
    remote = "https://github.com/prestonvanloon/go",
    vcs = "git",
)

go_repository(
    name = "com_github_modern_go_concurrent",
    commit = "bacd9c7ef1dd9b15be4a9909b8ac7a4e313eec94",
    importpath = "github.com/modern-go/concurrent",
)

go_repository(
    name = "io_k8s_utils",
    commit = "3dccf664f023863740c508fb4284e49742bedfa4",
    importpath = "k8s.io/utils",
)

go_repository(
    name = "com_github_patrickmn_go_cache",
    commit = "46f407853014144407b6c2ec7ccc76bf67958d93",
    importpath = "github.com/patrickmn/go-cache",
)

go_repository(
    name = "com_github_prysmaticlabs_ethereumapis",
    commit = "df460bd3d84be4ff3df0658395c7dc9d2a7e7b3d",
    importpath = "github.com/prysmaticlabs/ethereumapis",
)

go_repository(
    name = "com_github_cloudflare_roughtime",
    importpath = "github.com/cloudflare/roughtime",
    sum = "h1:jeSxE3fepJdhASERvBHI6RFkMhISv6Ir2JUybYLIVXs=",
    version = "v0.0.0-20200205191924-a69ef1dab727",
)

go_repository(
    name = "com_googlesource_roughtime_roughtime_git",
    build_file_generation = "on",
    commit = "51f6971f5f06ec101e5fbcabe5a49477708540f3",
    importpath = "roughtime.googlesource.com/roughtime.git",
)

go_repository(
    name = "com_github_paulbellamy_ratecounter",
    commit = "524851a93235ac051e3540563ed7909357fe24ab",  # v0.2.0
    importpath = "github.com/paulbellamy/ratecounter",
)

go_repository(
    name = "com_github_mattn_go_runewidth",
    importpath = "github.com/mattn/go-runewidth",
    sum = "h1:2BvfKmzob6Bmd4YsL0zygOqfdFnK7GR4QL06Do4/p7Y=",
    version = "v0.0.4",
)

go_repository(
    name = "com_github_minio_highwayhash",
    importpath = "github.com/minio/highwayhash",
    sum = "h1:iMSDhgUILCr0TNm8LWlSjF8N0ZIj2qbO8WHp6Q/J2BA=",
    version = "v1.0.0",
)

go_repository(
    name = "org_golang_x_exp",
    importpath = "golang.org/x/exp",
    sum = "h1:n9HxLrNxWWtEb1cA950nuEEj3QnKbtsCJ6KjcgisNUs=",
    version = "v0.0.0-20191002040644-a1355ae1e2c3",
)

go_repository(
    name = "in_gopkg_confluentinc_confluent_kafka_go_v1",
    importpath = "gopkg.in/confluentinc/confluent-kafka-go.v1",
    patch_args = ["-p1"],
    patches = ["//third_party:in_gopkg_confluentinc_confluent_kafka_go_v1.patch"],
    sum = "h1:roy97m/3wj9/o8OuU3sZ5wildk30ep38k2x8nhNbKrI=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_naoina_toml",
    importpath = "github.com/naoina/toml",
    sum = "h1:PT/lllxVVN0gzzSqSlHEmP8MJB4MY2U7STGxiouV4X8=",
    version = "v0.1.1",
)

go_repository(
    name = "com_github_elastic_gosigar",
    importpath = "github.com/elastic/gosigar",
    sum = "h1:GzPQ+78RaAb4J63unidA/JavQRKrB6s8IOzN6Ib59jo=",
    version = "v0.10.5",
)

go_repository(
    name = "com_github_naoina_go_stringutil",
    importpath = "github.com/naoina/go-stringutil",
    sum = "h1:rCUeRUHjBjGTSHl0VC00jUPLz8/F9dDzYI70Hzifhks=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_influxdata_influxdb",
    importpath = "github.com/influxdata/influxdb",
    sum = "h1:uSeBTNO4rBkbp1Be5FKRsAmglM9nlx25TzVQRQt1An4=",
    version = "v1.7.9",
)

go_repository(
    name = "com_github_robertkrimen_otto",
    importpath = "github.com/robertkrimen/otto",
    sum = "h1:1VUlQbCfkoSGv7qP7Y+ro3ap1P1pPZxgdGVqiTVy5C4=",
    version = "v0.0.0-20180617131154-15f95af6e78d",
)

go_repository(
    name = "com_github_peterh_liner",
    importpath = "github.com/peterh/liner",
    sum = "h1:f+aAedNJA6uk7+6rXsYBnhdo4Xux7ESLe+kcuVUF5os=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_graph_gophers_graphql_go",
    importpath = "github.com/graph-gophers/graphql-go",
    sum = "h1:HwRCZlPXN00r58jaIPE11HXn7EvhheQrE+Cxw0vkrH0=",
    version = "v0.0.0-20191031232829-adde0d0f76a3",
)

go_repository(
    name = "com_github_rjeczalik_notify",
    importpath = "github.com/rjeczalik/notify",
    sum = "h1:MiTWrPj55mNDHEiIX5YUSKefw/+lCQVoAFmD6oQm5w8=",
    version = "v0.9.2",
)

go_repository(
    name = "com_github_mohae_deepcopy",
    importpath = "github.com/mohae/deepcopy",
    sum = "h1:RWengNIwukTxcDr9M+97sNutRR1RKhG96O6jWumTTnw=",
    version = "v0.0.0-20170929034955-c48cc78d4826",
)

go_repository(
    name = "in_gopkg_olebedev_go_duktape_v3",
    importpath = "gopkg.in/olebedev/go-duktape.v3",
    sum = "h1:uuol9OUzSvZntY1v963NAbVd7A+PHLMz1FlCe3Lorcs=",
    version = "v3.0.0-20190709231704-1e4459ed25ff",
)

go_repository(
    name = "in_gopkg_sourcemap_v1",
    importpath = "gopkg.in/sourcemap.v1",
    sum = "h1:inv58fC9f9J3TK2Y2R1NPntXEn3/wjWHkonhIUODNTI=",
    version = "v1.0.5",
)

go_repository(
    name = "com_github_fatih_color",
    importpath = "github.com/fatih/color",
    sum = "h1:DkWD4oS2D8LGGgTQ6IvwJJXSL5Vp2ffcQg58nFV38Ys=",
    version = "v1.7.0",
)

go_repository(
    name = "com_github_protolambda_zssz",
    commit = "632f11e5e281660402bd0ac58f76090f3503def0",
    importpath = "github.com/protolambda/zssz",
)

go_repository(
    name = "com_github_googleapis_gnostic",
    commit = "25d8b0b6698593f520d9d8dc5a88e6b16ca9ecc0",
    importpath = "github.com/googleapis/gnostic",
)

go_repository(
    name = "com_github_googleapis_gax_go_v2",
    importpath = "github.com/googleapis/gax-go/v2",
    sum = "h1:sjZBwGj9Jlw33ImPtvFviGYvseOtDM7hkSKB7+Tv3SM=",
    version = "v2.0.5",
)

go_repository(
    name = "com_github_golang_groupcache",
    importpath = "github.com/golang/groupcache",
    sum = "h1:uHTyIjqVhYRhLbJ8nIiOJHkEZZ+5YoOsAbD3sk82NiE=",
    version = "v0.0.0-20191027212112-611e8accdfc9",
)

go_repository(
    name = "com_github_uber_jaeger_client_go",
    importpath = "github.com/uber/jaeger-client-go",
    sum = "h1:HgqpYBng0n7tLJIlyT4kPCIv5XgCsF+kai1NnnrJzEU=",
    version = "v2.20.1+incompatible",
)

go_repository(
    name = "com_github_dgraph_io_ristretto",
    commit = "99d1bbbf28e64530eb246be0568fc7709a35ebdd",  # v0.0.1
    importpath = "github.com/dgraph-io/ristretto",
)

go_repository(
    name = "com_github_cespare_xxhash",
    commit = "d7df74196a9e781ede915320c11c378c1b2f3a1f",
    importpath = "github.com/cespare/xxhash",
)

go_repository(
    name = "com_github_ipfs_go_detect_race",
    importpath = "github.com/ipfs/go-detect-race",
    sum = "h1:qX/xay2W3E4Q1U7d9lNs1sU9nvguX0a7319XbyQ6cOk=",
    version = "v0.0.1",
)

go_repository(
    name = "com_github_kevinms_leakybucket_go",
    importpath = "github.com/kevinms/leakybucket-go",
    sum = "h1:oq6BiN7v0MfWCRcJAxSV+hesVMAAV8COrQbTjYNnso4=",
    version = "v0.0.0-20190611015032-8a3d0352aa79",
)

go_repository(
    name = "com_github_wealdtech_go_eth2_wallet",
    importpath = "github.com/wealdtech/go-eth2-wallet",
    sum = "h1:Hna/w4EKBJIs86VprIq7ez063A6kwk31d/O3Gs+MpYc=",
    version = "v1.9.3",
)

go_repository(
    name = "com_github_wealdtech_go_eth2_wallet_hd_v2",
    importpath = "github.com/wealdtech/go-eth2-wallet-hd/v2",
    sum = "h1:kiCvdexK3zRC2GwZHSHq+hS+irVNtMs5pNADyumeeRM=",
    version = "v2.0.1",
)

go_repository(
    name = "com_github_wealdtech_go_eth2_wallet_nd_v2",
    importpath = "github.com/wealdtech/go-eth2-wallet-nd/v2",
    sum = "h1:4lB6GY5oHQn5xwn/Sxm1e9SeVCaxa7q/0hqXUQYUNwU=",
    version = "v2.0.1",
)

go_repository(
    name = "com_github_wealdtech_go_eth2_wallet_store_filesystem",
    importpath = "github.com/wealdtech/go-eth2-wallet-store-filesystem",
    sum = "h1:Lc6wVTjIYeD+2hLAIzq1SugTWR527vEX4tEr5v3zxJc=",
    version = "v1.7.2",
)

go_repository(
    name = "com_github_wealdtech_go_eth2_wallet_store_s3",
    importpath = "github.com/wealdtech/go-eth2-wallet-store-s3",
    sum = "h1:xzyQDxbe5nr7xG0ByevTV2S8qkeOZvvjp+leBJcpxXQ=",
    version = "v1.6.2",
)

go_repository(
    name = "com_github_wealdtech_go_eth2_wallet_encryptor_keystorev4",
    importpath = "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4",
    sum = "h1:IcpS4VpXhYz+TVupB5n6C6IQzaKwG+Rc8nvgCa/da4c=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_wealdtech_go_eth2_wallet_types_v2",
    importpath = "github.com/wealdtech/go-eth2-wallet-types/v2",
    sum = "h1:Ct3RrNJTapBiG2GxVl53Kfgy96f0GEUV7bediTu91u8=",
    version = "v2.0.1",
)

go_repository(
    name = "com_github_wealdtech_go_eth2_types_v2",
    build_directives = [
        "gazelle:resolve go github.com/herumi/bls-eth-go-binary/bls @herumi_bls_eth_go_binary//:go_default_library",
    ],
    importpath = "github.com/wealdtech/go-eth2-types/v2",
    sum = "h1:qfmgaCBkH2N11LHCXsRWYz7OOxc+1QXrKHlS9yDnFsw=",
    version = "v2.3.0",
)

go_repository(
    name = "com_github_wealdtech_go_eth2_util",
    importpath = "github.com/wealdtech/go-eth2-util",
    sum = "h1:MyM16V7Qhd9q2ZaRa0WteBg2bWb8UplIKjZr8aeBZP0=",
    version = "v1.1.4",
)

go_repository(
    name = "com_github_wealdtech_go_ecodec",
    importpath = "github.com/wealdtech/go-ecodec",
    sum = "h1:yggrTSckcPJRaxxOxQF7FPm21kgE8WA6+f5jdq5Kr8o=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_wealdtech_go_bytesutil",
    importpath = "github.com/wealdtech/go-bytesutil",
    sum = "h1:ocEg3Ke2GkZ4vQw5lp46rmO+pfqCCTgq35gqOy8JKVc=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_wealdtech_go_indexer",
    importpath = "github.com/wealdtech/go-indexer",
    sum = "h1:/S4rfWQbSOnnYmwnvuTVatDibZ8o1s9bmTCHO16XINg=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_shibukawa_configdir",
    commit = "e180dbdc8da04c4fa04272e875ce64949f38bd3e",
    importpath = "github.com/shibukawa/configdir",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_noise",
    importpath = "github.com/libp2p/go-libp2p-noise",
    sum = "h1:J1gHJRNFEk7NdiaPQQqAvxEy+7hhCsVv3uzduWybmqY=",
    version = "v0.0.0-20200302201340-8c54356e12c9",
)

go_repository(
    name = "com_github_ferranbt_fastssz",
    commit = "06015a5d84f9e4eefe2c21377ca678fa8f1a1b09",
    importpath = "github.com/ferranbt/fastssz",
    nofuzz = True,
)

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

go_repository(
    name = "com_github_burntsushi_toml",
    importpath = "github.com/BurntSushi/toml",
    sum = "h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=",
    version = "v0.3.1",
)

go_repository(
    name = "com_github_cpuguy83_go_md2man_v2",
    importpath = "github.com/cpuguy83/go-md2man/v2",
    sum = "h1:EoUDS0afbrsXAZ9YQ9jdu/mZ2sXgT1/2yyNng4PGlyM=",
    version = "v2.0.0",
)

go_repository(
    name = "com_github_russross_blackfriday_v2",
    importpath = "github.com/russross/blackfriday/v2",
    sum = "h1:lPqVAte+HuHNfhJ/0LC98ESWRz8afy9tM/0RK8m9o+Q=",
    version = "v2.0.1",
)

go_repository(
    name = "com_github_shurcool_sanitized_anchor_name",
    importpath = "github.com/shurcooL/sanitized_anchor_name",
    sum = "h1:PdmoCO6wvbs+7yrJyMORt4/BmY5IYyJwS/kOiWx8mHo=",
    version = "v1.0.0",
)

go_repository(
    name = "in_gopkg_urfave_cli_v2",
    importpath = "gopkg.in/urfave/cli.v2",
    sum = "h1:OvXt/p4cdwNl+mwcWMq/AxaKFkhdxcjx+tx+qf4EOvY=",
    version = "v2.0.0-20190806201727-b62605953717",
)

go_repository(
    name = "in_gopkg_urfave_cli_v1",
    importpath = "gopkg.in/urfave/cli.v1",
    sum = "h1:NdAVW6RYxDif9DhDHaAortIu956m2c0v+09AZBPTbE0=",
    version = "v1.20.0",
)

go_repository(
    name = "com_github_wealdtech_eth2_signer_api",
    build_file_proto_mode = "disable_global",
    importpath = "github.com/wealdtech/eth2-signer-api",
    sum = "h1:Fs0GfrdhboBKW7zaMvIvUHJaOB1ibpAmRG3lkB53in4=",
    version = "v1.3.0",
)

go_repository(
    name = "com_github_prysmaticlabs_prombbolt",
    importpath = "github.com/prysmaticlabs/prombbolt",
    sum = "h1:bVD46NhbqEE6bsIqj42TCS3ELUdumti3WfAw9DXNtkg=",
    version = "v0.0.0-20200324184628-09789ef63796",
)

load("@com_github_prysmaticlabs_prombbolt//:repositories.bzl", "prombbolt_dependencies")

prombbolt_dependencies()

go_repository(
    name = "com_github_ianlancetaylor_cgosymbolizer",
    importpath = "github.com/ianlancetaylor/cgosymbolizer",
    sum = "h1:GWsU1WjSE2rtvyTYGcndqmPPkQkBNV7pEuZdnGtwtu4=",
    version = "v0.0.0-20200321040036-d43e30eacb43",
)

go_repository(
    name = "org_golang_x_mod",
    importpath = "golang.org/x/mod",
    sum = "h1:KU7oHjnv3XNWfa5COkzUifxZmxp1TyI7ImMXqFxLwvQ=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_golang_gddo",
    commit = "3c2cc9a6329d9842b3bbdaf307a8110d740cf94c",
    importpath = "github.com/golang/gddo",
)
