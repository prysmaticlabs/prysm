load("@io_bazel_rules_go//go:def.bzl", "go_context", "go_rule")
load(
    "@io_bazel_rules_go//go/private:providers.bzl",
    "GoLibrary",
    "INFERRED_PATH",
)
load(
    "@io_bazel_rules_go//go/private:mode.bzl",
    "LINKMODE_C_ARCHIVE",
)
load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_binary",
    "go_library",
)

# TODO: Confirm this is the correct template for main.go
main_tpl = """
// Generated file. DO NOT EDIT.

package main
import (
    "unsafe"
    target "%s"
)
// #include <stdint.h>
import "C"
//export LLVMFuzzerTestOneInput
func LLVMFuzzerTestOneInput(data *C.char, size C.size_t) C.int {
	s := make([]byte, size)
	copy(s, (*[1 << 30]byte)(unsafe.Pointer(data))[:size:size])
	target.%s(s)
	return 0
}
func main() {
}
"""

# This isn't working, using --define=gotags=libfuzzer for now
# See: https://docs.google.com/document/d/1vc8v-kXjvgZOdQdnxPTaV0rrLxtP2XwnD2tAZlYJOqw/edit#
def _impl(settings, attr):
    return {"//:gotags": "libfuzzer"}

cfg_libfuzz_gotag = transition(
    implementation = _impl,
    inputs = [],
    outputs = ["//:gotags"],
)

# Notes:
# I think the best approach here is to write a macro that generates two targets
# - A go_binary with linkmode being c-archive
# - A wrapper rule with the transition to enforce the proper go tags

def _gen_fuzz_main_impl(ctx):
    if ctx.var.get("gotags") != "libfuzzer":
        fail("Gotags must be set to libfuzzer. Use --config=fuzz or --config=fuzzit until the transition rule is fixed.")

    pkg = ctx.attr.target_pkg
    func = ctx.attr.func

    ctx.actions.write(ctx.outputs.out, main_tpl % (pkg, func))

gen_fuzz_main = rule(
    implementation = _gen_fuzz_main_impl,
    attrs = {
        "target_pkg": attr.string(mandatory = True),
        "func": attr.string(mandatory = True),
        "_whitelist_function_transition": attr.label(default = "@bazel_tools//tools/whitelists/function_transition_whitelist"),
    },
    outputs = {"out": "generated_main.fuzz.go"},
    cfg = cfg_libfuzz_gotag,
)

# TODO: Add a outgoing (incoming?) transition rule for the go_library to apply the correct gotags.
def go_fuzz_library(
        name,
        #        corpus,
        func = "Fuzz",
        repository = "",
        size = "medium",
        tags = [],
        **kwargs):
    go_library(
        name = name + "_lib_with_fuzzer",
        tags = ["manual"] + tags,
        visibility = ["//visibility:private"],
        **kwargs
    )
    gen_fuzz_main(
        name = name + "_libfuzz_main",
        target_pkg = "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks",  # this needs to be inferred from the go library above.
        func = func,
        tags = ["manual"] + tags,
        visibility = ["//visibility:private"],
    )
    go_binary(
        name = name + "_binary",
        srcs = [name + "_libfuzz_main"],
        deps = [name + "_lib_with_fuzzer"],
        linkmode = LINKMODE_C_ARCHIVE,
        cgo = True,
        tags = ["manual"] + tags,
        visibility = ["//visibility:private"],
        gc_goopts = ["-d=libfuzzer"],
    )
    native.genrule(
        name = name,
        outs = [name + ".a"],
        srcs = [name + "_binary"],
        cmd = "cp $< $@",
        visibility = kwargs.get("visibility"),
        tags = ["manual"] + tags,
    )

    #    if not (corpus.startswith("//") or corpus.startswith(":") or corpus.startswith("@")):
    #        corpus_name = name + "_corpus"
    #        corpus = native.glob([corpus + "/**"])
    #        native.filegroup(
    #            name = corpus_name,
    #            srcs = corpus,
    #        )
    #    else:
    #        corpus_name = corpus

    native.cc_test(
        name = name + "_with_libfuzzer",
        linkopts = ["-fsanitize=fuzzer"],
        linkstatic = 1,
        testonly = 1,
        srcs = [":" + name],
        deps = ["@herumi_bls_eth_go_binary//:lib"],
        tags = ["manual", "fuzzer"] + tags,
    )
