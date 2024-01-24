load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_file")
load("@distroless//private/remote:debian_archive.bzl", "debian_archive")

def prysm_image_deps():
    """
    These dependencies are pulled from https://debian.pkgs.org and support Debian 11.
    """
    http_file(
        name = "bash_amd64",
        sha256 = "f702ef058e762d7208a9c83f6f6bbf02645533bfd615c54e8cdcce842cd57377",
        urls = [
            "http://ftp.us.debian.org/debian/pool/main/b/bash/bash_5.1-2+deb11u1_amd64.deb",
            "http://http.us.debian.org/debian/pool/main/b/bash/bash_5.1-2+deb11u1_amd64.deb",
            "http://ftp.uk.debian.org/debian/pool/main/b/bash/bash_5.1-2+deb11u1_amd64.deb",
            "http://ftp.au.debian.org/debian/pool/main/b/bash/bash_5.1-2+deb11u1_amd64.deb",
            "https://prysmaticlabs.com/uploads/bash_5.1-2+deb11u1_amd64.deb",
        ],
    )

    http_file(
        name = "bash_arm64",
        sha256 = "d7c7af5d86f43a885069408a89788f67f248e8124c682bb73936f33874e0611b",
        urls = [
            "http://ftp.us.debian.org/debian/pool/main/b/bash/bash_5.1-2+deb11u1_arm64.deb",
            "http://http.us.debian.org/debian/pool/main/b/bash/bash_5.1-2+deb11u1_arm64.deb",
            "http://ftp.uk.debian.org/debian/pool/main/b/bash/bash_5.1-2+deb11u1_arm64.deb",
            "http://ftp.au.debian.org/debian/pool/main/b/bash/bash_5.1-2+deb11u1_arm64.deb",
            "https://prysmaticlabs.com/uploads/bash_5.1-2+deb11u1_arm64.deb",
        ],
    )

    http_file(
        name = "libtinfo6_amd64",
        sha256 = "96ed58b8fd656521e08549c763cd18da6cff1b7801a3a22f29678701a95d7e7b",
        urls = [
            "http://ftp.us.debian.org/debian/pool/main/n/ncurses/libtinfo6_6.2+20201114-2+deb11u2_amd64.deb",
            "http://http.us.debian.org/debian/pool/main/n/ncurses/libtinfo6_6.2+20201114-2+deb11u2_amd64.deb",
            "http://ftp.uk.debian.org/debian/pool/main/n/ncurses/libtinfo6_6.2+20201114-2+deb11u2_amd64.deb",
            "http://ftp.au.debian.org/debian/pool/main/n/ncurses/libtinfo6_6.2+20201114-2+deb11u2_amd64.deb",
            "https://prysmaticlabs.com/uploads/libtinfo6_6.2+20201114-2+deb11u2_amd64.deb",
        ],
    )

    http_file(
        name = "libtinfo6_arm64",
        sha256 = "58027c991756930a2abb2f87a829393d3fdbfb76f4eca9795ef38ea2b0510e27",
        urls = [
            "http://ftp.us.debian.org/debian/pool/main/n/ncurses/libtinfo6_6.2+20201114-2+deb11u1_arm64.deb",
            "http://http.us.debian.org/debian/pool/main/n/ncurses/libtinfo6_6.2+20201114-2+deb11u1_arm64.deb",
            "http://ftp.uk.debian.org/debian/pool/main/n/ncurses/libtinfo6_6.2+20201114-2+deb11u1_arm64.deb",
            "http://ftp.au.debian.org/debian/pool/main/n/ncurses/libtinfo6_6.2+20201114-2+deb11u1_arm64.deb",
            "https://prysmaticlabs.com/uploads/libtinfo6_6.2+20201114-2+deb11u2_arm64.deb",
        ],
    )

    http_file(
        name = "coreutils_amd64",
        sha256 = "3558a412ab51eee4b60641327cb145bb91415f127769823b68f9335585b308d4",
        urls = [
            "http://ftp.us.debian.org/debian/pool/main/c/coreutils/coreutils_8.32-4+b1_amd64.deb",
            "http://http.us.debian.org/debian/pool/main/c/coreutils/coreutils_8.32-4+b1_amd64.deb",
            "http://ftp.uk.debian.org/debian/pool/main/c/coreutils/coreutils_8.32-4+b1_amd64.deb",
            "http://ftp.au.debian.org/debian/pool/main/c/coreutils/coreutils_8.32-4+b1_amd64.deb",
            "https://prysmaticlabs.com/uploads/coreutils_8.32-4+b1_amd64.deb",
        ],
    )

    http_file(
        name = "coreutils_arm64",
        sha256 = "6210c84d6ff84b867dc430f661f22f536e1704c27bdb79de38e26f75b853d9c0",
        urls = [
            "http://ftp.us.debian.org/debian/pool/main/c/coreutils/coreutils_8.32-4_arm64.deb",
            "http://http.us.debian.org/debian/pool/main/c/coreutils/coreutils_8.32-4_arm64.deb",
            "http://ftp.uk.debian.org/debian/pool/main/c/coreutils/coreutils_8.32-4_arm64.deb",
            "http://ftp.au.debian.org/debian/pool/main/c/coreutils/coreutils_8.32-4_arm64.deb",
            "https://prysmaticlabs.com/uploads/coreutils_8.32-4_arm64.deb",
        ],
    )

    debian_archive(
        name = "amd64_debian11_coreutils",
        package_name = "coreutils",
        sha256 = "3558a412ab51eee4b60641327cb145bb91415f127769823b68f9335585b308d4",
        urls = [
            "https://snapshot.debian.org/archive/debian/20231214T085654Z/pool/main/c/coreutils/coreutils_8.32-4+b1_amd64.deb",
            "https://prysmaticlabs.com/uploads/coreutils_8.32-4+b1_amd64.deb",
        ],
    )

    debian_archive(
        name = "arm64_debian11_coreutils",
        package_name = "coreutils",
        sha256 = "6210c84d6ff84b867dc430f661f22f536e1704c27bdb79de38e26f75b853d9c0",
        urls = [
            "https://snapshot.debian.org/archive/debian/20231214T085654Z/pool/main/c/coreutils/coreutils_8.32-4_arm64.deb",
            "https://prysmaticlabs.com/uploads/coreutils_8.32-4_arm64.deb",
        ],
    )
