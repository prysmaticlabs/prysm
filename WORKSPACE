load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.16.2/rules_go-0.16.2.tar.gz",
    sha256 = "f87fa87475ea107b3c69196f39c82b7bbf58fe27c62a338684c20ca17d1d8613",
)

http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.15.0/bazel-gazelle-0.15.0.tar.gz"],
    sha256 = "6e875ab4b6bf64a38c352887760f21203ab054676d9c1b274963907e0768740d",
)

http_archive(
    name = "com_github_atlassian_bazel_tools",
    strip_prefix = "bazel-tools-6fef37f33dfa0189be9df4d3d60e6291bfe71177",
    urls = ["https://github.com/atlassian/bazel-tools/archive/6fef37f33dfa0189be9df4d3d60e6291bfe71177.zip"],
)

git_repository(
    name = "io_bazel_rules_docker",
    commit = "7401cb256222615c497c0dee5a4de5724a4f4cc7",  # 2018-06-22
    remote = "https://github.com/bazelbuild/rules_docker.git",
)

load("@io_bazel_rules_docker//docker:docker.bzl", "docker_repositories")

docker_repositories()

git_repository(
    name = "build_bazel_rules_nodejs",
    remote = "https://github.com/bazelbuild/rules_nodejs.git",
    tag = "0.16.0",
)

load("@build_bazel_rules_nodejs//:package.bzl", "rules_nodejs_dependencies")

rules_nodejs_dependencies()

load("@build_bazel_rules_nodejs//:defs.bzl", "node_repositories", "yarn_install")

node_repositories()

yarn_install(
    name = "npm",
    package_json = "//:package.json",
    yarn_lock = "//:yarn.lock",
)

# This requires rules_docker to be fully instantiated before it is pulled in.
git_repository(
    name = "io_bazel_rules_k8s",
    commit = "2054f7bf4d51f9e439313c56d7a208960a8a179f",  # 2018-07-29
    remote = "https://github.com/bazelbuild/rules_k8s.git",
)

load("@io_bazel_rules_k8s//k8s:k8s.bzl", "k8s_repositories", "k8s_defaults")

k8s_repositories()

_CLUSTER = "minikube"

_NAMESPACE = "default"

[k8s_defaults(
    name = "k8s_" + kind,
    cluster = _CLUSTER,
    #context = _CONTEXT,
    kind = kind,
    namespace = _NAMESPACE,
) for kind in [
    "deploy",
    "service",
    "secret",
    "priority_class",
    "pod",
]]

load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")

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

git_repository(
    name = "io_kubernetes_build",
    commit = "4ce715fbe67d8fbed05ec2bb47a148e754100a4b",
    remote = "https://github.com/kubernetes/repo-infra.git",
)

git_repository(
    name = "com_github_jmhodges_bazel_gomock",
    commit = "5b73edb74e569ff404b3beffc809d6d9f205e0e4",
    remote = "https://github.com/jmhodges/bazel_gomock.git",
)

go_repository(
    name = "com_github_golang_mock",
    commit = "8a44ef6e8be577e050008c7886f24fc705d709fb",
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
    # Last updated September 09, 2018
    commit = "f4b3f83362a4cf2928e57914af040aea76c8a7d6",
)

go_repository(
    name = "com_github_urfave_cli",
    importpath = "github.com/urfave/cli",
    commit = "8e01ec4cd3e2d84ab2fe90d8210528ffbb06d8ff",
)

go_repository(
    name = "com_github_go_yaml_yaml",
    importpath = "github.com/go-yaml/yaml",
    commit = "51d6538a90f86fe93ac480b35f37b2be17fef232",
)

go_repository(
    name = "com_github_x_cray_logrus_prefixed_formatter",
    importpath = "github.com/x-cray/logrus-prefixed-formatter",
    commit = "bb2702d423886830dee131692131d35648c382e2",
)

go_repository(
    name = "com_github_mgutz_ansi",
    importpath = "github.com/mgutz/ansi",
    commit = "9520e82c474b0a04dd04f8a40959027271bab992",
)

go_repository(
    name = "com_github_fjl_memsize",
    importpath = "github.com/fjl/memsize",
    commit = "2a09253e352a56f419bd88effab0483f52da4c7d",
)

go_repository(
    name = "com_github_libp2p_go_libp2p",
    commit = "9356373d00ab1aef3e20c8202b682f93799acf78",
    importpath = "github.com/libp2p/go-libp2p",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_peer",
    commit = "dd9b45c0649b38aebe65f98cb460676b4214a42c",
    importpath = "github.com/libp2p/go-libp2p-peer",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_crypto",
    build_file_proto_mode = "disable_global",
    commit = "3120e9f9526fe05f2d3905961a5e0701b85579d9",
    importpath = "github.com/libp2p/go-libp2p-crypto",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr",
    commit = "ec8630b6b7436b5d7f6c1c2366d3d7214d1b29e2",
    importpath = "github.com/multiformats/go-multiaddr",
)

go_repository(
    name = "com_github_ipfs_go_log",
    commit = "de9a213953d6ec0e053b56e9d79800565c3fc9ca",
    importpath = "github.com/ipfs/go-log",
)

go_repository(
    name = "com_github_multiformats_go_multihash",
    commit = "8be2a682ab9f254311de1375145a2f78a809b07d",
    importpath = "github.com/multiformats/go-multihash",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_swarm",
    commit = "839f88f8de4d0f8300facdcdf7aa2124d020b2b6",
    importpath = "github.com/libp2p/go-libp2p-swarm",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_host",
    commit = "5ab1f4ea0a8ad6f9cd7264bc9a3b6d908d07e21a",
    importpath = "github.com/libp2p/go-libp2p-host",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_peerstore",
    commit = "6295e61c9fd2f13ad159c6241be3b371918045e2",
    importpath = "github.com/libp2p/go-libp2p-peerstore",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_circuit",
    commit = "f83937ed3384bb289ba39ee0c4f428f26013390a",
    importpath = "github.com/libp2p/go-libp2p-circuit",
)

go_repository(
    name = "com_github_coreos_go_semver",
    commit = "e214231b295a8ea9479f11b70b35d5acf3556d9b",
    importpath = "github.com/coreos/go-semver",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_interface_connmgr",
    commit = "a75b79dbf71f47f55b00928ead7d5a44355e2b62",
    importpath = "github.com/libp2p/go-libp2p-interface-connmgr",
)

go_repository(
    name = "com_github_libp2p_go_conn_security_multistream",
    commit = "cda5d4f4ed431a63fdba6d48abab2acb9d2c7a9d",
    importpath = "github.com/libp2p/go-conn-security-multistream",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_metrics",
    commit = "20c0e3fed14ddf84ac8192038accfd393610ed82",
    importpath = "github.com/libp2p/go-libp2p-metrics",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_net",
    commit = "22c96766db92ab111e506ebcd9cc6511ed32e553",
    importpath = "github.com/libp2p/go-libp2p-net",
)

go_repository(
    name = "com_github_whyrusleeping_mafmt",
    commit = "3b86bcbec8cbb09d205c1492e898ce3d0e81c4d5",
    importpath = "github.com/whyrusleeping/mafmt",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_net",
    commit = "cba4f9fea8613343eb7ecc4ddadd8e7298a00c39",
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
    commit = "efa589957cd060542a26d2dd7832fd6a6c6c3ade",
    importpath = "github.com/mattn/go-colorable",
)

go_repository(
    name = "com_github_whyrusleeping_mdns",
    commit = "ef14215e6b30606f4ce84174ed7a644a05cb1af3",
    importpath = "github.com/whyrusleeping/mdns",
)

go_repository(
    name = "com_github_btcsuite_btcd",
    commit = "67e573d211ace594f1366b4ce9d39726c4b19bd0",
    importpath = "github.com/btcsuite/btcd",
)

go_repository(
    name = "com_github_minio_sha256_simd",
    commit = "ad98a36ba0da87206e3378c556abbfeaeaa98668",
    importpath = "github.com/minio/sha256-simd",
)

go_repository(
    name = "com_github_mr_tron_base58",
    commit = "4df4dc6e86a912614d09719d10cad427b087cbfb",
    importpath = "github.com/mr-tron/base58",
)

go_repository(
    name = "com_github_whyrusleeping_go_smux_yamux",
    commit = "49458276a01f7fbc32ff62c8955fa3e852b8e772",
    importpath = "github.com/whyrusleeping/go-smux-yamux",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_secio",
    build_file_proto_mode = "disable_global",
    commit = "8f95e95b9fedc69b1367362a14f1ad3b5bd5bd46",
    importpath = "github.com/libp2p/go-libp2p-secio",
)

go_repository(
    name = "com_github_libp2p_go_tcp_transport",
    commit = "4a25127ad66b71ae4c91f1f42205b2ce679dd926",
    importpath = "github.com/libp2p/go-tcp-transport",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_protocol",
    commit = "b29f3d97e3a2fb8b29c5d04290e6cb5c5018004b",
    importpath = "github.com/libp2p/go-libp2p-protocol",
)

go_repository(
    name = "com_github_jbenet_goprocess",
    commit = "b497e2f366b8624394fb2e89c10ab607bebdde0b",
    importpath = "github.com/jbenet/goprocess",
)

go_repository(
    name = "com_github_multiformats_go_multistream",
    commit = "aea59cd120a7f60ed64cc98ffc1af2e6a84c470f",
    importpath = "github.com/multiformats/go-multistream",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_loggables",
    commit = "2edffda90e410fab8ca3663511d33b59314d4b07",
    importpath = "github.com/libp2p/go-libp2p-loggables",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_nat",
    commit = "b82aac8589e138824736b2a9d466981dbce6b0d4",
    importpath = "github.com/libp2p/go-libp2p-nat",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_dns",
    commit = "78f39e8892d4f8c5c9f18679cc06d0b40ecab8cf",
    importpath = "github.com/multiformats/go-multiaddr-dns",
)

go_repository(
    name = "com_github_fd_go_nat",
    commit = "bad65a492f32121a87197f4a085905c35e2a367e",
    importpath = "github.com/fd/go-nat",
)

go_repository(
    name = "com_github_whyrusleeping_go_logging",
    commit = "0457bb6b88fc1973573aaf6b5145d8d3ae972390",
    importpath = "github.com/whyrusleeping/go-logging",
)

go_repository(
    name = "com_github_mattn_go_isatty",
    commit = "6ca4dbf54d38eea1a992b3c722a76a5d1c4cb25c",
    importpath = "github.com/mattn/go-isatty",
)

go_repository(
    name = "com_github_libp2p_go_stream_muxer",
    commit = "9c6bd93eecbbab56630bb2688bb435d9fd1dfeb1",
    importpath = "github.com/libp2p/go-stream-muxer",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_transport_upgrader",
    commit = "8dde02b5e75342c09725bc601cf28e9e98f920c7",
    importpath = "github.com/libp2p/go-libp2p-transport-upgrader",
)

go_repository(
    name = "com_github_libp2p_go_testutil",
    commit = "f967bbd5fcb7fb6337504e5d78c53c865e80733c",
    importpath = "github.com/libp2p/go-testutil",
)

go_repository(
    name = "com_github_whyrusleeping_go_smux_multistream",
    commit = "e799b10bc6eea2cc5ce18f7b7b4d8e1a0439476d",
    importpath = "github.com/whyrusleeping/go-smux-multistream",
)

go_repository(
    name = "com_github_libp2p_go_maddr_filter",
    commit = "7f7ca1e79c453741adb1cc10d8892b186952e9e1",
    importpath = "github.com/libp2p/go-maddr-filter",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_transport",
    commit = "e6d021be15cb2bfc73fb24d3b16848bc2825bbf6",
    importpath = "github.com/libp2p/go-libp2p-transport",
)

go_repository(
    name = "com_github_libp2p_go_addr_util",
    commit = "6e2ee01c0ae21694bcc33aaa7d6ce214279134f6",
    importpath = "github.com/libp2p/go-addr-util",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_interface_pnet",
    commit = "d240acf619f63dfb776821a1d4d28a918f77edd5",
    importpath = "github.com/libp2p/go-libp2p-interface-pnet",
)

go_repository(
    name = "com_github_libp2p_go_conn_security",
    commit = "42db08f68cdfd8a1ec46eefde49160c2a9559b72",
    importpath = "github.com/libp2p/go-conn-security",
)

go_repository(
    name = "com_github_whyrusleeping_timecache",
    commit = "cfcb2f1abfee846c430233aef0b630a946e0a5a6",
    importpath = "github.com/whyrusleeping/timecache",
)

go_repository(
    name = "com_github_miekg_dns",
    commit = "3e6e47bc11bc7f93f9e2f1c7bd6481ba4802808b",
    importpath = "github.com/miekg/dns",
)

go_repository(
    name = "com_github_opentracing_opentracing_go",
    commit = "bd9c3193394760d98b2fa6ebb2291f0cd1d06a7d",
    importpath = "github.com/opentracing/opentracing-go",
)

go_repository(
    name = "com_github_libp2p_go_reuseport",
    commit = "dd0c37d7767bc38280bd9813145b65f8bd560629",
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
    commit = "36e9d2ebbde5e3f13ab2e25625fd453271d6522e",
    importpath = "github.com/satori/go.uuid",
)

go_repository(
    name = "com_github_sirupsen_logrus",
    importpath = "github.com/sirupsen/logrus",
    commit = "e54a77765aca7bbdd8e56c1c54f60579968b2dc9",
)

go_repository(
    name = "org_golang_x_sys",
    commit = "3c6ecd8f22c6f40fbeec94c000a069d7d87c7624",
    importpath = "golang.org/x/sys",
)

go_repository(
    name = "com_github_whyrusleeping_yamux",
    commit = "5364a42fe4b5efa5967c11c8f9b0f049cac0c4a9",
    importpath = "github.com/whyrusleeping/yamux",
)

go_repository(
    name = "com_github_libp2p_go_flow_metrics",
    commit = "7e5a55af485341567f98d6847a373eb5ddcdcd43",
    importpath = "github.com/libp2p/go-flow-metrics",
)

go_repository(
    name = "com_github_libp2p_go_msgio",
    commit = "031a413e66129d593337a3f5948d9364e7fc6d43",
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
    commit = "246ec4b8bd9a5a539827eca50a6e6d4ce50bb056",
    importpath = "github.com/libp2p/go-ws-transport",
)

go_repository(
    name = "org_golang_x_crypto",
    commit = "a49355c7e3f8fe157a85be2f77e6e269a0f89602",
    importpath = "golang.org/x/crypto",
)

go_repository(
    name = "com_github_jackpal_go_nat_pmp",
    commit = "d89d09f6f3329bc3c2479aa3cafd76a5aa93a35c",
    importpath = "github.com/jackpal/go-nat-pmp",
)

go_repository(
    name = "com_github_libp2p_go_reuseport_transport",
    commit = "58ea7103ffb4b5eb248c4421e60fdb30e9a56dad",
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
    commit = "483fb8d7c32fcb4b5636cd293a92e3935932e2f4",
    importpath = "github.com/gorilla/websocket",
)

go_repository(
    name = "com_github_whyrusleeping_go_smux_multiplex",
    commit = "2b855d4f3a3051b0133f7783bffe06e4b7833d1e",
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
    commit = "c4c61651e9e37fa117f53c5a906d3b63090d8445",
    importpath = "github.com/syndtr/goleveldb",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_blankhost",
    commit = "23a786e24e247b25b8708c8538d9e16cbfa4e4c9",
    importpath = "github.com/libp2p/go-libp2p-blankhost",
)

go_repository(
    name = "com_github_steakknife_hamming",
    commit = "c99c65617cd3d686aea8365fe563d6542f01d940",
    importpath = "github.com/steakknife/hamming",
)

go_repository(
    name = "io_opencensus_go",
    commit = "f21fe3feadc5461b952191052818685a410428d4",
    importpath = "go.opencensus.io",
)

go_repository(
    name = "org_golang_google_api",
    commit = "7ca32eb868bf53ea2fc406698eb98583a8073d19",
    importpath = "google.golang.org/api",
)

go_repository(
    name = "org_golang_x_sync",
    commit = "1d60e4601c6fd243af51cc01ddf169918a5407ca",
    importpath = "golang.org/x/sync",
)

go_repository(
    name = "com_github_golang_lint",
    commit = "c67002cb31c3a748b7688c27f20d8358b4193582",
    importpath = "github.com/golang/lint",
)

go_repository(
    name = "org_golang_x_lint",
    commit = "06c8688daad7faa9da5a0c2f163a3d14aac986ca",
    importpath = "golang.org/x/lint",
)

go_repository(
    name = "com_github_aristanetworks_goarista",
    commit = "5faa74ffbed7096292069fdcd0eae96146a3158a",
    importpath = "github.com/aristanetworks/goarista",
)

go_repository(
    name = "com_github_prometheus_client_golang",
    commit = "1cafe34db7fdec6022e17e00e1c1ea501022f3e4",
    importpath = "github.com/prometheus/client_golang",
)

go_repository(
    name = "com_github_prometheus_client_model",
    commit = "5c3871d89910bfb32f5fcab2aa4b9ec68e65a99f",
    importpath = "github.com/prometheus/client_model",
)

go_repository(
    name = "com_github_prometheus_common",
    commit = "bcb74de08d37a417cb6789eec1d6c810040f0470",
    importpath = "github.com/prometheus/common",
)

go_repository(
    name = "com_github_prometheus_procfs",
    commit = "185b4288413d2a0dd0806f78c90dde719829e5ae",
    importpath = "github.com/prometheus/procfs",
)

go_repository(
    name = "com_github_prometheus_prometheus",
    commit = "5464c64853f2c795163286ca81201abe4ab7e454",
    importpath = "github.com/prometheus/prometheus",
)

go_repository(
    name = "com_github_beorn7_perks",
    commit = "3a771d992973f24aa725d07868b467d1ddfceafb",
    importpath = "github.com/beorn7/perks",
)

go_repository(
    name = "com_github_matttproud_golang_protobuf_extensions",
    commit = "c12348ce28de40eed0136aa2b644d0ee0650e56c",
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
    commit = "058210c5a0d042677367d923eb8a6dc072a15f7f",
    importpath = "github.com/libp2p/go-buffer-pool",
)

go_repository(
    name = "com_github_libp2p_go_mplex",
    commit = "0ef5fed5ba589e7e8776c274510cfe0d806bac9c",
    importpath = "github.com/libp2p/go-mplex",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_pubsub",
    build_file_proto_mode = "disable_global",
    commit = "067e546891ad5f1cf311070cb338670423e327d8",
    importpath = "github.com/libp2p/go-libp2p-pubsub",
)

go_repository(
    name = "com_github_ipfs_go_ipfs_util",
    commit = "05b6094b6fa9c1e49b4b941061fdc147db1a21b7",
    importpath = "github.com/ipfs/go-ipfs-util",
)

go_repository(
    name = "com_github_google_uuid",
    commit = "9b3b1e0f5f99ae461456d768e7d301a7acdaa2d8",
    importpath = "github.com/google/uuid",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_kad_dht",
    commit = "d70e92744b101ddf9ed93066a54ce128fa53aaa8",
    importpath = "github.com/libp2p/go-libp2p-kad-dht",
    build_file_proto_mode = "disable_global",
)

go_repository(
    name = "com_github_ipfs_go_datastore",
    commit = "277eeb2fded2592392256b6e0f80111208ed8aca",
    importpath = "github.com/ipfs/go-datastore",
)

go_repository(
    name = "com_github_whyrusleeping_base32",
    commit = "c30ac30633ccdabefe87eb12465113f06f1bab75",
    importpath = "github.com/whyrusleeping/base32",
)

go_repository(
    name = "com_github_ipfs_go_cid",
    commit = "033594dcd6201e8e8628659c8d584bb800d1734c",
    importpath = "github.com/ipfs/go-cid",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_record",
    commit = "237ab9e10af172232eedad9e63ea8983c50859b1",
    importpath = "github.com/libp2p/go-libp2p-record",
    build_file_proto_mode = "disable_global",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_routing",
    commit = "cb72d923dcde7c3af89d09032443515bd0fe7075",
    importpath = "github.com/libp2p/go-libp2p-routing",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_kbucket",
    commit = "b181991db757142b92d913a2f93ea3ee92288aad",
    importpath = "github.com/libp2p/go-libp2p-kbucket",
)

go_repository(
    name = "com_github_jbenet_go_context",
    commit = "d14ea06fba99483203c19d92cfcd13ebe73135f4",
    importpath = "github.com/jbenet/go-context",
)

go_repository(
    name = "com_github_ipfs_go_todocounter",
    commit = "1e832b829506383050e6eebd12e05ea41a451532",
    importpath = "github.com/ipfs/go-todocounter",
)

go_repository(
    name = "com_github_whyrusleeping_go_keyspace",
    commit = "5b898ac5add1da7178a4a98e69cb7b9205c085ee",
    importpath = "github.com/whyrusleeping/go-keyspace",
)

go_repository(
    name = "com_github_multiformats_go_multibase",
    commit = "bb91b53e5695e699a86654d77d03db7bc7506d12",
    importpath = "github.com/multiformats/go-multibase",
)

go_repository(
    name = "com_github_hashicorp_golang_lru",
    commit = "20f1fb78b0740ba8c3cb143a61e86ba5c8669768",
    importpath = "github.com/hashicorp/golang-lru",
)

go_repository(
    name = "com_github_ipfs_go_ipfs_addr",
    commit = "489e2d62e87b8986d703b3b28d62a7fcc3978222",
    importpath = "github.com/ipfs/go-ipfs-addr",
)
