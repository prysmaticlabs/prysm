load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "bazel_skylib",
    url = "https://github.com/bazelbuild/bazel-skylib/archive/0.7.0.tar.gz",
    sha256 = "2c62d8cd4ab1e65c08647eb4afe38f51591f43f7f0885e7769832fa137633dcb",
    strip_prefix = "bazel-skylib-0.7.0",
)

http_archive(
    name = "io_bazel_rules_go",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.18.1/rules_go-0.18.1.tar.gz",
    sha256 = "77dfd303492f2634de7a660445ee2d3de2960cbd52f97d8c0dffa9362d3ddef9",
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "3c681998538231a2d24d0c07ed5a7658cb72bfb5fd4bf9911157c0e9ac6a2687",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.17.0/bazel-gazelle-0.17.0.tar.gz"],
)

http_archive(
    name = "com_github_atlassian_bazel_tools",
    sha256 = "2842e7e4315f43df4223b7af40ae7d3547cce967595505bc335dacad5d69a39f",
    strip_prefix = "bazel-tools-02472d814d1867de2ad75801760732c9e595c9d9",
    urls = ["https://github.com/atlassian/bazel-tools/archive/02472d814d1867de2ad75801760732c9e595c9d9.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_docker",
    url = "https://github.com/bazelbuild/rules_docker/archive/v0.7.0.tar.gz",
    strip_prefix = "rules_docker-0.7.0",
    sha256 = "aed1c249d4ec8f703edddf35cbe9dfaca0b5f5ea6e4cd9e83e99f3b0d1136c3d",
)

http_archive(
    name = "build_bazel_rules_nodejs",
    sha256 = "fb87ed5965cef93188af9a7287511639403f4b0da418961ce6defb9dcf658f51",
    urls = ["https://github.com/bazelbuild/rules_nodejs/releases/download/0.27.7/rules_nodejs-0.27.7.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_k8s",
    sha256 = "c18619d129c5a22798e8a8149c29a55f11f3b1a79872f5ad5282624e5c845f4a",
    strip_prefix = "rules_k8s-54e9742bd2b75facdcf30fe3a1307ca0fad17385",
    url = "https://github.com/bazelbuild/rules_k8s/archive/54e9742bd2b75facdcf30fe3a1307ca0fad17385.tar.gz",
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
    sha256 = "4985e8d2da96c9ec293897cc19573390c392e068e733b3d85972b43aac2f1309",
    strip_prefix = "repo-infra-52d76ba3344e755f5a9ab595b94b21df49448ffb",
    url = "https://github.com/kubernetes/repo-infra/archive/52d76ba3344e755f5a9ab595b94b21df49448ffb.tar.gz",
)

go_repository(
    name = "com_github_golang_mock",
    commit = "c20582278a829e4b3259747a3ce0eceb1763ee13",
    importpath = "github.com/golang/mock",
)

# External dependencies

go_repository(
    name = "com_github_ethereum_go_ethereum",
    commit = "86c88456c94c6a58ac48f0e192f520dc9aeb14a4",
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
    commit = "51d6538a90f86fe93ac480b35f37b2be17fef232",
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
    commit = "2308ba3c0036ddf8ac390c893b71d946c57a7328",  # v0.0.2
    importpath = "github.com/libp2p/go-libp2p",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_peer",
    commit = "6ce6ea8d949b826bc1e663ba5bc321c3cc9c2cce",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-peer",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_crypto",
    build_file_proto_mode = "disable_global",
    commit = "b0ed0e663e8b6832bad3f4502b2f6551ff2686cd",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-crypto",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr",
    commit = "ce21123d5172669bbf37a166078bc5f9d345ec2f",  # v0.0.2
    importpath = "github.com/multiformats/go-multiaddr",
)

go_repository(
    name = "com_github_ipfs_go_log",
    commit = "91b837264c0f35dd4e2be341d711316b91d3573d",  # v0.0.1
    importpath = "github.com/ipfs/go-log",
)

go_repository(
    name = "com_github_multiformats_go_multihash",
    commit = "6b439b7c6e3c44c112171540500be697ba235235",  # v0.0.1
    importpath = "github.com/multiformats/go-multihash",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_swarm",
    commit = "60a7ff4a5eda1873184a80689a0c7a3b36e76452",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-swarm",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_host",
    commit = "7e2de4bb2b77a5b155b9993309b34e0e8b1890b0",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-host",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_peerstore",
    commit = "d2a21b6d4a88bd06c1cd5700b38df4529e556b28",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-peerstore",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_circuit",
    commit = "0e57e52a3e3dc6d9e446117beded675d021fd3a4",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-circuit",
)

go_repository(
    name = "com_github_coreos_go_semver",
    commit = "e214231b295a8ea9479f11b70b35d5acf3556d9b",
    importpath = "github.com/coreos/go-semver",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_interface_connmgr",
    commit = "8bddc198c80e6242d016361b3c5f02fb920e9faa",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-interface-connmgr",
)

go_repository(
    name = "com_github_libp2p_go_conn_security_multistream",
    commit = "d4c3bf7e5688390e4106d3ec73aafd7d0856143b",  # v0.0.1
    importpath = "github.com/libp2p/go-conn-security-multistream",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_metrics",
    commit = "f7021db7729eb519a86ec6a90b5b476e6072f6f7",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-metrics",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_net",
    commit = "3835e54052bfbea8a8cc81758c4981aeda28de51",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-net",
)

go_repository(
    name = "com_github_whyrusleeping_mafmt",
    commit = "158fbc0fcc20b3d8a251df520466e16f66eb3651",  # v1.2.8
    importpath = "github.com/whyrusleeping/mafmt",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_net",
    commit = "bd61b0499a3cfc893a8eb109c5669342b1671881",  # v0.0.1
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
    commit = "a72cc0875a1e95edd309d3134bc7c11bf2d7360b",
    importpath = "github.com/gxed/hashland",
)

go_repository(
    name = "com_github_mattn_go_colorable",
    commit = "3a70a971f94a22f2fa562ffcc7a0eb45f5daf045",
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
    commit = "89529c6904fcd077434931b4eac8b4b2f0991baf",  # v1.1.0
    importpath = "github.com/mr-tron/base58",
)

go_repository(
    name = "com_github_whyrusleeping_go_smux_yamux",
    commit = "7402600b69ab8af34116420d9b9cd4e1b8968901",  # v2.0.9
    importpath = "github.com/whyrusleeping/go-smux-yamux",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_secio",
    build_file_proto_mode = "disable_global",
    commit = "c3eee8a92db23bd65c434512810c60bec85e7ac9",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-secio",
)

go_repository(
    name = "com_github_libp2p_go_tcp_transport",
    commit = "69079ce9b2140ebe9e750c70ac30b94fef08f627",  # v0.0.1
    importpath = "github.com/libp2p/go-tcp-transport",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_protocol",
    commit = "c170ad966a647835bda0cbe4dc710cf9f94949b6",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-protocol",
)

go_repository(
    name = "com_github_jbenet_goprocess",
    commit = "b497e2f366b8624394fb2e89c10ab607bebdde0b",
    importpath = "github.com/jbenet/goprocess",
)

go_repository(
    name = "com_github_multiformats_go_multistream",
    commit = "f3f44044ac2444cd3a017c0b269f8da65b0012f1",  # v0.0.1
    importpath = "github.com/multiformats/go-multistream",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_loggables",
    commit = "332c68ea465a3353571377635eb0fc257a0c38c4",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-loggables",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_nat",
    commit = "3efa307896b71a07705d8d0cf7f19a4595c62d4f",  # v0.0.2
    importpath = "github.com/libp2p/go-libp2p-nat",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_dns",
    commit = "e7c544d7a325c57bdbd7e9ba9c035a6701c5c7d2",  # v0.0.2
    importpath = "github.com/multiformats/go-multiaddr-dns",
)

go_repository(
    name = "com_github_fd_go_nat",
    commit = "d6a8fd67f078b7cc7a16d623ee52e859b2daccd5",  # v1.0.0
    importpath = "github.com/fd/go-nat",
)

go_repository(
    name = "com_github_whyrusleeping_go_logging",
    commit = "0457bb6b88fc1973573aaf6b5145d8d3ae972390",
    importpath = "github.com/whyrusleeping/go-logging",
)

go_repository(
    name = "com_github_mattn_go_isatty",
    commit = "c2a7a6ca930a4cd0bc33a3f298eb71960732a3a7",  # v0.0.7
    importpath = "github.com/mattn/go-isatty",
)

go_repository(
    name = "com_github_libp2p_go_stream_muxer",
    commit = "96b8d4dc6b2577782a7c6c33776c115e3d2eacf6",  # v0.0.1
    importpath = "github.com/libp2p/go-stream-muxer",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_transport_upgrader",
    commit = "b4ab689796cdae99ad6641abcedcd7c23258de0c",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-transport-upgrader",
)

go_repository(
    name = "com_github_libp2p_go_testutil",
    commit = "425d0bf3c5d09720ab62cb2b79107d83651f271b",  # v0.0.1
    importpath = "github.com/libp2p/go-testutil",
)

go_repository(
    name = "com_github_whyrusleeping_go_smux_multistream",
    commit = "90c0868b4933b5cb1bbcf68a3ca11c6767ccc048",  # v2.0.2
    importpath = "github.com/whyrusleeping/go-smux-multistream",
)

go_repository(
    name = "com_github_libp2p_go_maddr_filter",
    commit = "1258bb7e3e3a37bb9a93aaf77ef88f8405b39bce",  # v0.0.1
    importpath = "github.com/libp2p/go-maddr-filter",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_transport",
    commit = "760cba29c65701ce7b688f238f6c93cc1d899dde",  # v0.0.4
    importpath = "github.com/libp2p/go-libp2p-transport",
)

go_repository(
    name = "com_github_libp2p_go_addr_util",
    commit = "4cd36c0f325f9e38f1e31ff7a10b9d94d53a11cf",  # v0.0.1
    importpath = "github.com/libp2p/go-addr-util",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_interface_pnet",
    commit = "b02026130a4daafb8001e039e116121cb92ae1a9",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-interface-pnet",
)

go_repository(
    name = "com_github_libp2p_go_conn_security",
    commit = "80b6115ae32f104b2ed14d16a8551248f4599338",  # v0.0.1
    importpath = "github.com/libp2p/go-conn-security",
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
    commit = "656e61dfadd241c7cbdd22a023fa81ecb6860ea8",
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
    name = "com_github_satori_go_uuid",
    commit = "fe3bd84a0754cc4deb35c1398cd0ebd23f1d8a41",  # v1.2.0
    importpath = "github.com/satori/go.uuid",
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
    name = "com_github_whyrusleeping_yamux",
    commit = "5364a42fe4b5efa5967c11c8f9b0f049cac0c4a9",  # v1.1.5
    importpath = "github.com/whyrusleeping/yamux",
)

go_repository(
    name = "com_github_libp2p_go_flow_metrics",
    commit = "1f5b3acc846b2c8ce4c4e713296af74f5c24df55",  # v0.0.1
    importpath = "github.com/libp2p/go-flow-metrics",
)

go_repository(
    name = "com_github_libp2p_go_msgio",
    commit = "2efd92d62d528c56c6711346e82df0c74a5742a8",  # v0.0.1
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
    commit = "2114e839891f0a824d9a36919165fed197522d87",  # v0.0.1
    importpath = "github.com/libp2p/go-ws-transport",
)

go_repository(
    name = "org_golang_x_crypto",
    commit = "8dd112bcdc25174059e45e07517d9fc663123347",
    importpath = "golang.org/x/crypto",
)

go_repository(
    name = "com_github_jackpal_go_nat_pmp",
    commit = "b977c5efcadd604e306a97fdf06ff544f785df56",  # v1.0.1
    importpath = "github.com/jackpal/go-nat-pmp",
)

go_repository(
    name = "com_github_libp2p_go_reuseport_transport",
    commit = "a38ecc623029c60063f999d16a7b62c1053adfc3",  # v0.0.1
    importpath = "github.com/libp2p/go-reuseport-transport",
)

go_repository(
    name = "com_github_libp2p_go_sockaddr",
    commit = "864d6651a43fce610e83b4c32dc80f5f85b6d28b",  # v0.0.1
    importpath = "github.com/libp2p/go-sockaddr",
)

go_repository(
    name = "com_github_whyrusleeping_go_notifier",
    commit = "097c5d47330ff6a823f67e3515faa13566a62c6f",
    importpath = "github.com/whyrusleeping/go-notifier",
)

go_repository(
    name = "com_github_gorilla_websocket",
    commit = "7c8e298727d149d7c329b4dec7e94e1932ac5c11",
    importpath = "github.com/gorilla/websocket",
)

go_repository(
    name = "com_github_whyrusleeping_go_smux_multiplex",
    commit = "69b7a74ecfb84be57bdcf4b264a7042d80569287",  # v3.0.16
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
    commit = "4217c9f31f5816db02addc94e56061da77f288d8",
    importpath = "github.com/syndtr/goleveldb",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_blankhost",
    commit = "177484ab3c284e75a9fa25b3114b2c5a4395deaa",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-blankhost",
)

go_repository(
    name = "com_github_steakknife_hamming",
    commit = "c99c65617cd3d686aea8365fe563d6542f01d940",
    importpath = "github.com/steakknife/hamming",
)

go_repository(
    name = "io_opencensus_go",
    commit = "986bcac05415cee0c283e43f5097b0448871d74c",  # v0.19.1
    importpath = "go.opencensus.io",
)

go_repository(
    name = "org_golang_google_api",
    commit = "e742f5a8defa1f9f5d723dfa04c962e680dc33f0",  # v0.2.0
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
    commit = "2daed26f633ce23fab5efc02fb3f2e803b3526c3",
    importpath = "github.com/prometheus/client_golang",
)

go_repository(
    name = "com_github_prometheus_client_model",
    commit = "fd36f4220a901265f90734c3183c5f0c91daa0b8",
    importpath = "github.com/prometheus/client_model",
)

go_repository(
    name = "com_github_prometheus_common",
    commit = "7a3416fd1f41341905f138746c51a5092e7ddf7a",
    importpath = "github.com/prometheus/common",
)

go_repository(
    name = "com_github_prometheus_procfs",
    commit = "bbced9601137e764853b2fad7ec3e2dc4c504e02",
    importpath = "github.com/prometheus/procfs",
)

go_repository(
    name = "com_github_prometheus_prometheus",
    commit = "ed970a17fe1585310bd79093b788602df9fc73c3",  # v2.7.2
    importpath = "github.com/prometheus/prometheus",
)

go_repository(
    name = "com_github_beorn7_perks",
    commit = "3a771d992973f24aa725d07868b467d1ddfceafb",
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
    commit = "eecb57f6a721b4a66062ebeb61d1da7ce1c38fbf",  # v0.0.1
    importpath = "github.com/libp2p/go-buffer-pool",
)

go_repository(
    name = "com_github_libp2p_go_mplex",
    commit = "b16006283f6036f9a461f78692c397b134a75393",  # v0.0.1
    importpath = "github.com/libp2p/go-mplex",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_pubsub",
    build_file_proto_mode = "disable_global",
    commit = "25cbf38777e869acb77e8079c3c76d6b430e66ad",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-pubsub",
)

go_repository(
    name = "com_github_ipfs_go_ipfs_util",
    commit = "a4bb5361e49427531f9a716ead2ce4bd9bdd7959",  # v0.0.1
    importpath = "github.com/ipfs/go-ipfs-util",
)

go_repository(
    name = "com_github_google_uuid",
    commit = "0cd6bf5da1e1c83f8b45653022c74f71af0538a4",
    importpath = "github.com/google/uuid",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_kad_dht",
    build_file_proto_mode = "disable_global",
    commit = "7bdf621a661616003dcf678d662667c5d476be26",  # v0.0.4
    importpath = "github.com/libp2p/go-libp2p-kad-dht",
)

go_repository(
    name = "com_github_ipfs_go_datastore",
    commit = "1b371980cb3fe9edba40c0f0584042e5cb4477a4",  # v0.0.1
    importpath = "github.com/ipfs/go-datastore",
)

go_repository(
    name = "com_github_whyrusleeping_base32",
    commit = "c30ac30633ccdabefe87eb12465113f06f1bab75",
    importpath = "github.com/whyrusleeping/base32",
)

go_repository(
    name = "com_github_ipfs_go_cid",
    commit = "e7e67e08cfba888a4297224402e12832bdc15ea0",  # v0.0.1
    importpath = "github.com/ipfs/go-cid",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_record",
    build_file_proto_mode = "disable_global",
    commit = "4c2093f40950fe401a612a725ef0765c0c767b60",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-record",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_routing",
    commit = "ab366780f22ac0d2625105a9f6d3cace1817dd30",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-routing",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_kbucket",
    commit = "fb3bb85b76cc6efea5566649702b3fe6ecbad05f",  # v0.1.0
    importpath = "github.com/libp2p/go-libp2p-kbucket",
)

go_repository(
    name = "com_github_jbenet_go_context",
    commit = "d14ea06fba99483203c19d92cfcd13ebe73135f4",
    importpath = "github.com/jbenet/go-context",
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
    commit = "d4ef632c52ff995d94eaed926529eede8faa3965",  # v0.0.1
    importpath = "github.com/libp2p/go-libp2p-discovery",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_autonat",
    commit = "16d66b2ad60fc6b6956b0c2bd332111a94db6dbb",  # v0.0.2
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
    commit = "ef82de70bb3f60c65fb8eebacbb2d122ef517385",  # v0.0.3
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
    commit = "f52f9bc132541d2aa914f42100c36d10b1ef7e0c",  # v0.37.0
    importpath = "cloud.google.com/go",
)

go_repository(
    name = "com_github_inconshreveable_mousetrap",
    commit = "76626ae9c91c4f2a10f34cad8ce83ea42c93bb75",  # v1.0.0
    importpath = "github.com/inconshreveable/mousetrap",
)

go_repository(
    name = "com_github_deckarep_golang_set",
    commit = "699df6a3acf6867538e50931511e9dc403da108a",
    importpath = "github.com/deckarep/golang-set",
)

go_repository(
    name = "com_github_go_stack_stack",
    commit = "2fee6af1a9795aafbe0253a0cfbdf668e1fb8a9a",
    importpath = "github.com/go-stack/stack",
)

go_repository(
    name = "com_github_rs_cors",
    commit = "76f58f330d76a55c5badc74f6212e8a15e742c77",
    importpath = "github.com/rs/cors",
)

go_repository(
    name = "com_github_golang_snappy",
    commit = "2a8bb927dd31d8daada140a5d09578521ce5c36a",
    importpath = "github.com/golang/snappy",
)

go_repository(
    name = "in_gopkg_urfave_cli_v1",
    commit = "cfb38830724cc34fedffe9a2a29fb54fa9169cd1",
    importpath = "gopkg.in/urfave/cli.v1",
)

go_repository(
    name = "com_github_rjeczalik_notify",
    commit = "629144ba06a1c6af28c1e42c228e3d42594ce081",
    importpath = "github.com/rjeczalik/notify",
)

go_repository(
    name = "com_github_edsrzf_mmap_go",
    commit = "904c4ced31cdffe19e971afa0b3d319ff06d9c72",
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
    commit = "afaefda3ea643e9292b6f4596403ed5c742561b4",
    importpath = "github.com/phoreproject/bls",
)

go_repository(
    name = "com_github_multiformats_go_base32",
    commit = "a9c2755c3d1672dbe6a7e4a5d182169fa30b6a8e",  # v0.0.3
    importpath = "github.com/multiformats/go-base32",
)

go_repository(
    name = "com_github_allegro_bigcache",
    commit = "84a0ff3f153cbd7e280a19029a864bb04b504e62",  # v1.2.0
    importpath = "github.com/allegro/bigcache",
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
