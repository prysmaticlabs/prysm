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
    commit = "693af58b4d51b8fcc7f9d89576da170765980581",
    importpath = "github.com/urfave/cli",
)

go_repository(
    name = "com_github_go_yaml_yaml",
    commit = "51d6538a90f86fe93ac480b35f37b2be17fef232",
    importpath = "github.com/go-yaml/yaml",
)

go_repository(
    name = "com_github_x_cray_logrus_prefixed_formatter",
    commit = "bb2702d423886830dee131692131d35648c382e2",
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
    commit = "c4577c770392f9b24753da6e0863da2466683427",
    importpath = "github.com/libp2p/go-libp2p",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_peer",
    commit = "6ce6ea8d949b826bc1e663ba5bc321c3cc9c2cce",
    importpath = "github.com/libp2p/go-libp2p-peer",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_crypto",
    build_file_proto_mode = "disable_global",
    commit = "b0ed0e663e8b6832bad3f4502b2f6551ff2686cd",
    importpath = "github.com/libp2p/go-libp2p-crypto",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr",
    commit = "ce21123d5172669bbf37a166078bc5f9d345ec2f",
    importpath = "github.com/multiformats/go-multiaddr",
)

go_repository(
    name = "com_github_ipfs_go_log",
    commit = "91b837264c0f35dd4e2be341d711316b91d3573d",
    importpath = "github.com/ipfs/go-log",
)

go_repository(
    name = "com_github_multiformats_go_multihash",
    commit = "1a04c485626b992b36afcaa599584fdb0770c397",
    importpath = "github.com/multiformats/go-multihash",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_swarm",
    commit = "22f358a2d09ac82ee9ff0774d06010913fa900ad",
    importpath = "github.com/libp2p/go-libp2p-swarm",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_host",
    commit = "2aa838333967d181d28f8612a1109cb58df3c77f",
    importpath = "github.com/libp2p/go-libp2p-host",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_peerstore",
    commit = "d2a21b6d4a88bd06c1cd5700b38df4529e556b28",
    importpath = "github.com/libp2p/go-libp2p-peerstore",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_circuit",
    commit = "ae9b94afecf55cc91e0afe91f7ce2fbf513244b2",
    importpath = "github.com/libp2p/go-libp2p-circuit",
)

go_repository(
    name = "com_github_coreos_go_semver",
    commit = "e214231b295a8ea9479f11b70b35d5acf3556d9b",
    importpath = "github.com/coreos/go-semver",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_interface_connmgr",
    commit = "6a96ea29ac65d4ee4cc52c25ac41c036bbb7992a",
    importpath = "github.com/libp2p/go-libp2p-interface-connmgr",
)

go_repository(
    name = "com_github_libp2p_go_conn_security_multistream",
    commit = "104c8a4422af6d10f488a14b64a43d7c92be0475",
    importpath = "github.com/libp2p/go-conn-security-multistream",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_metrics",
    commit = "a64ed6c7351276b3f21cbfa033ceba9bd096c3af",
    importpath = "github.com/libp2p/go-libp2p-metrics",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_net",
    commit = "e71fff5ba6e902c9ee793495d44bdf6d497a3c12",
    importpath = "github.com/libp2p/go-libp2p-net",
)

go_repository(
    name = "com_github_whyrusleeping_mafmt",
    commit = "faeb70738e904fdbef70bcf62ec8c38726c16d17",
    importpath = "github.com/whyrusleeping/mafmt",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_net",
    commit = "bd61b0499a3cfc893a8eb109c5669342b1671881",
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
    commit = "fe73eb13120270ef478822e38664f5e56dc39547",
    importpath = "github.com/mr-tron/base58",
)

go_repository(
    name = "com_github_whyrusleeping_go_smux_yamux",
    commit = "28bea8f315d17495c5cad8547b661245486ebeee",
    importpath = "github.com/whyrusleeping/go-smux-yamux",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_secio",
    build_file_proto_mode = "disable_global",
    commit = "aa2813e066f68d5ce565bb007a043a2f69ac6148",
    importpath = "github.com/libp2p/go-libp2p-secio",
)

go_repository(
    name = "com_github_libp2p_go_tcp_transport",
    commit = "80ec1b10b24c88ff1ac7d1e32966528747a16237",
    importpath = "github.com/libp2p/go-tcp-transport",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_protocol",
    commit = "89147e29c7ab3b76c6f8be5b6fcdf3839aec286f",
    importpath = "github.com/libp2p/go-libp2p-protocol",
)

go_repository(
    name = "com_github_jbenet_goprocess",
    commit = "b497e2f366b8624394fb2e89c10ab607bebdde0b",
    importpath = "github.com/jbenet/goprocess",
)

go_repository(
    name = "com_github_multiformats_go_multistream",
    commit = "c5a72a718aa128e5f0bb5d4bd5d9a518bceb15e5",
    importpath = "github.com/multiformats/go-multistream",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_loggables",
    commit = "0b7472f2b9b44c3b1af5d6f6121be821704a6dcd",
    importpath = "github.com/libp2p/go-libp2p-loggables",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_nat",
    commit = "669534f43c3f972de906203cfd8548bc7fda0729",
    importpath = "github.com/libp2p/go-libp2p-nat",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_dns",
    commit = "25bd1ef04b2908f182226bff33da5057f3ae306e",
    importpath = "github.com/multiformats/go-multiaddr-dns",
)

go_repository(
    name = "com_github_fd_go_nat",
    commit = "e3ba0d89e7d9f0a458bf08baae8db007eb7d242d",
    importpath = "github.com/fd/go-nat",
)

go_repository(
    name = "com_github_whyrusleeping_go_logging",
    commit = "0457bb6b88fc1973573aaf6b5145d8d3ae972390",
    importpath = "github.com/whyrusleeping/go-logging",
)

go_repository(
    name = "com_github_mattn_go_isatty",
    commit = "369ecd8cea9851e459abb67eb171853e3986591e",
    importpath = "github.com/mattn/go-isatty",
)

go_repository(
    name = "com_github_libp2p_go_stream_muxer",
    commit = "85c59fe9952848ae238f31e07272ab0d1f0309d6",
    importpath = "github.com/libp2p/go-stream-muxer",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_transport_upgrader",
    commit = "83703858bb4649336a7bc91d359a3c69e9c932ce",
    importpath = "github.com/libp2p/go-libp2p-transport-upgrader",
)

go_repository(
    name = "com_github_libp2p_go_testutil",
    commit = "873eaa1a26ba46f53f62d5ffd4193f74c73ff76a",
    importpath = "github.com/libp2p/go-testutil",
)

go_repository(
    name = "com_github_whyrusleeping_go_smux_multistream",
    commit = "8e5c10881353c91b4214466d916ad8429508c056",
    importpath = "github.com/whyrusleeping/go-smux-multistream",
)

go_repository(
    name = "com_github_libp2p_go_maddr_filter",
    commit = "f2e84f9bcf48cffa2188e236ccb596891d6b574b",
    importpath = "github.com/libp2p/go-maddr-filter",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_transport",
    commit = "e8580c8a519d0fd04959a4ab25777d2819801447",
    importpath = "github.com/libp2p/go-libp2p-transport",
)

go_repository(
    name = "com_github_libp2p_go_addr_util",
    commit = "4b89cdc9db2b86a95dcb96b32194a65f36f78a3c",
    importpath = "github.com/libp2p/go-addr-util",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_interface_pnet",
    commit = "1db4baf9fd85048385e8043b588b3708ffeac1e5",
    importpath = "github.com/libp2p/go-libp2p-interface-pnet",
)

go_repository(
    name = "com_github_libp2p_go_conn_security",
    commit = "b2fb4ac68c410f0365f4aa807bde5cfdcb73acdc",
    importpath = "github.com/libp2p/go-conn-security",
)

go_repository(
    name = "com_github_whyrusleeping_timecache",
    commit = "cfcb2f1abfee846c430233aef0b630a946e0a5a6",
    importpath = "github.com/whyrusleeping/timecache",
)

go_repository(
    name = "com_github_miekg_dns",
    commit = "092d7745b47c1a19a6533514ff7a05387b5c5bf9",
    importpath = "github.com/miekg/dns",
)

go_repository(
    name = "com_github_opentracing_opentracing_go",
    commit = "25a84ff92183e2f8ac018ba1db54f8a07b3c0e04",
    importpath = "github.com/opentracing/opentracing-go",
)

go_repository(
    name = "com_github_libp2p_go_reuseport",
    commit = "6a75ffd916e5e70a5231da13d63ee5bba1109742",
    importpath = "github.com/libp2p/go-reuseport",
)

go_repository(
    name = "com_github_huin_goupnp",
    commit = "656e61dfadd241c7cbdd22a023fa81ecb6860ea8",
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
    commit = "b2ce2384e17bbe0c6d34077efa39dbab3e09123b",
    importpath = "github.com/satori/go.uuid",
)

go_repository(
    name = "com_github_sirupsen_logrus",
    commit = "fa3c1df5136d05b1b54b72794cf89579f2b39f7b",
    importpath = "github.com/sirupsen/logrus",
)

go_repository(
    name = "org_golang_x_sys",
    commit = "a34e9553db1e492c9a76e60db2296ae7e5fbb772",
    importpath = "golang.org/x/sys",
)

go_repository(
    name = "com_github_whyrusleeping_yamux",
    commit = "5364a42fe4b5efa5967c11c8f9b0f049cac0c4a9",
    importpath = "github.com/whyrusleeping/yamux",
)

go_repository(
    name = "com_github_libp2p_go_flow_metrics",
    commit = "5b4a5750cdd34754b1a417b4c7c7d1c24790b4d4",
    importpath = "github.com/libp2p/go-flow-metrics",
)

go_repository(
    name = "com_github_libp2p_go_msgio",
    commit = "2efd92d62d528c56c6711346e82df0c74a5742a8",
    importpath = "github.com/libp2p/go-msgio",
)

go_repository(
    name = "com_github_jackpal_gateway",
    commit = "cbcf4e3f3baee7952fc386c8b2534af4d267c875",
    importpath = "github.com/jackpal/gateway",
)

go_repository(
    name = "com_github_whyrusleeping_multiaddr_filter",
    commit = "e903e4adabd70b78bc9293b6ee4f359afb3f9f59",
    importpath = "github.com/whyrusleeping/multiaddr-filter",
)

go_repository(
    name = "com_github_libp2p_go_ws_transport",
    commit = "7b8b65749f06928c9aba331ea62f13000ac43846",
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
    commit = "8a125a1f2dbb108d778e6db8892119e4351b8c89",
    importpath = "github.com/libp2p/go-reuseport-transport",
)

go_repository(
    name = "com_github_libp2p_go_sockaddr",
    commit = "a7494d4eefeb607c8bc491cf8850a6e8dbd41cab",
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
    commit = "b4ed6c794676f7b9f186d0355bc00033b0167c80",
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
    commit = "19c1feb9386425afc25c2931e44eb57dffc78a89",
    importpath = "github.com/libp2p/go-libp2p-blankhost",
)

go_repository(
    name = "com_github_steakknife_hamming",
    commit = "c99c65617cd3d686aea8365fe563d6542f01d940",
    importpath = "github.com/steakknife/hamming",
)

go_repository(
    name = "io_opencensus_go",
    commit = "22a302cdd30bb5e3c0c894c70decd0b53e96eda0",
    importpath = "go.opencensus.io",
)

go_repository(
    name = "org_golang_google_api",
    commit = "1e61edccee94899d35fb4f2be2c3dbf636d1bb2b",
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
    commit = "d8c06bb2b797de6c208b2a7995327ba1b94c77da",
    importpath = "github.com/prometheus/prometheus",
)

go_repository(
    name = "com_github_beorn7_perks",
    commit = "3a771d992973f24aa725d07868b467d1ddfceafb",
    importpath = "github.com/beorn7/perks",
)

go_repository(
    name = "com_github_matttproud_golang_protobuf_extensions",
    commit = "c182affec369e30f25d3eb8cd8a478dee585ae7d",
    importpath = "github.com/matttproud/golang_protobuf_extensions",
)

go_repository(
    name = "com_github_boltdb_bolt",
    commit = "fd01fc79c553a8e99d512a07e8e0c63d4a3ccfc5",
    importpath = "github.com/boltdb/bolt",
)

go_repository(
    name = "com_github_pborman_uuid",
    commit = "8b1b92947f46224e3b97bb1a3a5b0382be00d31e",
    importpath = "github.com/pborman/uuid",
)

go_repository(
    name = "com_github_libp2p_go_buffer_pool",
    commit = "eecb57f6a721b4a66062ebeb61d1da7ce1c38fbf",
    importpath = "github.com/libp2p/go-buffer-pool",
)

go_repository(
    name = "com_github_libp2p_go_mplex",
    commit = "070b6c36acd45760112a21fda1a6bff6feaa226e",
    importpath = "github.com/libp2p/go-mplex",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_pubsub",
    build_file_proto_mode = "disable_global",
    commit = "f8276dc198de05a85607bd2f24f26fa61381a50e",
    importpath = "github.com/libp2p/go-libp2p-pubsub",
)

go_repository(
    name = "com_github_ipfs_go_ipfs_util",
    commit = "a4bb5361e49427531f9a716ead2ce4bd9bdd7959",
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
    commit = "f1a42512708e2f510c3f3019cae549a70bc3049b",
    importpath = "github.com/libp2p/go-libp2p-kad-dht",
)

go_repository(
    name = "com_github_ipfs_go_datastore",
    commit = "1b371980cb3fe9edba40c0f0584042e5cb4477a4",
    importpath = "github.com/ipfs/go-datastore",
)

go_repository(
    name = "com_github_whyrusleeping_base32",
    commit = "c30ac30633ccdabefe87eb12465113f06f1bab75",
    importpath = "github.com/whyrusleeping/base32",
)

go_repository(
    name = "com_github_ipfs_go_cid",
    commit = "e7e67e08cfba888a4297224402e12832bdc15ea0",
    importpath = "github.com/ipfs/go-cid",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_record",
    build_file_proto_mode = "disable_global",
    commit = "0f29df9dd657d87a8f1bd74020d39381a178ad19",
    importpath = "github.com/libp2p/go-libp2p-record",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_routing",
    commit = "efb912a4c564986aa9bb541a6c853919cc9a72a7",
    importpath = "github.com/libp2p/go-libp2p-routing",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_kbucket",
    commit = "719fa0d0a8531cbdf0be22205eb5f647e65779a1",
    importpath = "github.com/libp2p/go-libp2p-kbucket",
)

go_repository(
    name = "com_github_jbenet_go_context",
    commit = "d14ea06fba99483203c19d92cfcd13ebe73135f4",
    importpath = "github.com/jbenet/go-context",
)

go_repository(
    name = "com_github_ipfs_go_todocounter",
    commit = "bc75efcf13e6e50fbba27679ba5451585d70c954",
    importpath = "github.com/ipfs/go-todocounter",
)

go_repository(
    name = "com_github_whyrusleeping_go_keyspace",
    commit = "5b898ac5add1da7178a4a98e69cb7b9205c085ee",
    importpath = "github.com/whyrusleeping/go-keyspace",
)

go_repository(
    name = "com_github_multiformats_go_multibase",
    commit = "d63641945dc1749baa23686ad0564ad63fef0493",
    importpath = "github.com/multiformats/go-multibase",
)

go_repository(
    name = "com_github_hashicorp_golang_lru",
    commit = "7087cb70de9f7a8bc0a10c375cb0d2280a8edf9c",
    importpath = "github.com/hashicorp/golang-lru",
)

go_repository(
    name = "com_github_ipfs_go_ipfs_addr",
    commit = "b43fab3140b14e74a04dd8722f4373c449039a66",
    importpath = "github.com/ipfs/go-ipfs-addr",
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
    commit = "51d6538a90f86fe93ac480b35f37b2be17fef232",
    importpath = "gopkg.in/yaml.v2",
)

go_repository(
    name = "com_github_spf13_pflag",
    commit = "24fa6976df40757dce6aea913e7b81ade90530e1",
    importpath = "github.com/spf13/pflag",
)

go_repository(
    name = "com_github_spf13_cobra",
    commit = "7547e83b2d85fd1893c7d76916f67689d761fecb",
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
    commit = "886a7fbe3eb1c874d46f623bfa70af45f425b3d1",
    importpath = "github.com/hashicorp/go-multierror",
)

go_repository(
    name = "com_github_hashicorp_errwrap",
    commit = "8a6fb523712970c966eefc6b39ed2c5e74880354",
    importpath = "github.com/hashicorp/errwrap",
)

go_repository(
    name = "com_google_cloud_go",
    commit = "7e92b4d51047c91ec65bab1144f1c3183d7db9d6",
    importpath = "cloud.google.com/go",
)

go_repository(
    name = "com_github_inconshreveable_mousetrap",
    commit = "76626ae9c91c4f2a10f34cad8ce83ea42c93bb75",
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
    commit = "a9c2755c3d1672dbe6a7e4a5d182169fa30b6a8e",
    importpath = "github.com/multiformats/go-base32",
)
