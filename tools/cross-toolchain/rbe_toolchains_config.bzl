load("@bazel_toolchains//rules:rbe_repo.bzl", "rbe_autoconfig")
load("@prysm//tools/cross-toolchain:configs/versions.bzl", _generated_toolchain_config_suite_autogen_spec = "TOOLCHAIN_CONFIG_AUTOGEN_SPEC")

_PRYSM_BUILD_IMAGE_REGISTRY = "gcr.io"
_PRYSM_BUILD_IMAGE_REPOSITORY = "prysmaticlabs/rbe-worker"
_PRYSM_BUILD_IMAGE_DIGEST = "sha256:ebf03ab43a88499e70c8bb4dad70b8bdbc9f9923a87bc72c39e9e7690fcd49f2"
_PRYSM_BUILD_IMAGE_JAVA_HOME = "/usr/lib/jvm/java-8-openjdk-amd64"
_CONFIGS_OUTPUT_BASE = "tools/cross-toolchain/configs"

_CLANG_ENV = {
    "BAZEL_COMPILER": "clang",
    "BAZEL_LINKLIBS": "-l%:libstdc++.a",
    "BAZEL_LINKOPTS": "-lm:-static-libgcc",
    "BAZEL_USE_LLVM_NATIVE_COVERAGE": "1",
    "GCOV": "llvm-profdata",
    "CC": "clang",
    "CXX": "clang++",
}

_GCC_ENV = {
    "BAZEL_COMPILER": "gcc",
    "BAZEL_LINKLIBS": "-l%:libstdc++.a",
    "BAZEL_LINKOPTS": "-lm:-static-libgcc",
    "CC": "gcc",
    "CXX": "g++",
}

_TOOLCHAIN_CONFIG_SUITE_SPEC = {
    "container_registry": _PRYSM_BUILD_IMAGE_REGISTRY,
    "container_repo": _PRYSM_BUILD_IMAGE_REPOSITORY,
    "output_base": _CONFIGS_OUTPUT_BASE,
    "repo_name": "prysm",
    "toolchain_config_suite_autogen_spec": _generated_toolchain_config_suite_autogen_spec,
}

def _rbe_toolchains_generator():
    rbe_autoconfig(
        name = "rbe_ubuntu_clang_gen",
        digest = _PRYSM_BUILD_IMAGE_DIGEST,
        export_configs = True,
        java_home = _PRYSM_BUILD_IMAGE_JAVA_HOME,
        registry = _PRYSM_BUILD_IMAGE_REGISTRY,
        repository = _PRYSM_BUILD_IMAGE_REPOSITORY,
        env = _CLANG_ENV,
        toolchain_config_spec_name = "clang",
        toolchain_config_suite_spec = _TOOLCHAIN_CONFIG_SUITE_SPEC,
        use_checked_in_confs = "False",
        config_repos = [
            "prysm_toolchains",
        ],
        use_legacy_platform_definition = False,
        exec_compatible_with = [
            "@bazel_tools//platforms:x86_64",
            "@bazel_tools//platforms:linux",
            "@bazel_tools//tools/cpp:clang",
        ],
    )

    rbe_autoconfig(
        name = "rbe_ubuntu_gcc_gen",
        digest = _PRYSM_BUILD_IMAGE_DIGEST,
        export_configs = True,
        java_home = _PRYSM_BUILD_IMAGE_JAVA_HOME,
        registry = _PRYSM_BUILD_IMAGE_REGISTRY,
        repository = _PRYSM_BUILD_IMAGE_REPOSITORY,
        env = _GCC_ENV,
        toolchain_config_spec_name = "gcc",
        toolchain_config_suite_spec = _TOOLCHAIN_CONFIG_SUITE_SPEC,
        use_checked_in_confs = "False",
        config_repos = [
            "prysm_toolchains",
        ],
        use_legacy_platform_definition = False,
        exec_compatible_with = [
            "@bazel_tools//platforms:x86_64",
            "@bazel_tools//platforms:linux",
            "@bazel_tools//tools/cpp:gcc",
        ],
    )

def _generated_rbe_toolchains():
    rbe_autoconfig(
        name = "rbe_ubuntu_clang",
        digest = _PRYSM_BUILD_IMAGE_DIGEST,
        export_configs = True,
        java_home = _PRYSM_BUILD_IMAGE_JAVA_HOME,
        registry = _PRYSM_BUILD_IMAGE_REGISTRY,
        repository = _PRYSM_BUILD_IMAGE_REPOSITORY,
        toolchain_config_spec_name = "clang",
        toolchain_config_suite_spec = _TOOLCHAIN_CONFIG_SUITE_SPEC,
        use_checked_in_confs = "Force",
        use_legacy_platform_definition = False,
        exec_compatible_with = [
            "@bazel_tools//platforms:x86_64",
            "@bazel_tools//platforms:linux",
            "@bazel_tools//tools/cpp:clang",
        ],
    )

def rbe_toolchains_config():
    _rbe_toolchains_generator()
    _generated_rbe_toolchains()
