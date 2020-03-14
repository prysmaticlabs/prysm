load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

# Bolt DB is an archived project which is no longer maintained. As of go 1.14,
# the go compiler adds checkptr to all builds when using -race or -msan. Since
# bolt DB violates this check, we must disable checkptr for this library.
#
# See: https://golang.org/doc/go1.14
# See: https://github.com/etcd-io/bbolt/issues/187

go_library(
    name = "go_default_library",
    srcs = [
        "bolt_386.go",
        "bolt_amd64.go",
        "bolt_arm.go",
        "bolt_arm64.go",
        "bolt_linux.go",
        "bolt_openbsd.go",
        "bolt_ppc64.go",
        "bolt_ppc64le.go",
        "bolt_s390x.go",
        "bolt_unix.go",
        "bolt_unix_solaris.go",
        "bolt_windows.go",
        "boltsync_unix.go",
        "bucket.go",
        "cursor.go",
        "db.go",
        "doc.go",
        "errors.go",
        "freelist.go",
        "node.go",
        "page.go",
        "tx.go",
    ],
    gc_goopts = ["-d=checkptr=0"],  # Required due to unsafe pointer usage.
    importpath = "github.com/boltdb/bolt",
    visibility = ["//visibility:public"],
    deps = select({
        "@io_bazel_rules_go//go/platform:solaris": [
            "@org_golang_x_sys//unix:go_default_library",
        ],
        "//conditions:default": [],
    }),
)

go_test(
    name = "go_default_test",
    srcs = [
        "bucket_test.go",
        "cursor_test.go",
        "db_test.go",
        "freelist_test.go",
        "node_test.go",
        "page_test.go",
        "quick_test.go",
        "simulation_test.go",
        "tx_test.go",
    ],
    embed = [":go_default_library"],
)
