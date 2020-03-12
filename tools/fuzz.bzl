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

# Notes.
# Dockerfile
"""
FROM golang:1.14.0

# install deps
RUN apt update
RUN apt install -y clang

# install new fuzz tooling
RUN go get -u github.com/mdempsky/go114-fuzz-build

# install go-fuzz-corpus
RUN go get -v github.com/dvyukov/go-fuzz-corpus/png

# the "$GOPATH" is "/go"
WORKDIR /go/src/github.com/dvyukov/go-fuzz-corpus/png

# build the fuzzing case
RUN go114-fuzz-build -o png.a .
RUN clang -fsanitize=fuzzer png.a -o png

# start fuzzing
CMD ["./png"]
"""

# This isn't working, using --define=gotags=libfuzzer for now
def _impl(settings, attr):
    return {"//:libfuzz_gotags": "libfuzz"}

cfg_libfuzz_gotag = transition(
    implementation = _impl,
    inputs = [],
    outputs = ["//:libfuzz_gotags"],
)

# Notes:
# I think the best approach here is to write a macro that generates two targets
# - A go_binary with linkmode being c-archive
# - A wrapper rule with the transition to enforce the proper go tags

def _gen_fuzz_main_impl(ctx):
    if ctx.var.get("gotags") != "libfuzzer":
        fail("Gotags must be set to libfuzzer. Use --define=gotags=libfuzzer until the transition rule is fixed.")

    pkg = ctx.attr.target_pkg
    func = "Fuzz"
    ctx.actions.write(ctx.outputs.out, main_tpl % (pkg, func))

gen_fuzz_main = rule(
    implementation = _gen_fuzz_main_impl,
    attrs = {
        "target_pkg": attr.string(mandatory = True),
    },
    outputs = {"out": "generated_main.fuzz.go"},
)

def go_fuzz_library(name, **kwargs):
    # TODO: Add a outgoing transition rule for the go_library to apply the correct gotags.
    go_library(
        name = name + "_lib_with_fuzzer",
        tags = ["manual"],  # TODO: Add tags from kwargs?
        **kwargs
    )
    gen_fuzz_main(
        name = name + "_libfuzz_main",
        target_pkg = "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks",  # this needs to be inferred from the go library above.
        tags = ["manual"],
    )
    go_binary(
        name = name + "_binary",
        srcs = [name + "_libfuzz_main"],
        deps = [name + "_lib_with_fuzzer"],
        linkmode = LINKMODE_C_ARCHIVE,
        cgo = True,
        tags = ["manual"],
    )
    native.genrule(
        name = name,
        outs = [name + ".a"],
        srcs = [name + "_binary"],
        cmd = "cp $< $@",
    )

#def _go_fuzz_library_impl(ctx):
#    if ctx.var["gotags"] != "libfuzzer":
#        fail("Gotags must be set to libfuzzer. Use --define=gotags=libfuzzer until the transition rule is fixed.")
#    if ctx.attr.linkmode != LINKMODE_C_ARCHIVE:
#        fail("Link mode must be c-archive")
#
#    go = go_context(ctx)
#    if go.pathtype == INFERRED_PATH:
#        fail("importpath must be specified in this library or one of its embedded libraries")
#    library = go.new_library(go)
#
#    # Need to set these build flags.
#    #    	buildFlags := []string{
#    #    		"-buildmode", "c-archive",
#    #    		"-gcflags", "all=-d=libfuzzer",
#    #    		"-tags", tags,
#    #    		"-trimpath",
#    #    	}
#
#    source = go.library_to_source(go, ctx.attr, library, ctx.coverage_instrumented())
#    archive = go.archive(go, source)
#
#    return [
#        library,
#        source,
#        archive,
#        DefaultInfo(
#            files = depset([archive.data.file]),
#        ),
#        OutputGroupInfo(
#            cgo_exports = archive.cgo_exports,
#            compilation_outputs = [archive.data.file],
#        ),
#    ]

#go_fuzz_library = go_rule(
#    implementation = _go_fuzz_library_impl,
#    attrs = {
#        "srcs": attr.label_list(allow_files = True),
#        "deps": attr.label_list(providers = [GoLibrary]),
#        "embed": attr.label_list(providers = [GoLibrary]),
#        "fuzz_func": attr.string(default = "Fuzz"),
#        "linkmode": attr.string(default = LINKMODE_C_ARCHIVE),
#        # Whitelist is required or bazel complains.
#        # "_whitelist_function_transition": attr.label(default = "@bazel_tools//tools/whitelists/function_transition_whitelist"),
#    },
#    #cfg = cfg_libfuzz_gotag,
#)
