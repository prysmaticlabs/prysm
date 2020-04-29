load("@io_bazel_rules_go//go:def.bzl", _go_library = "go_library")

def go_library(name, **kwargs):
    gc_goopts = []

    # if kwargs["gc_goopts"], go_goopts+=kwargs["gc_goopts"]

    gc_goopts += select({
        "@prysm//tools/go:libfuzz_enabled": ["-d=libfuzzer"],
        "//conditions:default": [],
    })

    kwargs["gc_goopts"] = gc_goopts
    _go_library(name = name, **kwargs)
