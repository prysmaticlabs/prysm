load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "bazel_skylib",
    sha256 = "2ea8a5ed2b448baf4a6855d3ce049c4c452a6470b1efd1504fdb7c1c134d220a",
    strip_prefix = "bazel-skylib-0.8.0",
    url = "https://github.com/bazelbuild/bazel-skylib/archive/0.8.0.tar.gz",
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "f04d2373bcaf8aa09bccb08a98a57e721306c8f6043a2a0ee610fd6853dcde3d",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.18.6/rules_go-0.18.6.tar.gz",
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
    sha256 = "aed1c249d4ec8f703edddf35cbe9dfaca0b5f5ea6e4cd9e83e99f3b0d1136c3d",
    strip_prefix = "rules_docker-0.7.0",
    url = "https://github.com/bazelbuild/rules_docker/archive/v0.7.0.tar.gz",
)

http_archive(
    name = "build_bazel_rules_nodejs",
    sha256 = "1db950bbd27fb2581866e307c0130983471d4c3cd49c46063a2503ca7b6770a4",
    urls = ["https://github.com/bazelbuild/rules_nodejs/releases/download/0.29.0/rules_nodejs-0.29.0.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_k8s",
    sha256 = "f37af27b3825dbaa811bcf4c3fcab581437fc0bd777e86468f19604ca2e99c6b",
    strip_prefix = "rules_k8s-60571086ea6e10b1ddd2512d5c0fd32d01fa5701",
    url = "https://github.com/bazelbuild/rules_k8s/archive/60571086ea6e10b1ddd2512d5c0fd32d01fa5701.tar.gz",
)

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

load("@build_bazel_rules_nodejs//:defs.bzl", "node_repositories", "yarn_install")

node_repositories()

yarn_install(
    name = "npm",
    package_json = "//:package.json",
    yarn_lock = "//:yarn.lock",
)

# This requires rules_docker to be fully instantiated before it is pulled in.
load("@io_bazel_rules_k8s//k8s:k8s.bzl", "k8s_defaults", "k8s_repositories")

k8s_repositories()

[k8s_defaults(
    name = "k8s_" + kind,
    cluster = "minikube",  # DO NOT CHANGE THIS!
    kind = kind,
) for kind in [
    "cluster_role",
    "configmap",
    "deploy",
    "ingress",
    "issuer",
    "job",
    "gateway",
    "namespace",
    "pod",
    "priority_class",
    "secret",
    "service",
    "service_account",
]]

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
    sha256 = "dd02a62c2a458295f561e280411b04d2efbd97e4954986a401a9a1334cc32cc3",
    strip_prefix = "repo-infra-1b2ddaf3fb8775a5d0f4e28085cf846f915977a8",
    url = "https://github.com/kubernetes/repo-infra/archive/1b2ddaf3fb8775a5d0f4e28085cf846f915977a8.tar.gz",
)

http_archive(
    name = "eth2_spec_tests",
    build_file_content = """
filegroup(
    name = "test_data",
    srcs = glob([
        "**/*.yaml",
    ]),
    visibility = ["//visibility:public"],
)
    """,
    sha256 = "56847989737e816ab7d23f3bb2422347dfa81271bae81a94de512c01461fab25",
    url = "https://github.com/prysmaticlabs/eth2.0-spec-tests/releases/download/v0.7.1/base64_encoded_archive.tar.gz",
)

http_archive(
    name = "com_github_bazelbuild_buildtools",
    strip_prefix = "buildtools-bf564b4925ab5876a3f64d8b90fab7f769013d42",
    url = "https://github.com/bazelbuild/buildtools/archive/bf564b4925ab5876a3f64d8b90fab7f769013d42.zip",
)

load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")

buildifier_dependencies()

http_archive(
    name = "com_github_prysmaticlabs_go_ssz",
    sha256 = "f6fd5d623a988337810b956ddaf612dce771d9d0f9256934c8f4b1379f1cb2f6",
    strip_prefix = "go-ssz-2e84733edbac32aca6d47feafc4441e43b10047f",
    url = "https://github.com/prysmaticlabs/go-ssz/archive/2e84733edbac32aca6d47feafc4441e43b10047f.tar.gz",
)

load("@com_github_prysmaticlabs_go_ssz//:deps.bzl", "go_ssz_dependencies")

go_ssz_dependencies()

go_repository(
    name = "com_github_golang_mock",
    commit = "51421b967af1f557f93a59e0057aaf15ca02e29c",  # v1.2.0
    importpath = "github.com/golang/mock",
)

# External dependencies

go_repository(
    name = "com_github_ethereum_go_ethereum",
    commit = "099afb3fd89784f9e3e594b7c2ed11335ca02a9b",
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
    name = "com_github_urfave_cli",
    commit = "cfb38830724cc34fedffe9a2a29fb54fa9169cd1",  # v1.20.0
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
    commit = "e69d17141ca58ba6afbf13098e90c9377938e590",  # v0.2.0
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
    commit = "5b1de2f51ff2368d5ce94a659f15ef26be273cd0",  # v0.0.4
    importpath = "github.com/multiformats/go-multiaddr",
)

go_repository(
    name = "com_github_ipfs_go_log",
    commit = "91b837264c0f35dd4e2be341d711316b91d3573d",  # v0.0.1
    importpath = "github.com/ipfs/go-log",
)

go_repository(
    name = "com_github_multiformats_go_multihash",
    commit = "0e239d8fa37b597bd150660e5b6845570aa5b833",  # v0.0.6
    importpath = "github.com/multiformats/go-multihash",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_swarm",
    commit = "4a42085d76199475c2014c4557895d42d2ff85d9",  # v0.1.1
    importpath = "github.com/libp2p/go-libp2p-swarm",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_host",
    commit = "fb741ff65522f904e7d46f527c9a823f32346f83",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-host",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_peerstore",
    commit = "c11298943ef400535dac08ec6cbeff747cbe7e99",  # v0.1.1
    importpath = "github.com/libp2p/go-libp2p-peerstore",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_circuit",
    commit = "e65c36f3bb806cf658db0f0b612879899e2d28dc",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-circuit",
)

go_repository(
    name = "com_github_coreos_go_semver",
    commit = "6e25b691b0ebe9657dd0ee60d73a9f8716f0c6f5",  # v0.3.0
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
    commit = "bd61b0499a3cfc893a8eb109c5669342b1671881",  # v0.0.1
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
    name = "com_github_whyrusleeping_mdns",
    commit = "ef14215e6b30606f4ce84174ed7a644a05cb1af3",
    importpath = "github.com/whyrusleeping/mdns",
)

go_repository(
    name = "com_github_btcsuite_btcd",
    commit = "306aecffea325e97f513b3ff0cf7895a5310651d",
    importpath = "github.com/btcsuite/btcd",
)

go_repository(
    name = "com_github_minio_sha256_simd",
    commit = "05b4dd3047e5d6e86cb4e0477164b850cd896261",  # v0.1.0
    importpath = "github.com/minio/sha256-simd",
)

go_repository(
    name = "com_github_mr_tron_base58",
    commit = "89529c6904fcd077434931b4eac8b4b2f0991baf",  # v1.1.0
    importpath = "github.com/mr-tron/base58",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_secio",
    build_file_proto_mode = "disable_global",
    commit = "a158134b5708e33fa36545d8ba8e27ea1c8ae54e",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-secio",
)

go_repository(
    name = "com_github_libp2p_go_tcp_transport",
    commit = "415627e90148700bf97890e54b193a42125c3b66",  # v0.1.0
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
    commit = "e7c544d7a325c57bdbd7e9ba9c035a6701c5c7d2",  # v0.0.2
    importpath = "github.com/multiformats/go-multiaddr-dns",
)

go_repository(
    name = "com_github_whyrusleeping_go_logging",
    commit = "0457bb6b88fc1973573aaf6b5145d8d3ae972390",
    importpath = "github.com/whyrusleeping/go-logging",
)

go_repository(
    name = "com_github_mattn_go_isatty",
    commit = "1311e847b0cb909da63b5fecfb5370aa66236465",  # v0.0.8
    importpath = "github.com/mattn/go-isatty",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_transport_upgrader",
    commit = "07ed92ccf9aba3a2e4b2fddc4c17ced060524922",  # v0.1.1
    importpath = "github.com/libp2p/go-libp2p-transport-upgrader",
)

go_repository(
    name = "com_github_libp2p_go_testutil",
    commit = "9a5d4c55819de9fd3e07181003d1e722621f6b84",  # v0.1.0
    importpath = "github.com/libp2p/go-testutil",
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
    name = "com_github_miekg_dns",
    commit = "8fc2e5773bbd308ca2fcc962fd8d25c1bd0f6743",  # v1.1.4
    importpath = "github.com/miekg/dns",
)

go_repository(
    name = "com_github_opentracing_opentracing_go",
    commit = "1949ddbfd147afd4d964a9f00b24eb291e0e7c38",  # v1.0.2
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
    commit = "e1e72e9de974bd926e5c56f83753fba2df402ce5",  # v1.3.0
    importpath = "github.com/sirupsen/logrus",
)

go_repository(
    name = "org_golang_x_sys",
    commit = "a34e9553db1e492c9a76e60db2296ae7e5fbb772",
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
    commit = "bfe829fefc91f676644aee0dc057097c605ae5ab",  # v1.0.5
    importpath = "github.com/jackpal/gateway",
)

go_repository(
    name = "com_github_whyrusleeping_multiaddr_filter",
    commit = "e903e4adabd70b78bc9293b6ee4f359afb3f9f59",
    importpath = "github.com/whyrusleeping/multiaddr-filter",
)

go_repository(
    name = "com_github_libp2p_go_ws_transport",
    commit = "6efd965516262a6b6e46ea987b94904ef13e59bc",  # v0.1.0
    importpath = "github.com/libp2p/go-ws-transport",
)

go_repository(
    name = "org_golang_x_crypto",
    commit = "8dd112bcdc25174059e45e07517d9fc663123347",
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
    commit = "66b9c49e59c6c48f0ffce28c2d8b8a5678502c6d",  # v1.4.0
    importpath = "github.com/gorilla/websocket",
)

go_repository(
    name = "com_github_syndtr_goleveldb",
    commit = "9d007e481048296f09f59bd19bb7ae584563cd95",  # v1.0.0
    importpath = "github.com/syndtr/goleveldb",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_blankhost",
    commit = "a50d1c7d55c7bbc52879616e7e0c8cdf38747c1a",  # v0.1.3
    importpath = "github.com/libp2p/go-libp2p-blankhost",
)

go_repository(
    name = "com_github_steakknife_hamming",
    commit = "c99c65617cd3d686aea8365fe563d6542f01d940",
    importpath = "github.com/steakknife/hamming",
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
    commit = "31bed53e4047fd6c510e43a941f90cb31be0972a",  # v0.6.0
    importpath = "github.com/prometheus/common",
)

go_repository(
    name = "com_github_prometheus_procfs",
    commit = "fc7f7514de80507d58d5359759cb9e5fb48b35d4",  # v0.0.2
    importpath = "github.com/prometheus/procfs",
)

go_repository(
    name = "com_github_beorn7_perks",
    commit = "4ded152d4a3e2847f17f185a27b2041ae7b63979",  # v1.0.0
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
    commit = "49274b0e8aecdf6cad59d768e5702ff00aa48488",  # v0.1.0
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
    commit = "874e3d3fa068272afc6006b29c51ec8529b1b5ea",  # v0.1.1
    importpath = "github.com/libp2p/go-libp2p-kad-dht",
)

go_repository(
    name = "com_github_ipfs_go_datastore",
    commit = "aa9190c18f1576be98e974359fd08c64ca0b5a94",  # v0.0.5
    importpath = "github.com/ipfs/go-datastore",
)

go_repository(
    name = "com_github_whyrusleeping_base32",
    commit = "c30ac30633ccdabefe87eb12465113f06f1bab75",
    importpath = "github.com/whyrusleeping/base32",
)

go_repository(
    name = "com_github_ipfs_go_cid",
    commit = "b1cc3e404d48791056147f118ea7e7ea94eb946f",  # v0.0.2
    importpath = "github.com/ipfs/go-cid",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_record",
    build_file_proto_mode = "disable_global",
    commit = "4837430afd8f3864d4805d7a1675521abb1096b4",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-record",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_routing",
    commit = "f4ece6c1baa8e77ee488b25014fcb1059955ed0f",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-routing",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_kbucket",
    commit = "3752ea0128fd84b4fef0a66739b8ca95c8a471b6",  # v0.2.0
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
    commit = "7087cb70de9f7a8bc0a10c375cb0d2280a8edf9c",
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
    commit = "f55edac94c9bbba5d6182a4be46d86a2c9b5b50e",
    importpath = "github.com/konsorten/go-windows-terminal-sequences",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_interface_conn",
    commit = "c7cda99284db0bea441058da8fd1f1373c763ed6",
    importpath = "github.com/libp2p/go-libp2p-interface-conn",
)

go_repository(
    name = "io_k8s_client_go",
    commit = "8abb21031259350aad0799bb42ba213ee8bb3399",
    importpath = "k8s.io/client-go",
)

go_repository(
    name = "io_k8s_apimachinery",
    build_file_proto_mode = "disable_global",
    commit = "4a9a8137c0a17bc4594f544987b3f0d48b2e3d3a",
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
    commit = "b7bd5f2d334ce968edc54f5fdb2ac67ce39c56d5",
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
    commit = "298182f68c66c05229eb03ac171abe6e309ee79a",  # v1.0.3
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
    commit = "cf81fad90a1a1de334c4fc27e23eb9a4224b627a",  # v0.41.0
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
    name = "com_github_rjeczalik_notify",
    commit = "69d839f37b13a8cb7a78366f7633a4071cb43be7",  # v0.9.2
    importpath = "github.com/rjeczalik/notify",
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
    name = "org_gonum_v1_gonum",
    commit = "70a1e933af10e87000d2ccabdd509b87d8626153",
    importpath = "gonum.org/v1/gonum",
)

go_repository(
    name = "org_golang_x_exp",
    commit = "438050ddec5e7f808979ed57d041cebbc8e2d8a9",
    importpath = "golang.org/x/exp",
)

go_repository(
    name = "com_github_prestonvanloon_go_recaptcha",
    commit = "0834cef6e8bd3a7ebdb3ac7def9440ee47d501a4",
    importpath = "github.com/prestonvanloon/go-recaptcha",
)

go_repository(
    name = "com_github_phoreproject_bls",
    commit = "b495094dc72c7043b549f511a798391201624b14",
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
    commit = "c250d6563d4d4c20252cd865923440e829844f4e",  # v1.0.0
    importpath = "github.com/grpc-ecosystem/go-grpc-middleware",
)

go_repository(
    name = "com_github_apache_thrift",
    commit = "384647d290e2e4a55a14b1b7ef1b7e66293a2c33",  # v0.12.0
    importpath = "github.com/apache/thrift",
)

go_repository(
    name = "com_github_grpc_ecosystem_go_grpc_prometheus",
    commit = "502116f1a0a0c1140aab04fd3787489209b357d3",  # v1.2.0
    importpath = "github.com/grpc-ecosystem/go-grpc-prometheus",
)

go_repository(
    name = "com_github_karlseguin_ccache",
    commit = "ec06cd93a07565b373789b0078ba88fe697fddd9",
    importpath = "github.com/karlseguin/ccache",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_connmgr",
    commit = "152025a671fcc297333095f8e4afc98d90b30df7",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-connmgr",
)

go_repository(
    name = "com_github_joonix_log",
    commit = "13fe31bbdd7a6f706b9114e188cdb53856be4d64",
    importpath = "github.com/joonix/log",
)

go_repository(
    name = "grpc_ecosystem_grpc_gateway",
    commit = "8fd5fd9d19ce68183a6b0934519dfe7fe6269612",  # v1.9.0
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
    commit = "786c4f4e0f0af96fb69223268da4d0bf123841d8",  # v0.0.6
    importpath = "github.com/libp2p/go-libp2p-core",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_testing",
    commit = "6d4ca71943f35271918e28f9a9950002e17b4f16",  # v0.0.4
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
    commit = "7d8102a98552c80f8a5ccb9c01e670fac17fd6df",  # v0.0.1
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
    commit = "4afad1f6206cb9222914f2ec6ab9d0b414705c54",
    importpath = "github.com/libp2p/go-eventbus",
)
