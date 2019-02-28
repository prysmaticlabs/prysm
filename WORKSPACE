load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "bazel_skylib",
    url = "https://github.com/bazelbuild/bazel-skylib/archive/0.7.0.tar.gz",
    sha256 = "2c62d8cd4ab1e65c08647eb4afe38f51591f43f7f0885e7769832fa137633dcb",
    strip_prefix = "bazel-skylib-0.7.0",
)

http_archive(
    name = "io_bazel_rules_go",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.17.1/rules_go-0.17.1.tar.gz",
    sha256 = "6776d68ebb897625dead17ae510eac3d5f6342367327875210df44dbe2aeeb19",
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "3c681998538231a2d24d0c07ed5a7658cb72bfb5fd4bf9911157c0e9ac6a2687",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.17.0/bazel-gazelle-0.17.0.tar.gz"],
)

http_archive(
    name = "com_github_atlassian_bazel_tools",
    sha256 = "e7d0c0e2963a7f9cb2c377e241502119dae24909708adef1918e8dcb70ae9e8c",
    strip_prefix = "bazel-tools-6fef37f33dfa0189be9df4d3d60e6291bfe71177",
    urls = ["https://github.com/atlassian/bazel-tools/archive/6fef37f33dfa0189be9df4d3d60e6291bfe71177.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_docker",
    url = "https://github.com/bazelbuild/rules_docker/archive/v0.7.0.tar.gz",
    strip_prefix = "rules_docker-0.7.0",
    sha256 = "aed1c249d4ec8f703edddf35cbe9dfaca0b5f5ea6e4cd9e83e99f3b0d1136c3d",
)

http_archive(
    name = "build_bazel_rules_nodejs",
    urls = ["https://github.com/bazelbuild/rules_nodejs/releases/download/0.26.0/rules_nodejs-0.26.0.tar.gz"],
    sha256 = "5c86b055c57e15bf32d9009a15bcd6d8e190c41b1ff2fb18037b75e0012e4e7c",
)

http_archive(
    name = "io_bazel_rules_k8s",
    sha256 = "828fb1ac4c44280be95306b885a326e40110eeba50bffa05e72ddd3b5cdc5d33",
    strip_prefix = "rules_k8s-2206972072d64e5d2d966d81cc6c5fb77fd58dcb",
    url = "https://github.com/bazelbuild/rules_k8s/archive/2206972072d64e5d2d966d81cc6c5fb77fd58dcb.tar.gz",
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
load("@io_bazel_rules_k8s//k8s:k8s.bzl", "k8s_repositories", "k8s_defaults")

k8s_repositories()

[k8s_defaults(
    name = "k8s_" + kind,
    cluster = "minikube",
    kind = kind,
) for kind in [
    "cluster_role",
    "configmap",
    "deploy",
    "ingress",
    "job",
    "namespace",
    "pod",
    "priority_class",
    "secret",
    "service",
    "service_account",
]]

load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains()

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
    url = "https://github.com/prestonvanloon/prysm-testnet-site/archive/ee6c0659223775395cd5e592a6c0e5c6d4abe7e1.tar.gz",
    strip_prefix = "prysm-testnet-site-ee6c0659223775395cd5e592a6c0e5c6d4abe7e1",
    sha256 = "c02fcf85202d184c6fa4e78fa16295729affbfd60e4827728ec189d7e3895d90",
    build_file_content = """
proto_library(
  name = "faucet_proto",
  srcs = ["src/proto/faucet.proto"],
  visibility = ["//visibility:public"],
)""",
)

http_archive(
    name = "io_kubernetes_build",
    sha256 = "b4e7819861f2ec89b7309bd0c44fb3348c3a4a8ee494ec7668edb3960ff11814",
    strip_prefix = "repo-infra-4ce715fbe67d8fbed05ec2bb47a148e754100a4b",
    url = "https://github.com/kubernetes/repo-infra/archive/4ce715fbe67d8fbed05ec2bb47a148e754100a4b.tar.gz",
)

go_repository(
    name = "com_github_golang_mock",
    commit = "c20582278a829e4b3259747a3ce0eceb1763ee13",
    importpath = "github.com/golang/mock",
)

# External dependencies

go_repository(
    name = "com_github_ethereum_go_ethereum",
    commit = "2ed4a57e9522dc2a023577f0b92ff623c3412303",
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
    importpath = "github.com/urfave/cli",
    tag = "v1.20.0",
)

go_repository(
    name = "com_github_go_yaml_yaml",
    importpath = "github.com/go-yaml/yaml",
    tag = "v2.2.2",
)

go_repository(
    name = "com_github_x_cray_logrus_prefixed_formatter",
    importpath = "github.com/x-cray/logrus-prefixed-formatter",
    tag = "v0.5.2",
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
    importpath = "github.com/libp2p/go-libp2p",
    tag = "gx/v6.0.29",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_peer",
    importpath = "github.com/libp2p/go-libp2p-peer",
    tag = "gx/v2.4.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_crypto",
    build_file_proto_mode = "disable_global",
    tag = "gx/v2.0.5",
    importpath = "github.com/libp2p/go-libp2p-crypto",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr",
    tag = "v0.0.1",
    importpath = "github.com/multiformats/go-multiaddr",
)

go_repository(
    name = "com_github_ipfs_go_log",
    importpath = "github.com/ipfs/go-log",
    tag = "v1.5.7",
)

go_repository(
    name = "com_github_multiformats_go_multihash",
    importpath = "github.com/multiformats/go-multihash",
    tag = "gx/v1.0.10",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_swarm",
    importpath = "github.com/libp2p/go-libp2p-swarm",
    tag = "gx/v3.0.22",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_host",
    importpath = "github.com/libp2p/go-libp2p-host",
    tag = "gx/v3.0.15",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_peerstore",
    importpath = "github.com/libp2p/go-libp2p-peerstore",
    tag = "gx/v2.0.6",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_circuit",
    importpath = "github.com/libp2p/go-libp2p-circuit",
    tag = "gx/v2.3.2",
)

go_repository(
    name = "com_github_coreos_go_semver",
    importpath = "github.com/coreos/go-semver",
    tag = "v0.2.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_interface_connmgr",
    importpath = "github.com/libp2p/go-libp2p-interface-connmgr",
    tag = "gx/v0.0.21",
)

go_repository(
    name = "com_github_libp2p_go_conn_security_multistream",
    importpath = "github.com/libp2p/go-conn-security-multistream",
    tag = "gx/v0.1.15",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_metrics",
    importpath = "github.com/libp2p/go-libp2p-metrics",
    tag = "gx/v2.1.7",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_net",
    importpath = "github.com/libp2p/go-libp2p-net",
    tag = "gx/v3.0.15",
)

go_repository(
    name = "com_github_whyrusleeping_mafmt",
    importpath = "github.com/whyrusleeping/mafmt",
    tag = "v1.2.8",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_net",
    tag = "v0.0.1",
    importpath = "github.com/multiformats/go-multiaddr-net",
)

go_repository(
    name = "com_github_agl_ed25519",
    commit = "5312a61534124124185d41f09206b9fef1d88403",
    importpath = "github.com/agl/ed25519",
)

go_repository(
    name = "com_github_minio_blake2b_simd",
    commit = "3f5f724cb5b182a5c278d6d3d55b40e7f8c2efb4",
    importpath = "github.com/minio/blake2b-simd",
)

go_repository(
    name = "com_github_gxed_hashland",
    tag = "v0.0.1",
    importpath = "github.com/gxed/hashland",
)

go_repository(
    name = "com_github_mattn_go_colorable",
    tag = "v0.1.1",
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
    commit = "2d45a736cd16732fe6a57563cc20d8b035193e58",
    importpath = "github.com/minio/sha256-simd",
)

go_repository(
    name = "com_github_mr_tron_base58",
    importpath = "github.com/mr-tron/base58",
    tag = "v1.1.0",
)

go_repository(
    name = "com_github_whyrusleeping_go_smux_yamux",
    importpath = "github.com/whyrusleeping/go-smux-yamux",
    tag = "v2.0.8",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_secio",
    build_file_proto_mode = "disable_global",
    importpath = "github.com/libp2p/go-libp2p-secio",
    tag = "gx/v2.0.17",
)

go_repository(
    name = "com_github_libp2p_go_tcp_transport",
    importpath = "github.com/libp2p/go-tcp-transport",
    tag = "gx/v2.0.16",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_protocol",
    importpath = "github.com/libp2p/go-libp2p-protocol",
    tag = "gx/v1.0.0",
)

go_repository(
    name = "com_github_jbenet_goprocess",
    commit = "b497e2f366b8624394fb2e89c10ab607bebdde0b",
    importpath = "github.com/jbenet/goprocess",
)

go_repository(
    name = "com_github_multiformats_go_multistream",
    tag = "v0.0.1",
    importpath = "github.com/multiformats/go-multistream",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_loggables",
    importpath = "github.com/libp2p/go-libp2p-loggables",
    tag = "gx/v1.1.24",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_nat",
    importpath = "github.com/libp2p/go-libp2p-nat",
    tag = "gx/v0.8.8",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_dns",
    importpath = "github.com/multiformats/go-multiaddr-dns",
    tag = "gx/v0.2.5",
)

go_repository(
    name = "com_github_fd_go_nat",
    importpath = "github.com/fd/go-nat",
    tag = "v1.0.0",
)

go_repository(
    name = "com_github_whyrusleeping_go_logging",
    commit = "0457bb6b88fc1973573aaf6b5145d8d3ae972390",
    importpath = "github.com/whyrusleeping/go-logging",
)

go_repository(
    name = "com_github_mattn_go_isatty",
    importpath = "github.com/mattn/go-isatty",
    tag = "v0.0.6",
)

go_repository(
    name = "com_github_libp2p_go_stream_muxer",
    importpath = "github.com/libp2p/go-stream-muxer",
    tag = "gx/v3.0.1",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_transport_upgrader",
    importpath = "github.com/libp2p/go-libp2p-transport-upgrader",
    tag = "gx/v0.1.16",
)

go_repository(
    name = "com_github_libp2p_go_testutil",
    importpath = "github.com/libp2p/go-testutil",
    tag = "gx/v1.2.10",
)

go_repository(
    name = "com_github_whyrusleeping_go_smux_multistream",
    importpath = "github.com/whyrusleeping/go-smux-multistream",
    tag = "v2.0.2",
)

go_repository(
    name = "com_github_libp2p_go_maddr_filter",
    importpath = "github.com/libp2p/go-maddr-filter",
    tag = "gx/v1.1.10",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_transport",
    importpath = "github.com/libp2p/go-libp2p-transport",
    tag = "gx/v3.0.15",
)

go_repository(
    name = "com_github_libp2p_go_addr_util",
    importpath = "github.com/libp2p/go-addr-util",
    tag = "gx/v2.0.7",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_interface_pnet",
    importpath = "github.com/libp2p/go-libp2p-interface-pnet",
    tag = "gx/v3.0.0",
)

go_repository(
    name = "com_github_libp2p_go_conn_security",
    importpath = "github.com/libp2p/go-conn-security",
    tag = "gx/v0.1.15",
)

go_repository(
    name = "com_github_whyrusleeping_timecache",
    commit = "cfcb2f1abfee846c430233aef0b630a946e0a5a6",
    importpath = "github.com/whyrusleeping/timecache",
)

go_repository(
    name = "com_github_miekg_dns",
    tag = "v1.1.4",
    importpath = "github.com/miekg/dns",
)

go_repository(
    name = "com_github_opentracing_opentracing_go",
    importpath = "github.com/opentracing/opentracing-go",
    tag = "v1.0.2",
)

go_repository(
    name = "com_github_libp2p_go_reuseport",
    importpath = "github.com/libp2p/go-reuseport",
    tag = "gx/v0.2.0",
)

go_repository(
    name = "com_github_huin_goupnp",
    importpath = "github.com/huin/goupnp",
    tag = "v1.0.0",
)

go_repository(
    name = "com_github_spaolacci_murmur3",
    commit = "f09979ecbc725b9e6d41a297405f65e7e8804acc",
    importpath = "github.com/spaolacci/murmur3",
)

go_repository(
    name = "com_github_jbenet_go_temp_err_catcher",
    commit = "aac704a3f4f27190b4ccc05f303a4931fd1241ff",
    importpath = "github.com/jbenet/go-temp-err-catcher",
)

go_repository(
    name = "com_github_satori_go_uuid",
    importpath = "github.com/satori/go.uuid",
    tag = "v1.2.0",
)

go_repository(
    name = "com_github_sirupsen_logrus",
    importpath = "github.com/sirupsen/logrus",
    tag = "v1.3.0",
)

go_repository(
    name = "org_golang_x_sys",
    commit = "775f8194d0f9e65c46913c7be783d3d95a29333c",
    importpath = "golang.org/x/sys",
)

go_repository(
    name = "com_github_whyrusleeping_yamux",
    tag = "v1.1.5",
    importpath = "github.com/whyrusleeping/yamux",
)

go_repository(
    name = "com_github_libp2p_go_flow_metrics",
    importpath = "github.com/libp2p/go-flow-metrics",
    tag = "gx/v0.2.0",
)

go_repository(
    name = "com_github_libp2p_go_msgio",
    importpath = "github.com/libp2p/go-msgio",
    tag = "gx/v0.0.6",
)

go_repository(
    name = "com_github_jackpal_gateway",
    importpath = "github.com/jackpal/gateway",
    tag = "v1.0.5",
)

go_repository(
    name = "com_github_whyrusleeping_multiaddr_filter",
    commit = "e903e4adabd70b78bc9293b6ee4f359afb3f9f59",
    importpath = "github.com/whyrusleeping/multiaddr-filter",
)

go_repository(
    name = "com_github_libp2p_go_ws_transport",
    importpath = "github.com/libp2p/go-ws-transport",
    tag = "gx/v2.0.15",
)

go_repository(
    name = "org_golang_x_crypto",
    commit = "31a38585487a4b1fd6ff4f8f3db26f1fb296ac82",
    importpath = "golang.org/x/crypto",
)

go_repository(
    name = "com_github_jackpal_go_nat_pmp",
    importpath = "github.com/jackpal/go-nat-pmp",
    tag = "v1.0.1",
)

go_repository(
    name = "com_github_libp2p_go_reuseport_transport",
    importpath = "github.com/libp2p/go-reuseport-transport",
    tag = "gx/v0.2.0",
)

go_repository(
    name = "com_github_libp2p_go_sockaddr",
    importpath = "github.com/libp2p/go-sockaddr",
    tag = "gx/v1.0.3",
)

go_repository(
    name = "com_github_whyrusleeping_go_notifier",
    commit = "097c5d47330ff6a823f67e3515faa13566a62c6f",
    importpath = "github.com/whyrusleeping/go-notifier",
)

go_repository(
    name = "com_github_gorilla_websocket",
    importpath = "github.com/gorilla/websocket",
    tag = "v1.4.0",
)

go_repository(
    name = "com_github_whyrusleeping_go_smux_multiplex",
    importpath = "github.com/whyrusleeping/go-smux-multiplex",
    tag = "v3.0.16",
)

go_repository(
    name = "com_github_gxed_eventfd",
    commit = "80a92cca79a8041496ccc9dd773fcb52a57ec6f9",
    importpath = "github.com/gxed/eventfd",
)

go_repository(
    name = "com_github_gxed_goendian",
    commit = "0f5c6873267e5abf306ffcdfcfa4bf77517ef4a7",
    importpath = "github.com/gxed/GoEndian",
)

go_repository(
    name = "com_github_syndtr_goleveldb",
    tag = "v1.0.0",
    importpath = "github.com/syndtr/goleveldb",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_blankhost",
    importpath = "github.com/libp2p/go-libp2p-blankhost",
    tag = "gx/v0.3.15",
)

go_repository(
    name = "com_github_steakknife_hamming",
    importpath = "github.com/steakknife/hamming",
    tag = "0.2.5",
)

go_repository(
    name = "io_opencensus_go",
    importpath = "go.opencensus.io",
    tag = "v0.19.0",
)

go_repository(
    name = "org_golang_google_api",
    importpath = "google.golang.org/api",
    tag = "v0.1.0",
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
    importpath = "github.com/prometheus/client_golang",
    tag = "v0.9.2",
)

go_repository(
    name = "com_github_prometheus_client_model",
    commit = "fd36f4220a901265f90734c3183c5f0c91daa0b8",
    importpath = "github.com/prometheus/client_model",
)

go_repository(
    name = "com_github_prometheus_common",
    tag = "v0.2.0",
    importpath = "github.com/prometheus/common",
)

go_repository(
    name = "com_github_prometheus_procfs",
    commit = "bbced9601137e764853b2fad7ec3e2dc4c504e02",
    importpath = "github.com/prometheus/procfs",
)

go_repository(
    name = "com_github_prometheus_prometheus",
    tag = "v2.7.1",
    importpath = "github.com/prometheus/prometheus",
)

go_repository(
    name = "com_github_beorn7_perks",
    commit = "3a771d992973f24aa725d07868b467d1ddfceafb",
    importpath = "github.com/beorn7/perks",
)

go_repository(
    name = "com_github_matttproud_golang_protobuf_extensions",
    importpath = "github.com/matttproud/golang_protobuf_extensions",
    tag = "v1.0.1",
)

go_repository(
    name = "com_github_boltdb_bolt",
    importpath = "github.com/boltdb/bolt",
    tag = "v1.3.1",
)

go_repository(
    name = "com_github_pborman_uuid",
    importpath = "github.com/pborman/uuid",
    tag = "v1.2.0",
)

go_repository(
    name = "com_github_libp2p_go_buffer_pool",
    tag = "gx/v0.1.3",
    importpath = "github.com/libp2p/go-buffer-pool",
)

go_repository(
    name = "com_github_libp2p_go_mplex",
    importpath = "github.com/libp2p/go-mplex",
    tag = "gx/v0.2.30",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_pubsub",
    build_file_proto_mode = "disable_global",
    importpath = "github.com/libp2p/go-libp2p-pubsub",
    tag = "gx/v0.11.10",
)

go_repository(
    name = "com_github_ipfs_go_ipfs_util",
    importpath = "github.com/ipfs/go-ipfs-util",
    tag = "v1.2.8",
)

go_repository(
    name = "com_github_google_uuid",
    importpath = "github.com/google/uuid",
    tag = "v1.1.1",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_kad_dht",
    build_file_proto_mode = "disable_global",
    importpath = "github.com/libp2p/go-libp2p-kad-dht",
    tag = "gx/v4.4.12",
)

go_repository(
    name = "com_github_ipfs_go_datastore",
    importpath = "github.com/ipfs/go-datastore",
    tag = "v3.2.0",
)

go_repository(
    name = "com_github_whyrusleeping_base32",
    commit = "c30ac30633ccdabefe87eb12465113f06f1bab75",
    importpath = "github.com/whyrusleeping/base32",
)

go_repository(
    name = "com_github_ipfs_go_cid",
    importpath = "github.com/ipfs/go-cid",
    tag = "v0.9.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_record",
    build_file_proto_mode = "disable_global",
    importpath = "github.com/libp2p/go-libp2p-record",
    tag = "gx/v4.1.7",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_routing",
    importpath = "github.com/libp2p/go-libp2p-routing",
    tag = "gx/v2.7.1",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_kbucket",
    importpath = "github.com/libp2p/go-libp2p-kbucket",
    tag = "gx/v2.2.12",
)

go_repository(
    name = "com_github_jbenet_go_context",
    commit = "d14ea06fba99483203c19d92cfcd13ebe73135f4",
    importpath = "github.com/jbenet/go-context",
)

go_repository(
    name = "com_github_ipfs_go_todocounter",
    importpath = "github.com/ipfs/go-todocounter",
    tag = "v1.0.1",
)

go_repository(
    name = "com_github_whyrusleeping_go_keyspace",
    commit = "5b898ac5add1da7178a4a98e69cb7b9205c085ee",
    importpath = "github.com/whyrusleeping/go-keyspace",
)

go_repository(
    name = "com_github_multiformats_go_multibase",
    importpath = "github.com/multiformats/go-multibase",
    tag = "gx/v0.3.0",
)

go_repository(
    name = "com_github_hashicorp_golang_lru",
    importpath = "github.com/hashicorp/golang-lru",
    tag = "v0.5.1",
)

go_repository(
    name = "com_github_ipfs_go_ipfs_addr",
    importpath = "github.com/ipfs/go-ipfs-addr",
    tag = "v0.1.25",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_discovery",
    commit = "f6df2e8dcd01e739e4c0d06c799c4de736c0aab2",
    importpath = "github.com/libp2p/go-libp2p-discovery",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_autonat",
    commit = "ca1e8793f443d0219b265e2d14bedf8c7ae04c9b",
    importpath = "github.com/libp2p/go-libp2p-autonat",
)

go_repository(
    name = "com_github_konsorten_go_windows_terminal_sequences",
    importpath = "github.com/konsorten/go-windows-terminal-sequences",
    tag = "v1.0.2",
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
    commit = "24818f796faf91cd76ec7bddd72458fbced7a6c1",
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
    importpath = "gopkg.in/yaml.v2",
    tag = "v2.2.2",
)

go_repository(
    name = "com_github_spf13_pflag",
    importpath = "github.com/spf13/pflag",
    tag = "v1.0.3",
)

go_repository(
    name = "com_github_spf13_cobra",
    importpath = "github.com/spf13/cobra",
    tag = "v0.0.3",
)

go_repository(
    name = "com_github_aws_aws_sdk_go",
    importpath = "github.com/aws/aws-sdk-go",
    tag = "v1.16.9",
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
    importpath = "github.com/hashicorp/go-multierror",
    tag = "v1.0.0",
)

go_repository(
    name = "com_github_hashicorp_errwrap",
    importpath = "github.com/hashicorp/errwrap",
    tag = "v1.0.0",
)

go_repository(
    name = "com_google_cloud_go",
    tag = "v0.36.0",
    importpath = "cloud.google.com/go",
)

go_repository(
    name = "com_github_inconshreveable_mousetrap",
    importpath = "github.com/inconshreveable/mousetrap",
    tag = "v1.0.0",
)

go_repository(
    name = "com_github_deckarep_golang_set",
    importpath = "github.com/deckarep/golang-set",
    tag = "v1.7.1",
)

go_repository(
    name = "com_github_go_stack_stack",
    importpath = "github.com/go-stack/stack",
    tag = "v1.8.0",
)

go_repository(
    name = "com_github_rs_cors",
    importpath = "github.com/rs/cors",
    tag = "v1.6.0",
)

go_repository(
    name = "com_github_golang_snappy",
    tag = "v0.0.1",
    importpath = "github.com/golang/snappy",
)

go_repository(
    name = "in_gopkg_urfave_cli_v1",
    importpath = "gopkg.in/urfave/cli.v1",
    tag = "v1.20.0",
)

go_repository(
    name = "com_github_rjeczalik_notify",
    importpath = "github.com/rjeczalik/notify",
    tag = "v0.9.2",
)

go_repository(
    name = "com_github_edsrzf_mmap_go",
    importpath = "github.com/edsrzf/mmap-go",
    tag = "v1.0.0",
)

go_repository(
    name = "com_github_pkg_errors",
    importpath = "github.com/pkg/errors",
    tag = "v0.8.1",
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
    commit = "afaefda3ea643e9292b6f4596403ed5c742561b4",
    importpath = "github.com/phoreproject/bls",
)
