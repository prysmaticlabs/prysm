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
    sha256 = "489f85d7c17a901b9069c95f656154fdf1385db00f3aeb3e0319aed8745f9453",
    url = "https://github.com/ethereum/eth2.0-spec-tests/releases/download/v0.11.3/general.tar.gz",
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
    sha256 = "b83000fbcb60b7a5b8c0e805f3fee6953b17bfe0fe6658416e7d99e6d261f284",
    url = "https://github.com/ethereum/eth2.0-spec-tests/releases/download/v0.11.3/minimal.tar.gz",
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
    sha256 = "ae0c09ab49afa69085c91f9e2f2f4de6526d43b927609839b1597c674b4dccde",
    url = "https://github.com/ethereum/eth2.0-spec-tests/releases/download/v0.11.3/mainnet.tar.gz",
)

http_archive(
    name = "com_github_bazelbuild_buildtools",
    sha256 = "b5d7dbc6832f11b6468328a376de05959a1a9e4e9f5622499d3bab509c26b46a",
    strip_prefix = "buildtools-bf564b4925ab5876a3f64d8b90fab7f769013d42",
    url = "https://github.com/bazelbuild/buildtools/archive/bf564b4925ab5876a3f64d8b90fab7f769013d42.zip",
)

load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")

buildifier_dependencies()

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
    sha256 = "3b99a36c082a67ef6295eabd4fb3e32ab0bff7c6b0d397d6352697335f4e57eb",
    strip_prefix = "librdkafka-1.4.2",
    urls = ["https://github.com/edenhill/librdkafka/archive/v1.4.2.tar.gz"],
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

load("//:deps.bzl", "prysm_deps")

# gazelle:repository_macro deps.bzl%prysm_deps
prysm_deps()

load("@com_github_prysmaticlabs_go_ssz//:deps.bzl", "go_ssz_dependencies")

go_ssz_dependencies()

load("@prysm//third_party/herumi:herumi.bzl", "bls_dependencies")

bls_dependencies()

# Do NOT add new go dependencies here! Refer to DEPENDENCIES.md!

go_repository(
    name = "com_github_ajstarks_svgo",
    importpath = "github.com/ajstarks/svgo",
    sum = "h1:wVe6/Ea46ZMeNkQjjBW6xcqyQA/j5e0D6GytH95g0gQ=",
    version = "v0.0.0-20180226025133-644b8db467af",
)

go_repository(
    name = "com_github_andreyvit_diff",
    importpath = "github.com/andreyvit/diff",
    sum = "h1:bvNMNQO63//z+xNgfBlViaCIJKLlCJ6/fmUseuG0wVQ=",
    version = "v0.0.0-20170406064948-c7f18ee00883",
)

go_repository(
    name = "com_github_apache_arrow_go_arrow",
    importpath = "github.com/apache/arrow/go/arrow",
    sum = "h1:nxAtV4VajJDhKysp2kdcJZsq8Ss1xSA0vZTkVHHJd0E=",
    version = "v0.0.0-20191024131854-af6fa24be0db",
)

go_repository(
    name = "com_github_apilayer_freegeoip",
    importpath = "github.com/apilayer/freegeoip",
    sum = "h1:z1u2gv0/rsSi/HqMDB436AiUROXXim7st5DOg4Ikl4A=",
    version = "v3.5.0+incompatible",
)

go_repository(
    name = "com_github_bmizerany_pat",
    importpath = "github.com/bmizerany/pat",
    sum = "h1:y4B3+GPxKlrigF1ha5FFErxK+sr6sWxQovRMzwMhejo=",
    version = "v0.0.0-20170815010413-6226ea591a40",
)

go_repository(
    name = "com_github_boltdb_bolt",
    importpath = "github.com/boltdb/bolt",
    sum = "h1:JQmyP4ZBrce+ZQu0dY660FMfatumYDLun9hBCUVIkF4=",
    version = "v1.3.1",
)

go_repository(
    name = "com_github_c_bata_go_prompt",
    importpath = "github.com/c-bata/go-prompt",
    sum = "h1:uyKRz6Z6DUyj49QVijyM339UJV9yhbr70gESwbNU3e0=",
    version = "v0.2.2",
)

go_repository(
    name = "com_github_chzyer_logex",
    importpath = "github.com/chzyer/logex",
    sum = "h1:Swpa1K6QvQznwJRcfTfQJmTE72DqScAa40E+fbHEXEE=",
    version = "v1.1.10",
)

go_repository(
    name = "com_github_chzyer_readline",
    importpath = "github.com/chzyer/readline",
    sum = "h1:fY5BOSpyZCqRo5OhCuC+XN+r/bBCmeuuJtjz+bCNIf8=",
    version = "v0.0.0-20180603132655-2972be24d48e",
)

go_repository(
    name = "com_github_chzyer_test",
    importpath = "github.com/chzyer/test",
    sum = "h1:q763qf9huN11kDQavWsoZXJNW3xEE4JJyHa5Q25/sd8=",
    version = "v0.0.0-20180213035817-a1ea475d72b1",
)

go_repository(
    name = "com_github_data_dog_go_sqlmock",
    importpath = "github.com/DATA-DOG/go-sqlmock",
    sum = "h1:CWUqKXe0s8A2z6qCgkP4Kru7wC11YoAnoupUKFDnH08=",
    version = "v1.3.3",
)

go_repository(
    name = "com_github_dave_jennifer",
    importpath = "github.com/dave/jennifer",
    sum = "h1:S15ZkFMRoJ36mGAQgWL1tnr0NQJh9rZ8qatseX/VbBc=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_dgryski_go_bitstream",
    importpath = "github.com/dgryski/go-bitstream",
    sum = "h1:akOQj8IVgoeFfBTzGOEQakCYshWD6RNo1M5pivFXt70=",
    version = "v0.0.0-20180413035011-3522498ce2c8",
)

go_repository(
    name = "com_github_eclipse_paho_mqtt_golang",
    importpath = "github.com/eclipse/paho.mqtt.golang",
    sum = "h1:1F8mhG9+aO5/xpdtFkW4SxOJB67ukuDC3t2y2qayIX0=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_fogleman_gg",
    importpath = "github.com/fogleman/gg",
    sum = "h1:WXb3TSNmHp2vHoCroCIB1foO/yQ36swABL8aOVeDpgg=",
    version = "v1.2.1-0.20190220221249-0403632d5b90",
)

go_repository(
    name = "com_github_glycerine_go_unsnap_stream",
    importpath = "github.com/glycerine/go-unsnap-stream",
    sum = "h1:r04MMPyLHj/QwZuMJ5+7tJcBr1AQjpiAK/rZWRrQT7o=",
    version = "v0.0.0-20180323001048-9f0cb55181dd",
)

go_repository(
    name = "com_github_glycerine_goconvey",
    importpath = "github.com/glycerine/goconvey",
    sum = "h1:gclg6gY70GLy3PbkQ1AERPfmLMMagS60DKF78eWwLn8=",
    version = "v0.0.0-20190410193231-58a59202ab31",
)

go_repository(
    name = "com_github_go_gl_glfw",
    importpath = "github.com/go-gl/glfw",
    sum = "h1:QbL/5oDUmRBzO9/Z7Seo6zf912W/a6Sr4Eu0G/3Jho0=",
    version = "v0.0.0-20190409004039-e6da0acd62b1",
)

go_repository(
    name = "com_github_golang_freetype",
    importpath = "github.com/golang/freetype",
    sum = "h1:DACJavvAHhabrF08vX0COfcOBJRhZ8lUbR+ZWIs0Y5g=",
    version = "v0.0.0-20170609003504-e2365dfdc4a0",
)

go_repository(
    name = "com_github_golang_geo",
    importpath = "github.com/golang/geo",
    sum = "h1:lJwO/92dFXWeXOZdoGXgptLmNLwynMSHUmU6besqtiw=",
    version = "v0.0.0-20190916061304-5b978397cfec",
)

go_repository(
    name = "com_github_google_flatbuffers",
    importpath = "github.com/google/flatbuffers",
    sum = "h1:O7CEyB8Cb3/DmtxODGtLHcEvpr81Jm5qLg/hsHnxA2A=",
    version = "v1.11.0",
)

go_repository(
    name = "com_github_howeyc_fsnotify",
    importpath = "github.com/howeyc/fsnotify",
    sum = "h1:0gtV5JmOKH4A8SsFxG2BczSeXWWPvcMT0euZt5gDAxY=",
    version = "v0.9.0",
)

go_repository(
    name = "com_github_influxdata_flux",
    importpath = "github.com/influxdata/flux",
    sum = "h1:57tk1Oo4gpGIDbV12vUAPCMtLtThhaXzub1XRIuqv6A=",
    version = "v0.65.0",
)

go_repository(
    name = "com_github_influxdata_influxql",
    importpath = "github.com/influxdata/influxql",
    sum = "h1:sPsaumLFRPMwR5QtD3Up54HXpNND8Eu7G1vQFmi3quQ=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_influxdata_line_protocol",
    importpath = "github.com/influxdata/line-protocol",
    sum = "h1:/o3vQtpWJhvnIbXley4/jwzzqNeigJK9z+LZcJZ9zfM=",
    version = "v0.0.0-20180522152040-32c6aa80de5e",
)

go_repository(
    name = "com_github_influxdata_promql_v2",
    importpath = "github.com/influxdata/promql/v2",
    sum = "h1:kXn3p0D7zPw16rOtfDR+wo6aaiH8tSMfhPwONTxrlEc=",
    version = "v2.12.0",
)

go_repository(
    name = "com_github_influxdata_roaring",
    importpath = "github.com/influxdata/roaring",
    sum = "h1:UzJnB7VRL4PSkUJHwsyzseGOmrO/r4yA+AuxGJxiZmA=",
    version = "v0.4.13-0.20180809181101-fc520f41fab6",
)

go_repository(
    name = "com_github_influxdata_tdigest",
    importpath = "github.com/influxdata/tdigest",
    sum = "h1:MHTrDWmQpHq/hkq+7cw9oYAt2PqUw52TZazRA0N7PGE=",
    version = "v0.0.0-20181121200506-bf2b5ad3c0a9",
)

go_repository(
    name = "com_github_influxdata_usage_client",
    importpath = "github.com/influxdata/usage-client",
    sum = "h1:+TUUmaFa4YD1Q+7bH9o5NCHQGPMqZCYJiNW6lIIS9z4=",
    version = "v0.0.0-20160829180054-6d3895376368",
)

go_repository(
    name = "com_github_jsternberg_zap_logfmt",
    importpath = "github.com/jsternberg/zap-logfmt",
    sum = "h1:0Dz2s/eturmdUS34GM82JwNEdQ9hPoJgqptcEKcbpzY=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_jtolds_gls",
    importpath = "github.com/jtolds/gls",
    sum = "h1:xdiiI2gbIgH/gLH7ADydsJ1uDOEzR8yvV7C0MuV77Wo=",
    version = "v4.20.0+incompatible",
)

go_repository(
    name = "com_github_jung_kurt_gofpdf",
    importpath = "github.com/jung-kurt/gofpdf",
    sum = "h1:PJr+ZMXIecYc1Ey2zucXdR73SMBtgjPgwa31099IMv0=",
    version = "v1.0.3-0.20190309125859-24315acbbda5",
)

go_repository(
    name = "com_github_jwilder_encoding",
    importpath = "github.com/jwilder/encoding",
    sum = "h1:2jNeR4YUziVtswNP9sEFAI913cVrzH85T+8Q6LpYbT0=",
    version = "v0.0.0-20170811194829-b4e1701a28ef",
)

go_repository(
    name = "com_github_klauspost_crc32",
    importpath = "github.com/klauspost/crc32",
    sum = "h1:KAZ1BW2TCmT6PRihDPpocIy1QTtsAsrx6TneU/4+CMg=",
    version = "v0.0.0-20161016154125-cb6bfca970f6",
)

go_repository(
    name = "com_github_klauspost_pgzip",
    importpath = "github.com/klauspost/pgzip",
    sum = "h1:3L+neHp83cTjegPdCiOxVOJtRIy7/8RldvMTsyPYH10=",
    version = "v1.0.2-0.20170402124221-0bf5dcad4ada",
)

go_repository(
    name = "com_github_lib_pq",
    importpath = "github.com/lib/pq",
    sum = "h1:X5PMW56eZitiTeO7tKzZxFCSpbFZJtkMMooicw2us9A=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_mattn_go_sqlite3",
    importpath = "github.com/mattn/go-sqlite3",
    sum = "h1:LDdKkqtYlom37fkvqs8rMPFKAMe8+SgjbwZ6ex1/A/Q=",
    version = "v1.11.0",
)

go_repository(
    name = "com_github_mattn_go_tty",
    importpath = "github.com/mattn/go-tty",
    sum = "h1:d8RFOZ2IiFtFWBcKEHAFYJcPTf0wY5q0exFNJZVWa1U=",
    version = "v0.0.0-20180907095812-13ff1204f104",
)

go_repository(
    name = "com_github_mschoch_smat",
    importpath = "github.com/mschoch/smat",
    sum = "h1:VeRdUYdCw49yizlSbMEn2SZ+gT+3IUKx8BqxyQdz+BY=",
    version = "v0.0.0-20160514031455-90eadee771ae",
)

go_repository(
    name = "com_github_oschwald_maxminddb_golang",
    importpath = "github.com/oschwald/maxminddb-golang",
    sum = "h1:KAJSjdHQ8Kv45nFIbtoLGrGWqHFajOIm7skTyz/+Dls=",
    version = "v1.6.0",
)

go_repository(
    name = "com_github_philhofer_fwd",
    importpath = "github.com/philhofer/fwd",
    sum = "h1:UbZqGr5Y38ApvM/V/jEljVxwocdweyH+vmYvRPBnbqQ=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_pkg_term",
    importpath = "github.com/pkg/term",
    sum = "h1:tFwafIEMf0B7NlcxV/zJ6leBIa81D3hgGSgsE5hCkOQ=",
    version = "v0.0.0-20180730021639-bffc007b7fd5",
)

go_repository(
    name = "com_github_retailnext_hllpp",
    importpath = "github.com/retailnext/hllpp",
    sum = "h1:RnWNS9Hlm8BIkjr6wx8li5abe0fr73jljLycdfemTp0=",
    version = "v1.0.1-0.20180308014038-101a6d2f8b52",
)

go_repository(
    name = "com_github_robertkrimen_otto",
    importpath = "github.com/robertkrimen/otto",
    sum = "h1:+6NUiITWwE5q1KO6SAfUX918c+Tab0+tGAM/mtdlUyA=",
    version = "v0.0.0-20191219234010-c382bd3c16ff",
)

go_repository(
    name = "com_github_segmentio_kafka_go",
    importpath = "github.com/segmentio/kafka-go",
    sum = "h1:HtCSf6B4gN/87yc5qTl7WsxPKQIIGXLPPM1bMCPOsoY=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_smartystreets_assertions",
    importpath = "github.com/smartystreets/assertions",
    sum = "h1:zE9ykElWQ6/NYmHa3jpm/yHnI4xSofP+UP6SpjHcSeM=",
    version = "v0.0.0-20180927180507-b2de0cb4f26d",
)

go_repository(
    name = "com_github_smartystreets_goconvey",
    importpath = "github.com/smartystreets/goconvey",
    sum = "h1:fv0U8FUIMPNf1L9lnHLvLhgicrIVChEkdzIKYqbNC9s=",
    version = "v1.6.4",
)

go_repository(
    name = "com_github_tinylib_msgp",
    importpath = "github.com/tinylib/msgp",
    sum = "h1:DfdQrzQa7Yh2es9SuLkixqxuXS2SxsdYn0KbdrOGWD8=",
    version = "v1.0.2",
)

go_repository(
    name = "com_github_willf_bitset",
    importpath = "github.com/willf/bitset",
    sum = "h1:ekJIKh6+YbUIVt9DfNbkR5d6aFcFTLDRyJNAACURBg8=",
    version = "v1.1.3",
)

go_repository(
    name = "com_github_xlab_treeprint",
    importpath = "github.com/xlab/treeprint",
    sum = "h1:YdYsPAZ2pC6Tow/nPZOPQ96O3hm/ToAkGsPLzedXERk=",
    version = "v0.0.0-20180616005107-d6fb6747feb6",
)

go_repository(
    name = "com_google_cloud_go_bigquery",
    importpath = "cloud.google.com/go/bigquery",
    sum = "h1:sAbMqjY1PEQKZBWfbu6Y6bsupJ9c4QdHnzg/VvYTLcE=",
    version = "v1.3.0",
)

go_repository(
    name = "com_google_cloud_go_bigtable",
    importpath = "cloud.google.com/go/bigtable",
    sum = "h1:F4cCmA4nuV84V5zYQ3MKY+M1Cw1avHDuf3S/LcZPA9c=",
    version = "v1.2.0",
)

go_repository(
    name = "com_google_cloud_go_datastore",
    importpath = "cloud.google.com/go/datastore",
    sum = "h1:Kt+gOPPp2LEPWp8CSfxhsM8ik9CcyE/gYu+0r+RnZvM=",
    version = "v1.0.0",
)

go_repository(
    name = "com_google_cloud_go_pubsub",
    importpath = "cloud.google.com/go/pubsub",
    sum = "h1:9/vpR43S4aJaROxqQHQ3nH9lfyKKV0dC3vOmnw8ebQQ=",
    version = "v1.1.0",
)

go_repository(
    name = "com_google_cloud_go_storage",
    importpath = "cloud.google.com/go/storage",
    sum = "h1:RPUcBvDeYgQFMfQu1eBMq6piD1SXmLH+vK3qjewZPus=",
    version = "v1.5.0",
)

go_repository(
    name = "in_gopkg_sourcemap_v1",
    importpath = "gopkg.in/sourcemap.v1",
    sum = "h1:inv58fC9f9J3TK2Y2R1NPntXEn3/wjWHkonhIUODNTI=",
    version = "v1.0.5",
)

go_repository(
    name = "io_rsc_binaryregexp",
    importpath = "rsc.io/binaryregexp",
    sum = "h1:HfqmD5MEmC0zvwBuF187nq9mdnXjXsSivRiXN7SmRkE=",
    version = "v0.2.0",
)

go_repository(
    name = "org_collectd",
    importpath = "collectd.org",
    sum = "h1:iNBHGw1VvPJxH2B6RiFWFZ+vsjo1lCdRszBeOuwGi00=",
    version = "v0.3.0",
)

go_repository(
    name = "org_gonum_v1_gonum",
    importpath = "gonum.org/v1/gonum",
    sum = "h1:DJy6UzXbahnGUf1ujUNkh/NEtK14qMo2nvlBPs4U5yw=",
    version = "v0.6.0",
)

go_repository(
    name = "org_gonum_v1_netlib",
    importpath = "gonum.org/v1/netlib",
    sum = "h1:OE9mWmgKkjJyEmDAAtGMPjXu+YNeGvK9VTSHY6+Qihc=",
    version = "v0.0.0-20190313105609-8cb42192e0e0",
)

go_repository(
    name = "org_gonum_v1_plot",
    importpath = "gonum.org/v1/plot",
    sum = "h1:Qh4dB5D/WpoUUp3lSod7qgoyEHbDGPUWjIbnqdqqe1k=",
    version = "v0.0.0-20190515093506-e2840ee46a6b",
)
