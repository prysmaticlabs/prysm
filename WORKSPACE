load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

http_archive(
    name = "bazel_skylib",
    sha256 = "2ea8a5ed2b448baf4a6855d3ce049c4c452a6470b1efd1504fdb7c1c134d220a",
    strip_prefix = "bazel-skylib-0.8.0",
    url = "https://github.com/bazelbuild/bazel-skylib/archive/0.8.0.tar.gz",
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "313f2c7a23fecc33023563f082f381a32b9b7254f727a7dd2d6380ccc6dfe09b",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/rules_go/releases/download/0.19.3/rules_go-0.19.3.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/0.19.3/rules_go-0.19.3.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "3c681998538231a2d24d0c07ed5a7658cb72bfb5fd4bf9911157c0e9ac6a2687",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.17.0/bazel-gazelle-0.17.0.tar.gz"],
)

http_archive(
    name = "com_github_atlassian_bazel_tools",
    sha256 = "6b438f4d8c698f69ed4473cba12da3c3a7febf90ce8e3c383533d5a64d8c8f19",
    strip_prefix = "bazel-tools-6fbc36c639a8f376182bb0057dd557eb2440d4ed",
    urls = ["https://github.com/atlassian/bazel-tools/archive/6fbc36c639a8f376182bb0057dd557eb2440d4ed.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "9ff889216e28c918811b77999257d4ac001c26c1f7c7fb17a79bc28abf74182e",
    strip_prefix = "rules_docker-0.10.1",
    url = "https://github.com/bazelbuild/rules_docker/archive/v0.10.1.tar.gz",
)

http_archive(
    name = "build_bazel_rules_nodejs",
    sha256 = "da217044d24abd16667324626a33581f3eaccabf80985b2688d6a08ed2f864be",
    urls = ["https://github.com/bazelbuild/rules_nodejs/releases/download/0.37.1/rules_nodejs-0.37.1.tar.gz"],
)

git_repository(
    name = "graknlabs_bazel_distribution",
    commit = "d4a7864b90733ee8ab51c32d0657b2d4868d2f70",
    remote = "https://github.com/graknlabs/bazel-distribution",
    shallow_since = "1563544980 +0300",
)

# Override default import in rules_go with special patch until
# https://github.com/gogo/protobuf/pull/582 is merged.
git_repository(
    name = "com_github_gogo_protobuf",
    commit = "ba06b47c162d49f2af050fb4c75bcbc86a159d5c",  # v1.2.1, as of 2019-03-03
    patch_args = ["-p1"],
    patches = [
        "@io_bazel_rules_go//third_party:com_github_gogo_protobuf-gazelle.patch",
        "//third_party:com_github_gogo_protobuf-equal.patch",
    ],
    remote = "https://github.com/gogo/protobuf",
    shallow_since = "1550471403 +0200",
    # gazelle args: -go_prefix github.com/gogo/protobuf -proto legacy
)

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(nogo = "@//:nogo")

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

load("@com_github_atlassian_bazel_tools//gometalinter:deps.bzl", "gometalinter_dependencies")

gometalinter_dependencies()

load(
    "@io_bazel_rules_docker//go:image.bzl",
    _go_image_repos = "repositories",
)

_go_image_repos()

http_archive(
    name = "prysm_testnet_site",
    build_file_content = """
proto_library(
  name = "faucet_proto",
  srcs = ["src/proto/faucet.proto"],
  visibility = ["//visibility:public"],
)""",
    sha256 = "92c8e4d408704cd636ae528aeae9b4dd7da8448ae951e76ed93c2700e56d4735",
    strip_prefix = "prysm-testnet-site-5afe7bf22b10a2b65c4c6a7a767280c9f32c49a8",
    url = "https://github.com/prestonvanloon/prysm-testnet-site/archive/5afe7bf22b10a2b65c4c6a7a767280c9f32c49a8.tar.gz",
)

http_archive(
    name = "io_kubernetes_build",
    sha256 = "68aabfdac3d52d30ee03e16ff53aa6fd3d61f1d2502532b4ce792b0f47077800",
    strip_prefix = "repo-infra-24cc8eb96f265a7dc0c05e78288a41bbfb0fef88",
    url = "https://github.com/kubernetes/repo-infra/archive/24cc8eb96f265a7dc0c05e78288a41bbfb0fef88.tar.gz",
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
    sha256 = "386335fc3b055fad37088bd821929ff684bc00bb1a74e044e4b377ebd6e88fce",
    # File names are normally too long, see: https://github.com/ethereum/eth2.0-spec-tests/issues/15
    url = "https://prysmaticlabs.com/uploads/v0.8.3_general_spectests.tar.gz",
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
    sha256 = "7ab89a364796e3f8a9af84750c241e9c9e2170a34c1a4e160fdfa2cee5b03fb7",
    url = "https://github.com/ethereum/eth2.0-spec-tests/releases/download/v0.8.3/minimal.tar.gz",
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
    sha256 = "6274e3b77f393faf7b17cef10e93244c16316d3b7ae9c6b844501b12f432a7c3",
    url = "https://github.com/ethereum/eth2.0-spec-tests/releases/download/v0.8.3/mainnet.tar.gz",
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
    commit = "09745575a923640154bcf307fba8aedff47f240a",
    remote = "https://github.com/protocolbuffers/protobuf",
    shallow_since = "1558721209 -0700",
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

# External dependencies

go_repository(
    name = "com_github_ethereum_go_ethereum",
    commit = "e72457320e5ab146ea573d01041a1b80b87e1489",
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
    commit = "c54dc95f42bf03c9c35a4209b372af61d49f48c0",
    importpath = "github.com/prysmaticlabs/go-ssz",
)

go_repository(
    name = "com_github_urfave_cli",
    commit = "e6cf83ec39f6e1158ced1927d4ed14578fda8edb",  # v1.21.0
    importpath = "github.com/urfave/cli",
)

go_repository(
    name = "com_github_go_yaml_yaml",
    commit = "51d6538a90f86fe93ac480b35f37b2be17fef232",  # v2.2.2
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
    commit = "a3c39ad62a5d215b78fd95947dc304784faa98f6",  # v0.3.1
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
    commit = "f96df18bf0c217c77f6cc0f9e810a178cea12f38",  # v0.1.1
    importpath = "github.com/multiformats/go-multiaddr",
)

go_repository(
    name = "com_github_ipfs_go_log",
    commit = "91b837264c0f35dd4e2be341d711316b91d3573d",  # v0.0.1
    importpath = "github.com/ipfs/go-log",
)

go_repository(
    name = "com_github_multiformats_go_multihash",
    commit = "249ead2008065c476a2ee45e8e75e8b85d846a72",  # v0.0.8
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
    commit = "f4c9af195c69379f1cf284dba31985482a56f78e",  # v0.1.3
    importpath = "github.com/libp2p/go-libp2p-peerstore",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_circuit",
    commit = "0305622f3f146485f0ff6df0ae6c010787331ca7",  # v0.1.3
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
    commit = "a5c136c9742a7057dd70ba3af384a5bf0b9f17bc",  # v0.1.0
    importpath = "github.com/multiformats/go-multiaddr-net",
)

go_repository(
    name = "com_github_minio_blake2b_simd",
    commit = "3f5f724cb5b182a5c278d6d3d55b40e7f8c2efb4",
    importpath = "github.com/minio/blake2b-simd",
)

go_repository(
    name = "com_github_mattn_go_colorable",
    commit = "8029fb3788e5a4a9c00e415f586a6d033f5d38b3",  # v0.1.2
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
    commit = "7c3f577d99debb69c3b68be35fe14d9445a6569c",  # v0.2.0
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
    commit = "1dc239722b2ba3784472fb5301f62640fa5a8bc3",  # v0.1.3
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
    commit = "c50c291a61bceccb914366d93eb24f58594e9134",  # v0.0.4
    importpath = "github.com/libp2p/go-libp2p-nat",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_dns",
    commit = "fc9a9eabad5035b23eb95c167621aee7f0083b34",  # v0.1.0
    importpath = "github.com/multiformats/go-multiaddr-dns",
)

go_repository(
    name = "com_github_whyrusleeping_go_logging",
    commit = "0457bb6b88fc1973573aaf6b5145d8d3ae972390",
    importpath = "github.com/whyrusleeping/go-logging",
)

go_repository(
    name = "com_github_mattn_go_isatty",
    commit = "e1f7b56ace729e4a73a29a6b4fac6cd5fcda7ab3",  # v0.0.9
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
    commit = "1f5b3acc846b2c8ce4c4e713296af74f5c24df55",  # v0.0.1
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
    commit = "78f94f4dc7b89e60f4ef983cc661f8e793e8c5ed",  # v0.1.1
    importpath = "github.com/libp2p/go-ws-transport",
)

go_repository(
    name = "org_golang_x_crypto",
    commit = "4def268fd1a49955bfb3dda92fe3db4f924f2285",
    importpath = "golang.org/x/crypto",
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
    name = "com_github_libp2p_go_libp2p_blankhost",
    commit = "da3b45205dfce3ef3926054ffd5dee76f5903382",  # v0.1.4
    importpath = "github.com/libp2p/go-libp2p-blankhost",
)

go_repository(
    name = "io_opencensus_go",
    commit = "7bbec1755a8162b5923fc214a494773a701d506a",  # v0.22.0
    importpath = "go.opencensus.io",
)

go_repository(
    name = "io_opencensus_go_contrib_exporter_jaeger",
    commit = "5b8293c22f362562285c2acbc52f4a1870a47a33",
    importpath = "contrib.go.opencensus.io/exporter/jaeger",
    remote = "http://github.com/census-ecosystem/opencensus-go-exporter-jaeger",
    vcs = "git",
)

go_repository(
    name = "org_golang_google_api",
    commit = "aac82e61c0c8fe133c297b4b59316b9f481e1f0a",  # v0.6.0
    importpath = "google.golang.org/api",
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
    commit = "fd36f4220a901265f90734c3183c5f0c91daa0b8",
    importpath = "github.com/prometheus/client_model",
)

go_repository(
    name = "com_github_prometheus_common",
    commit = "287d3e634a1e550c9e463dd7e5a75a422c614505",  # v0.7.0
    importpath = "github.com/prometheus/common",
)

go_repository(
    name = "com_github_prometheus_procfs",
    commit = "499c85531f756d1129edd26485a5f73871eeb308",  # v0.0.5
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
    name = "com_github_boltdb_bolt",
    commit = "2f1ce7a837dcb8da3ec595b1dac9d0632f0f99e8",  # v1.3.1
    importpath = "github.com/boltdb/bolt",
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
    commit = "9f04364996b415168f0e0d7e9fc82272fbed4005",  # v0.1.1
    importpath = "github.com/libp2p/go-libp2p-pubsub",
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
    commit = "507733cb3c352085b791341655a0b7ef52a17533",  # v0.1.0
    importpath = "github.com/ipfs/go-datastore",
)

go_repository(
    name = "com_github_whyrusleeping_base32",
    commit = "c30ac30633ccdabefe87eb12465113f06f1bab75",
    importpath = "github.com/whyrusleeping/base32",
)

go_repository(
    name = "com_github_ipfs_go_cid",
    commit = "9bb7ea69202c6c9553479eb355ab8a8a97d43a2e",  # v0.0.3
    importpath = "github.com/ipfs/go-cid",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_record",
    build_file_proto_mode = "disable_global",
    commit = "3f535b1abcdf698e11ac16f618c2e64c4e5a114a",  # v0.1.1
    importpath = "github.com/libp2p/go-libp2p-record",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_routing",
    commit = "f4ece6c1baa8e77ee488b25014fcb1059955ed0f",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-routing",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_kbucket",
    commit = "8b77351e0f784a5f71749d23000897c8aee71a76",  # v0.2.1
    importpath = "github.com/libp2p/go-libp2p-kbucket",
)

go_repository(
    name = "com_github_ipfs_go_todocounter",
    commit = "bc75efcf13e6e50fbba27679ba5451585d70c954",  # v0.0.1
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
    commit = "7f827b33c0f158ec5dfbba01bb0b14a4541fd81d",  # v0.5.3
    importpath = "github.com/hashicorp/golang-lru",
)

go_repository(
    name = "com_github_ipfs_go_ipfs_addr",
    commit = "ac4881d4db36effbbeebf93d9172fcb20ed04c15",  # v0.0.1
    importpath = "github.com/ipfs/go-ipfs-addr",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_discovery",
    commit = "d248d63b0af8c023307da18ad7000a12020e06f0",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-discovery",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_autonat",
    commit = "3464f9b4f7bfbd7bb008813eacb626c7ab7fb9a3",  # v0.1.0
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
    commit = "0c47f9da00011ea9a8717671127ac21625c7a6c0",
    importpath = "k8s.io/client-go",
)

go_repository(
    name = "io_k8s_apimachinery",
    build_file_proto_mode = "disable_global",
    commit = "bfcf53abc9f82bad3e534fcb1c36599d3c989ebf",
    importpath = "k8s.io/apimachinery",
)

go_repository(
    name = "io_k8s_klog",
    commit = "9be023858d57e1beb4d7c29fa54093cea2cf9583",
    importpath = "k8s.io/klog",
)

go_repository(
    name = "com_github_google_gofuzz",
    commit = "f140a6486e521aad38f5917de355cbf147cc0496",  # v1.0.0
    importpath = "github.com/google/gofuzz",
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
    commit = "51d6538a90f86fe93ac480b35f37b2be17fef232",  # v2.2.2
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
    commit = "264def2dd949cdb8a803bb9f50fa29a67b798a6a",  # v0.46.3
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
    commit = "f66e05c21cd224e01c8a3ee7bc867aa79439e207",  # v1.8.0
    importpath = "github.com/go-stack/stack",
)

go_repository(
    name = "com_github_rs_cors",
    commit = "9a47f48565a795472d43519dd49aac781f3034fb",  # v1.6.0
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
    commit = "27936f6d90f9c8e1145f11ed52ffffbfdb9e0af7",
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
    commit = "384647d290e2e4a55a14b1b7ef1b7e66293a2c33",  # v0.12.0
    importpath = "github.com/apache/thrift",
)

go_repository(
    name = "com_github_grpc_ecosystem_go_grpc_prometheus",
    commit = "c225b8c3b01faf2899099b768856a9e916e5087b",  # v1.2.0
    importpath = "github.com/grpc-ecosystem/go-grpc-prometheus",
)

go_repository(
    name = "com_github_karlseguin_ccache",
    commit = "ec06cd93a07565b373789b0078ba88fe697fddd9",
    importpath = "github.com/karlseguin/ccache",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_connmgr",
    commit = "b46e9bdbcd8436b4fe4b30a53ec913c07e5e09c9",  # v0.1.1
    importpath = "github.com/libp2p/go-libp2p-connmgr",
)

go_repository(
    name = "com_github_joonix_log",
    commit = "13fe31bbdd7a6f706b9114e188cdb53856be4d64",
    importpath = "github.com/joonix/log",
)

go_repository(
    name = "grpc_ecosystem_grpc_gateway",
    commit = "740ef2ee80c49ed4a272e8c3b54ebf352109f572",
    importpath = "github.com/grpc-ecosystem/grpc-gateway",
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
    commit = "d204016fc64589d0ec2ba965dd686d768d7ab34a",  # v0.2.2
    importpath = "github.com/libp2p/go-libp2p-core",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_testing",
    commit = "1fa303da162dc57872d8fc553497f7602aa11c10",  # v0.1.0
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
    name = "com_github_libp2p_go_yamux",
    commit = "663972181d409e7263040f0b668462f87c85e1bd",  # v1.2.3
    importpath = "github.com/libp2p/go-yamux",
)

go_repository(
    name = "com_github_libp2p_go_nat",
    commit = "d13fdefb3bbb2fde2c6fc090a7ea992cec8b26df",  # v0.0.3
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
    commit = "a6479422943a734f4a0ba2729095fb20c6a233b2",
    importpath = "github.com/prysmaticlabs/go-bitfield",
)

load("@com_github_prysmaticlabs_go_ssz//:deps.bzl", "go_ssz_dependencies")

go_ssz_dependencies()

go_repository(
    name = "org_golang_google_grpc",
    build_file_proto_mode = "disable",
    commit = "1d89a3c832915b2314551c1d2a506874d62e53f7",  # v1.22.0
    importpath = "google.golang.org/grpc",
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
    commit = "ab0dd09aa10e2952b28e12ecd35681b20463ebab",  # v0.3.1
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
    commit = "4cd0c284b15f1735b8cc247df097d262b8903f9f",
    importpath = "sigs.k8s.io/yaml",
)

go_repository(
    name = "com_github_google_go_cmp",
    commit = "2d0692c2e9617365a95b295612ac0d4415ba4627",  # v0.3.1
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
    name = "com_github_googleapis_gnostic",
    commit = "25d8b0b6698593f520d9d8dc5a88e6b16ca9ecc0",
    importpath = "github.com/googleapis/gnostic",
)

go_repository(
    name = "com_github_prysmaticlabs_ethereumapis",
    commit = "c46adef987f5cc5af5463367754157035181e24c",
    importpath = "github.com/prysmaticlabs/ethereumapis",
)

go_repository(
    name = "com_github_cloudflare_roughtime",
    commit = "6b7e31ac9cb2d6048096585d2e8563ee60b28f84",
    importpath = "github.com/cloudflare/roughtime",
)

go_repository(
    name = "com_googlesource_roughtime_roughtime_git",
    build_file_generation = "on",
    commit = "51f6971f5f06ec101e5fbcabe5a49477708540f3",
    importpath = "roughtime.googlesource.com/roughtime.git",
)

go_repository(
    name = "com_github_paulbellamy_ratecounter",
    commit = "a803f0e4f07116687bb30965e8f7d0c32981b63c",
    importpath = "github.com/paulbellamy/ratecounter",
)
