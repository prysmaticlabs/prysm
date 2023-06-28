load("@rules_oci//oci:defs.bzl", "oci_image", "oci_image_index", "oci_push")
load("@rules_pkg//:pkg.bzl", "pkg_tar")
load("//tools:multi_arch.bzl", "multi_arch")
load("@rules_multirun//:defs.bzl", "command", "multirun")


def prysm_image_upload(
    name,
    binary,
    entrypoint,
    symlinks,
    repositories):

    pkg_tar(
        name = "binary_tar",
        srcs = [binary],
        symlinks=symlinks,
    )

    oci_image(
        name = "oci_image",
        base = "@linux_debian11_multiarch_base",
        entrypoint = entrypoint,
        tars = [
            "//tools:passwd_tar",
            "//tools:libtinfo6_tar",
            "//tools:bash_tar",
            ":binary_tar",
        ],
    )

    multi_arch(
        name = "oci_multiarch",
        image = ":oci_image",
        platforms = [
            "@io_bazel_rules_go//go/toolchain:linux_amd64_cgo",
            "@io_bazel_rules_go//go/toolchain:linux_arm64_cgo",
        ],
    )

    oci_image_index(
        name = "oci_image_index",
        images = [
            ":oci_multiarch",
        ],
    )

    [
        oci_push(
            name = "push_{}".format(i),
            image = ":oci_image_index",
            repository = repo,
        )
        for i, repo in enumerate(repositories)
    ]

    [
        command(
            name = "cmd_{}".format(i),
            command = ":push_{}".format(i),
            arguments = ["--tag", "{DOCKER_TAG}"],
        )
        for i in range(len(repositories))
    ]

    multirun(
        name = name,
        commands = [
        "cmd_{}".format(i)
            for i in range(len(repositories))
        ],
        #jobs = 0, # Set to 0 to run in parallel, defaults to sequential
    )