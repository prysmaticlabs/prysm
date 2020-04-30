load("@io_bazel_rules_go//go:def.bzl", _go_library = "go_library")
load("@bazel_gazelle//:deps.bzl", _go_repository = "go_repository")

def go_library(name, **kwargs):
    gc_goopts = []

    if "gc_goopts" in kwargs:
        go_goopts = kwargs["gc_goopts"]

    gc_goopts += select({
        "@prysm//tools/go:libfuzz_enabled": ["-d=libfuzzer"],
        "//conditions:default": [],
    })

    kwargs["gc_goopts"] = gc_goopts
    _go_library(name = name, **kwargs)

# A wrapper around go_repository to add gazelle directives.
def go_repository(**kwargs):
    # Some third party go tools may be used by the fuzzing pipeline to generate code. This causes
    # an issue when running with --config=fuzz and is not necessary since the dependency is not
    # part of the final binary.
    if "nofuzz" in kwargs:
        kwargs.pop("nofuzz", None)
        return _go_repository(**kwargs)

    directives = []
    if "build_directives" in kwargs:
        directives = kwargs["build_directives"]

    directives += [
        "gazelle:map_kind go_library go_library @prysm//tools/go:def.bzl",
    ]
    kwargs["build_directives"] = directives
    _go_repository(**kwargs)
