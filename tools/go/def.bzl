load("@io_bazel_rules_go//go/private/rules:library.bzl", _go_library = "go_library")
load("@io_bazel_rules_go//go/private/rules:test.bzl", "go_test_kwargs")
load("@bazel_gazelle//:deps.bzl", _go_repository = "go_repository")

def _go_test_transition_impl(settings, attr):
    """Edge transition to add minimal or mainnet build tags"""
    settings = dict(settings)

    if attr.eth_network == "minimal":
        settings["//proto:network"] = "minimal"
        settings["@io_bazel_rules_go//go/config:tags"] += ["minimal"]
    elif attr.eth_network == "mainnet":  # Default / optional
        settings["//proto:network"] = "mainnet"
        settings["@io_bazel_rules_go//go/config:tags"] += ["mainnet"]

    if attr.gotags:
        settings["@io_bazel_rules_go//go/config:tags"] += attr.gotags

    if str(settings["//command_line_option:compilation_mode"]) == "dbg":
        settings["@io_bazel_rules_go//go/config:debug"] = True
    return settings

go_test_transition = transition(
    implementation = _go_test_transition_impl,
    inputs = [
        "@io_bazel_rules_go//go/config:tags",
        "//proto:network",
        "//command_line_option:compilation_mode",
        "@io_bazel_rules_go//go/config:debug",
    ],
    outputs = [
        "@io_bazel_rules_go//go/config:tags",
        "//proto:network",
        "//command_line_option:compilation_mode",
        "@io_bazel_rules_go//go/config:debug",
    ],
)

def _go_test_transition_rule(**kwargs):
    """A wrapper around go_test to add an eth_network attribute and incoming edge transition to support compile time configuration"""
    kwargs = dict(kwargs)
    attrs = dict(kwargs["attrs"])
    attrs.update({
        "eth_network": attr.string(values = ["mainnet", "minimal"]),
        "_whitelist_function_transition": attr.label(
            default = "@bazel_tools//tools/whitelists/function_transition_whitelist",
        ),
    })
    kwargs["attrs"] = attrs
    kwargs["cfg"] = go_test_transition
    return rule(**kwargs)

go_test = _go_test_transition_rule(**go_test_kwargs)

def go_library(name, **kwargs):
    gc_goopts = []

    if "gc_goopts" in kwargs:
        go_goopts = kwargs["gc_goopts"]

    gc_goopts += select({
        "@prysm//tools/go:libfuzz_enabled": ["-d=libfuzzer,checkptr"],
        "//conditions:default": [],
    })

    kwargs["gc_goopts"] = gc_goopts
    _go_library(name = name, **kwargs)

# Maybe download a repository rule, if it doesn't exist already.
def maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)

# A wrapper around go_repository to add gazelle directives.
def go_repository(name, **kwargs):
    # Some third party go tools may be used by the fuzzing pipeline to generate code. This causes
    # an issue when running with --config=fuzz and is not necessary since the dependency is not
    # part of the final binary.
    if "nofuzz" in kwargs:
        kwargs.pop("nofuzz", None)
        return maybe(_go_repository, name, **kwargs)

    directives = []
    if "build_directives" in kwargs:
        directives = kwargs["build_directives"]

    directives += [
        "gazelle:map_kind go_library go_library @prysm//tools/go:def.bzl",
    ]
    kwargs["build_directives"] = directives
    maybe(_go_repository, name, **kwargs)
