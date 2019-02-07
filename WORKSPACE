load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "bazel_skylib",
    url = "https://github.com/bazelbuild/bazel-skylib/archive/0.6.0.tar.gz",
    sha256 = "eb5c57e4c12e68c0c20bc774bfbc60a568e800d025557bc4ea022c6479acc867",
    strip_prefix = "bazel-skylib-0.6.0",
)

http_archive(
    name = "io_bazel_rules_go",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.17.0/rules_go-0.17.0.tar.gz",
    sha256 = "492c3ac68ed9dcf527a07e6a1b2dcbf199c6bf8b35517951467ac32e421c06c1",
)

http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.16.0/bazel-gazelle-0.16.0.tar.gz"],
    sha256 = "7949fc6cc17b5b191103e97481cf8889217263acf52e00b560683413af204fcb",
)

http_archive(
    name = "com_github_atlassian_bazel_tools",
    strip_prefix = "bazel-tools-6fef37f33dfa0189be9df4d3d60e6291bfe71177",
    urls = ["https://github.com/atlassian/bazel-tools/archive/6fef37f33dfa0189be9df4d3d60e6291bfe71177.tar.gz"],
    sha256 = "e7d0c0e2963a7f9cb2c377e241502119dae24909708adef1918e8dcb70ae9e8c",
)

http_archive(
    name = "io_bazel_rules_docker",
    url = "https://github.com/bazelbuild/rules_docker/archive/v0.7.0.tar.gz",
    strip_prefix = "rules_docker-0.7.0",
    sha256 = "aed1c249d4ec8f703edddf35cbe9dfaca0b5f5ea6e4cd9e83e99f3b0d1136c3d",
)

http_archive(
    name = "build_bazel_rules_nodejs",
    urls = ["https://github.com/bazelbuild/rules_nodejs/releases/download/0.18.1/rules_nodejs-0.18.1.tar.gz"],
    sha256 = "04f91b18b1ea5ef589ecd5a820efc13581284f9b1062d9fcf6f8745794434a63",
)

http_archive(
    name = "io_bazel_rules_k8s",
    url = "https://github.com/bazelbuild/rules_k8s/archive/2206972072d64e5d2d966d81cc6c5fb77fd58dcb.tar.gz",
    strip_prefix = "rules_k8s-2206972072d64e5d2d966d81cc6c5fb77fd58dcb",
    sha256 = "828fb1ac4c44280be95306b885a326e40110eeba50bffa05e72ddd3b5cdc5d33",
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
    name = "io_kubernetes_build",
    url = "https://github.com/kubernetes/repo-infra/archive/4ce715fbe67d8fbed05ec2bb47a148e754100a4b.tar.gz",
    strip_prefix = "repo-infra-4ce715fbe67d8fbed05ec2bb47a148e754100a4b",
    sha256 = "b4e7819861f2ec89b7309bd0c44fb3348c3a4a8ee494ec7668edb3960ff11814",
)

go_repository(
    name = "com_github_golang_mock",
    commit = "c20582278a829e4b3259747a3ce0eceb1763ee13",
    importpath = "github.com/golang/mock",
)

# External dependencies

go_repository(
    name = "com_github_ethereum_go_ethereum",
    importpath = "github.com/ethereum/go-ethereum",
    # Note: go-ethereum is not bazel-friendly with regards to cgo. We have a
    # a fork that has resolved these issues by disabling HID/USB support and
    # some manual fixes for c imports in the crypto package. This is forked
    # branch should be updated from time to time with the latest go-ethereum
    # code.
    remote = "https://github.com/prysmaticlabs/bazel-go-ethereum",
    vcs = "git",
    commit = "ec3d1d97a481e6cba542751a6defae7c76e322f1",
)

go_repository(
    name = "com_github_urfave_cli",
    tag = "v1.20.0",
    importpath = "github.com/urfave/cli",
)

go_repository(
    name = "com_github_go_yaml_yaml",
    tag = "v2.2.2",
    importpath = "github.com/go-yaml/yaml",
)

go_repository(
    name = "com_github_x_cray_logrus_prefixed_formatter",
    tag = "v0.5.2",
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
    tag = "v6.0.29",
    importpath = "github.com/libp2p/go-libp2p",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_peer",
    tag = "v2.4.0",
    importpath = "github.com/libp2p/go-libp2p-peer",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_crypto",
    build_file_proto_mode = "disable_global",
    tag = "v2.0.1",
    importpath = "github.com/libp2p/go-libp2p-crypto",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr",
    commit = "312b9db3552cf2045efb3ab5d10104c3ec8ff79d",
    importpath = "github.com/multiformats/go-multiaddr",
)

go_repository(
    name = "com_github_ipfs_go_log",
    tag = "v1.5.7",
    importpath = "github.com/ipfs/go-log",
)

go_repository(
    name = "com_github_multiformats_go_multihash",
    tag = "v1.0.8",
    importpath = "github.com/multiformats/go-multihash",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_swarm",
    tag = "v3.0.22",
    importpath = "github.com/libp2p/go-libp2p-swarm",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_host",
    tag = "v3.0.15",
    importpath = "github.com/libp2p/go-libp2p-host",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_peerstore",
    tag = "v2.0.6",
    importpath = "github.com/libp2p/go-libp2p-peerstore",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_circuit",
    tag = "v2.3.2",
    importpath = "github.com/libp2p/go-libp2p-circuit",
)

go_repository(
    name = "com_github_coreos_go_semver",
    tag = "v0.2.0",
    importpath = "github.com/coreos/go-semver",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_interface_connmgr",
    tag = "v0.0.21",
    importpath = "github.com/libp2p/go-libp2p-interface-connmgr",
)

go_repository(
    name = "com_github_libp2p_go_conn_security_multistream",
    tag = "v0.1.15",
    importpath = "github.com/libp2p/go-conn-security-multistream",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_metrics",
    tag = "v2.1.7",
    importpath = "github.com/libp2p/go-libp2p-metrics",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_net",
    tag = "v3.0.15",
    importpath = "github.com/libp2p/go-libp2p-net",
)

go_repository(
    name = "com_github_whyrusleeping_mafmt",
    tag = "v1.2.8",
    importpath = "github.com/whyrusleeping/mafmt",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_net",
    commit = "c75d1cac17a0d84dbf8b2c53c61f0ebf0575183a",
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
    commit = "d9f6b97f8db22dd1e090fd0bbbe98f09cc7dd0a8",
    importpath = "github.com/gxed/hashland",
)

go_repository(
    name = "com_github_mattn_go_colorable",
    tag = "v0.1.0",
    importpath = "github.com/mattn/go-colorable",
)

go_repository(
    name = "com_github_whyrusleeping_mdns",
    commit = "ef14215e6b30606f4ce84174ed7a644a05cb1af3",
    importpath = "github.com/whyrusleeping/mdns",
)

go_repository(
    name = "com_github_btcsuite_btcd",
    commit = "ed77733ec07dfc8a513741138419b8d9d3de9d2d",
    importpath = "github.com/btcsuite/btcd",
)

go_repository(
    name = "com_github_minio_sha256_simd",
    commit = "2d45a736cd16732fe6a57563cc20d8b035193e58",
    importpath = "github.com/minio/sha256-simd",
)

go_repository(
    name = "com_github_mr_tron_base58",
    tag = "v1.1.0",
    importpath = "github.com/mr-tron/base58",
)

go_repository(
    name = "com_github_whyrusleeping_go_smux_yamux",
    tag = "v2.0.8",
    importpath = "github.com/whyrusleeping/go-smux-yamux",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_secio",
    build_file_proto_mode = "disable_global",
    tag = "v2.0.17",
    importpath = "github.com/libp2p/go-libp2p-secio",
)

go_repository(
    name = "com_github_libp2p_go_tcp_transport",
    tag = "v2.0.16",
    importpath = "github.com/libp2p/go-tcp-transport",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_protocol",
    tag = "v1.0.0",
    importpath = "github.com/libp2p/go-libp2p-protocol",
)

go_repository(
    name = "com_github_jbenet_goprocess",
    commit = "b497e2f366b8624394fb2e89c10ab607bebdde0b",
    importpath = "github.com/jbenet/goprocess",
)

go_repository(
    name = "com_github_multiformats_go_multistream",
    commit = "0c61f185f3d6e16bcda416874e7a0fca4696e7e0",
    importpath = "github.com/multiformats/go-multistream",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_loggables",
    tag = "v1.1.24",
    importpath = "github.com/libp2p/go-libp2p-loggables",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_nat",
    tag = "v0.8.8",
    importpath = "github.com/libp2p/go-libp2p-nat",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_dns",
    tag = "v0.2.5",
    importpath = "github.com/multiformats/go-multiaddr-dns",
)

go_repository(
    name = "com_github_fd_go_nat",
    tag = "v1.0.0",
    importpath = "github.com/fd/go-nat",
)

go_repository(
    name = "com_github_whyrusleeping_go_logging",
    commit = "0457bb6b88fc1973573aaf6b5145d8d3ae972390",
    importpath = "github.com/whyrusleeping/go-logging",
)

go_repository(
    name = "com_github_mattn_go_isatty",
    tag = "v0.0.4",
    importpath = "github.com/mattn/go-isatty",
)

go_repository(
    name = "com_github_libp2p_go_stream_muxer",
    tag = "v3.0.1",
    importpath = "github.com/libp2p/go-stream-muxer",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_transport_upgrader",
    tag = "v0.1.16",
    importpath = "github.com/libp2p/go-libp2p-transport-upgrader",
)

go_repository(
    name = "com_github_libp2p_go_testutil",
    tag = "v1.2.10",
    importpath = "github.com/libp2p/go-testutil",
)

go_repository(
    name = "com_github_whyrusleeping_go_smux_multistream",
    tag = "v2.0.2",
    importpath = "github.com/whyrusleeping/go-smux-multistream",
)

go_repository(
    name = "com_github_libp2p_go_maddr_filter",
    tag = "v1.1.10",
    importpath = "github.com/libp2p/go-maddr-filter",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_transport",
    tag = "v3.0.15",
    importpath = "github.com/libp2p/go-libp2p-transport",
)

go_repository(
    name = "com_github_libp2p_go_addr_util",
    tag = "v2.0.7",
    importpath = "github.com/libp2p/go-addr-util",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_interface_pnet",
    tag = "v3.0.0",
    importpath = "github.com/libp2p/go-libp2p-interface-pnet",
)

go_repository(
    name = "com_github_libp2p_go_conn_security",
    tag = "v0.1.15",
    importpath = "github.com/libp2p/go-conn-security",
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
    tag = "v1.0.2",
    importpath = "github.com/opentracing/opentracing-go",
)

go_repository(
    name = "com_github_libp2p_go_reuseport",
    tag = "v0.2.0",
    importpath = "github.com/libp2p/go-reuseport",
)

go_repository(
    name = "com_github_huin_goupnp",
    tag = "v1.0.0",
    importpath = "github.com/huin/goupnp",
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
    tag = "v1.2.0",
    importpath = "github.com/satori/go.uuid",
)

go_repository(
    name = "com_github_sirupsen_logrus",
    tag = "v1.3.0",
    importpath = "github.com/sirupsen/logrus",
)

go_repository(
    name = "org_golang_x_sys",
    commit = "41f3e6584952bb034a481797859f6ab34b6803bd",
    importpath = "golang.org/x/sys",
)

go_repository(
    name = "com_github_whyrusleeping_yamux",
    tag = "v1.1.5",
    importpath = "github.com/whyrusleeping/yamux",
)

go_repository(
    name = "com_github_libp2p_go_flow_metrics",
    tag = "v0.2.0",
    importpath = "github.com/libp2p/go-flow-metrics",
)

go_repository(
    name = "com_github_libp2p_go_msgio",
    tag = "v0.0.6",
    importpath = "github.com/libp2p/go-msgio",
)

go_repository(
    name = "com_github_jackpal_gateway",
    tag = "v1.0.5",
    importpath = "github.com/jackpal/gateway",
)

go_repository(
    name = "com_github_whyrusleeping_multiaddr_filter",
    commit = "e903e4adabd70b78bc9293b6ee4f359afb3f9f59",
    importpath = "github.com/whyrusleeping/multiaddr-filter",
)

go_repository(
    name = "com_github_libp2p_go_ws_transport",
    tag = "v2.0.15",
    importpath = "github.com/libp2p/go-ws-transport",
)

go_repository(
    name = "org_golang_x_crypto",
    commit = "b8fe1690c61389d7d2a8074a507d1d40c5d30448",
    importpath = "golang.org/x/crypto",
)

go_repository(
    name = "com_github_jackpal_go_nat_pmp",
    tag = "v1.0.1",
    importpath = "github.com/jackpal/go-nat-pmp",
)

go_repository(
    name = "com_github_libp2p_go_reuseport_transport",
    tag = "v0.2.0",
    importpath = "github.com/libp2p/go-reuseport-transport",
)

go_repository(
    name = "com_github_libp2p_go_sockaddr",
    tag = "v1.0.3",
    importpath = "github.com/libp2p/go-sockaddr",
)

go_repository(
    name = "com_github_whyrusleeping_go_notifier",
    commit = "097c5d47330ff6a823f67e3515faa13566a62c6f",
    importpath = "github.com/whyrusleeping/go-notifier",
)

go_repository(
    name = "com_github_gorilla_websocket",
    tag = "v1.4.0",
    importpath = "github.com/gorilla/websocket",
)

go_repository(
    name = "com_github_whyrusleeping_go_smux_multiplex",
    tag = "v3.0.16",
    importpath = "github.com/whyrusleeping/go-smux-multiplex",
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
    commit = "2f17a3356c6616cbfc4ae4c38147dc062a68fb0e",
    importpath = "github.com/syndtr/goleveldb",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_blankhost",
    tag = "v0.3.15",
    importpath = "github.com/libp2p/go-libp2p-blankhost",
)

go_repository(
    name = "com_github_steakknife_hamming",
    tag = "0.2.5",
    importpath = "github.com/steakknife/hamming",
)

go_repository(
    name = "io_opencensus_go",
    tag = "v0.19.0",
    importpath = "go.opencensus.io",
)

go_repository(
    name = "org_golang_google_api",
    tag = "v0.1.0",
    importpath = "google.golang.org/api",
)

go_repository(
    name = "org_golang_x_sync",
    commit = "37e7f081c4d4c64e13b10787722085407fe5d15f",
    importpath = "golang.org/x/sync",
)

go_repository(
    name = "com_github_golang_lint",
    commit = "8f45f776aaf18cebc8d65861cc70c33c60471952",
    importpath = "github.com/golang/lint",
)

go_repository(
    name = "org_golang_x_lint",
    commit = "8f45f776aaf18cebc8d65861cc70c33c60471952",
    importpath = "golang.org/x/lint",
)

go_repository(
    name = "com_github_aristanetworks_goarista",
    commit = "2166578f34481200ec14ddd6b9c4115db36c7b62",
    importpath = "github.com/aristanetworks/goarista",
)

go_repository(
    name = "com_github_prometheus_client_golang",
    tag = "v0.9.2",
    importpath = "github.com/prometheus/client_golang",
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
    commit = "488faf799f863e27e50c516468f76ae8f1da20a5",
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
    tag = "v1.0.1",
    importpath = "github.com/matttproud/golang_protobuf_extensions",
)

go_repository(
    name = "com_github_boltdb_bolt",
    tag = "v1.3.1",
    importpath = "github.com/boltdb/bolt",
)

go_repository(
    name = "com_github_pborman_uuid",
    tag = "v1.2.0",
    importpath = "github.com/pborman/uuid",
)

go_repository(
    name = "com_github_libp2p_go_buffer_pool",
    tag = "v0.1.3",
    importpath = "github.com/libp2p/go-buffer-pool",
)

go_repository(
    name = "com_github_libp2p_go_mplex",
    tag = "v0.2.30",
    importpath = "github.com/libp2p/go-mplex",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_pubsub",
    build_file_proto_mode = "disable_global",
    tag = "v0.11.10",
    importpath = "github.com/libp2p/go-libp2p-pubsub",
)

go_repository(
    name = "com_github_ipfs_go_ipfs_util",
    tag = "v1.2.8",
    importpath = "github.com/ipfs/go-ipfs-util",
)

go_repository(
    name = "com_github_google_uuid",
    tag = "v1.1.0",
    importpath = "github.com/google/uuid",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_kad_dht",
    build_file_proto_mode = "disable_global",
    tag = "v4.4.12",
    importpath = "github.com/libp2p/go-libp2p-kad-dht",
)

go_repository(
    name = "com_github_ipfs_go_datastore",
    tag = "v3.2.0",
    importpath = "github.com/ipfs/go-datastore",
)

go_repository(
    name = "com_github_whyrusleeping_base32",
    commit = "c30ac30633ccdabefe87eb12465113f06f1bab75",
    importpath = "github.com/whyrusleeping/base32",
)

go_repository(
    name = "com_github_ipfs_go_cid",
    tag = "v0.9.0",
    importpath = "github.com/ipfs/go-cid",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_record",
    build_file_proto_mode = "disable_global",
    tag = "v4.1.7",
    importpath = "github.com/libp2p/go-libp2p-record",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_routing",
    tag = "v2.7.1",
    importpath = "github.com/libp2p/go-libp2p-routing",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_kbucket",
    tag = "v2.2.12",
    importpath = "github.com/libp2p/go-libp2p-kbucket",
)

go_repository(
    name = "com_github_jbenet_go_context",
    commit = "d14ea06fba99483203c19d92cfcd13ebe73135f4",
    importpath = "github.com/jbenet/go-context",
)

go_repository(
    name = "com_github_ipfs_go_todocounter",
    tag = "v1.0.1",
    importpath = "github.com/ipfs/go-todocounter",
)

go_repository(
    name = "com_github_whyrusleeping_go_keyspace",
    commit = "5b898ac5add1da7178a4a98e69cb7b9205c085ee",
    importpath = "github.com/whyrusleeping/go-keyspace",
)

go_repository(
    name = "com_github_multiformats_go_multibase",
    tag = "v0.3.0",
    importpath = "github.com/multiformats/go-multibase",
)

go_repository(
    name = "com_github_hashicorp_golang_lru",
    tag = "v0.5.0",
    importpath = "github.com/hashicorp/golang-lru",
)

go_repository(
    name = "com_github_ipfs_go_ipfs_addr",
    tag = "v0.1.25",
    importpath = "github.com/ipfs/go-ipfs-addr",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_discovery",
    commit = "cc4105e21706452e5b0f7e05390f987017188d31",
    importpath = "github.com/libp2p/go-libp2p-discovery",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_autonat",
    commit = "93b1787f76de807b9ab3a7c7edd45cf906139bdb",
    importpath = "github.com/libp2p/go-libp2p-autonat",
)

go_repository(
    name = "com_github_konsorten_go_windows_terminal_sequences",
    tag = "v1.0.1",
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
    tag = "v2.2.2",
    importpath = "gopkg.in/yaml.v2",
)

go_repository(
    name = "com_github_spf13_pflag",
    tag = "v1.0.3",
    importpath = "github.com/spf13/pflag",
)

go_repository(
    name = "com_github_spf13_cobra",
    tag = "v0.0.3",
    importpath = "github.com/spf13/cobra",
)

go_repository(
    name = "com_github_aws_aws_sdk_go",
    tag = "v1.16.9",
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
    commit = "99b60b757ec124ebb7d6b7e97f153b19c10ce163",
    importpath = "golang.org/x/oauth2",
)

go_repository(
    name = "com_github_hashicorp_go_multierror",
    tag = "v1.0.0",
    importpath = "github.com/hashicorp/go-multierror",
)

go_repository(
    name = "com_github_hashicorp_errwrap",
    tag = "v1.0.0",
    importpath = "github.com/hashicorp/errwrap",
)

go_repository(
    name = "com_google_cloud_go",
    tag = "v0.36.0",
    importpath = "cloud.google.com/go",
)

go_repository(
    name = "com_github_inconshreveable_mousetrap",
    tag = "v1.0.0",
    importpath = "github.com/inconshreveable/mousetrap",
)

go_repository(
    name = "com_github_deckarep_golang_set",
    tag = "v1.7.1",
    importpath = "github.com/deckarep/golang-set",
)

go_repository(
    name = "com_github_go_stack_stack",
    tag = "v1.8.0",
    importpath = "github.com/go-stack/stack",
)

go_repository(
    name = "com_github_rs_cors",
    tag = "v1.6.0",
    importpath = "github.com/rs/cors",
)

go_repository(
    name = "com_github_golang_snappy",
    commit = "2e65f85255dbc3072edf28d6b5b8efc472979f5a",
    importpath = "github.com/golang/snappy",
)

go_repository(
    name = "in_gopkg_urfave_cli_v1",
    tag = "v1.20.0",
    importpath = "gopkg.in/urfave/cli.v1",
)

go_repository(
    name = "com_github_rjeczalik_notify",
    tag = "v0.9.2",
    importpath = "github.com/rjeczalik/notify",
)

go_repository(
    name = "com_github_edsrzf_mmap_go",
    tag = "v1.0.0",
    importpath = "github.com/edsrzf/mmap-go",
)

go_repository(
    name = "com_github_pkg_errors",
    tag = "v0.8.1",
    importpath = "github.com/pkg/errors",
)

go_repository(
    name = "in_gopkg_natefinch_npipe_v2",
    commit = "c1b8fa8bdccecb0b8db834ee0b92fdbcfa606dd6",
    importpath = "gopkg.in/natefinch/npipe.v2",
)
