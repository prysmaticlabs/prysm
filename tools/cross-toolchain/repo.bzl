load(":versions.bzl", "TOOLCHAIN_CONFIG_AUTOGEN_SPEC")

CUSTOM_TOOLCHAIN_CONFIG_SUITE_SPEC = {
    "repo_name": "prysm",
    "output_base": "tools",
    "container_repo": "prysmaticlabs/rbe-worker",
    "container_registry": "gcr.io", 
    "toolchain_config_suite_autogen_spec": TOOLCHAIN_CONFIG_AUTOGEN_SPEC,
}