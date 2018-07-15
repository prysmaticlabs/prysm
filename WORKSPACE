load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.13.0/rules_go-0.13.0.tar.gz"],
    sha256 = "ba79c532ac400cefd1859cbc8a9829346aa69e3b99482cd5a54432092cbc3933",
)

http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.13.0/bazel-gazelle-0.13.0.tar.gz"],
    sha256 = "bc653d3e058964a5a26dcad02b6c72d7d63e6bb88d94704990b908a1445b8758",
)

load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

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
    # Last updated July 15, 2018
    commit = "fad71da72e539ff79183a5d548b105c73ce4969f",
)

go_repository(
    name = "com_github_urfave_cli",
    importpath = "github.com/urfave/cli",
    commit = "8e01ec4cd3e2d84ab2fe90d8210528ffbb06d8ff",
)

go_repository(
    name = "com_github_fjl_memsize",
    importpath = "github.com/fjl/memsize",
    commit = "ca190fb6ffbc076ff49197b7168a760f30182d2e",
)

go_repository(
    name = "com_github_sirupsen_logrus",
    importpath = "github.com/sirupsen/logrus",
    commit = "e54a77765aca7bbdd8e56c1c54f60579968b2dc9",
)

go_repository(
    name = "org_golang_x_sys",
    commit = "1b2967e3c290b7c545b3db0deeda16e9be4f98a2",
    importpath = "golang.org/x/sys",
)

go_repository(
    name = "org_golang_x_crypto",
    commit = "a49355c7e3f8fe157a85be2f77e6e269a0f89602",
    importpath = "golang.org/x/crypto",
)
