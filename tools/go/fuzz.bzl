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

def _gen_fuzz_main_impl(ctx):
    if ctx.var.get("gotags") != "libfuzzer":
        fail("gotags must be set to libfuzzer. Use --config=fuzz or --config=fuzzit.")
    if "libfuzzer" not in ctx.var.get("gc_goopts"):
        fail("gc_goopts must be set to -d=libfuzzer. Use --config=fuzz or --config=fuzzit.")

    pkg = ctx.attr.target_pkg
    func = ctx.attr.func

    output_file_name = ctx.label.name + "_main.fuzz.go"
    output_file = ctx.actions.declare_file(output_file_name)
    ctx.actions.write(output_file, main_tpl % (pkg, func))
    return [DefaultInfo(files = depset([output_file]))]

gen_fuzz_main = rule(
    implementation = _gen_fuzz_main_impl,
    attrs = {
        "target_pkg": attr.string(mandatory = True),
        "func": attr.string(mandatory = True),
    },
)

fuzzer_options_tpl = """[libfuzzer]
max_len=%d
"""

def _generate_libfuzzer_config(ctx):
    output_file_name = ctx.label.name + ".options"
    output = fuzzer_options_tpl % (
        ctx.attr.max_len,
    )
    output_file = ctx.actions.declare_file(output_file_name)
    ctx.actions.write(output_file, output)
    return [DefaultInfo(files = depset([output_file]))]

gen_libfuzzer_config = rule(
    implementation = _generate_libfuzzer_config,
    attrs = {
        "max_len": attr.int(default = 0),
    },
)

def _upload_to_gcp_impl(ctx):
    return [
        DefaultInfo(),
    ]

upload_to_gcp = rule(
    implementation = _upload_to_gcp_impl,
    attrs = {
        "gcp_bucket": attr.string(mandatory = True),
        "libfuzzer_bundle": attr.label(mandatory = True),
        "afl_bundle": attr.label(mandatory = True),
    },
)

def go_fuzz_test(
        name,
        corpus,
        corpus_path,
        importpath,
        func = "Fuzz",
        repository = "",
        max_len = 0,
        gcp_bucket = "gs://builds.prysmaticlabs.appspot.com",
        size = "medium",
        tags = [],
        **kwargs):
    go_library(
        name = name + "_lib_with_fuzzer",
        tags = ["manual"] + tags,
        visibility = ["//visibility:private"],
        testonly = 1,
        importpath = importpath,
        gc_goopts = ["-d=libfuzzer"],
        **kwargs
    )
    gen_fuzz_main(
        name = name + "_libfuzz_main",
        target_pkg = importpath,
        func = func,
        tags = ["manual"] + tags,
        testonly = 1,
        visibility = ["//visibility:private"],
    )
    gen_libfuzzer_config(
        name = name + "_options",
        max_len = max_len,
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
        testonly = 1,
    )
    native.genrule(
        name = name,
        outs = [name + ".a"],
        srcs = [":" + name + "_binary"],
        cmd = "cp $< $@",
        visibility = kwargs.get("visibility"),
        tags = ["manual"] + tags,
        testonly = 1,
    )

    if not (corpus.startswith("//") or corpus.startswith(":") or corpus.startswith("@")):
        corpus_name = name + "_corpus"
        corpus = native.glob([corpus + "/**"])
        native.filegroup(
            name = corpus_name,
            srcs = corpus,
        )
    else:
        corpus_name = corpus

    additional_args = []
    if max_len > 0:
        additional_args += ["-max_len=%s" % max_len]

    native.cc_test(
        name = name + "_with_afl",
        linkopts = [
            "-fsanitize=address",
            "-fsanitize-coverage=trace-pc-guard",
        ],
        linkstatic = 1,
        testonly = 1,
        srcs = [":" + name],
        deps = [
            "@herumi_bls_eth_go_binary//:lib",
            "//third_party/afl:fuzzing_engine",
        ],
        tags = ["manual", "fuzzer"] + tags,
    )

    native.genrule(
        name = name + "_afl_bundle",
        outs = [name + "_afl_bundle.zip"],
        srcs = [
            "//third_party/afl:libs",
            ":" + name + "_with_afl",
        ],
        cmd = "cp $(location :" + name + "_with_afl) fuzzer; $(location @bazel_tools//tools/zip:zipper) cf $@ $(locations //third_party/afl:libs) fuzzer",
        tools = [
            "@bazel_tools//tools/zip:zipper",
        ],
        testonly = 1,
        tags = ["manual"] + tags,
    )

    native.cc_test(
        name = name + "_with_libfuzzer",
        linkopts = ["-fsanitize=fuzzer,address"],
        copts = ["-fsanitize=fuzzer,address"],
        linkstatic = 1,
        testonly = 1,
        srcs = [":" + name],
        deps = ["@herumi_bls_eth_go_binary//:lib"],
        tags = ["manual", "fuzzer"] + tags,
        args = [
            corpus_path,
            "-print_final_stats=1",
            "-use_value_profile=1",
            "-max_total_time=3540",  # One minute early of 3600.
        ] + additional_args,
        data = [corpus_name],
        timeout = "eternal",
    )

    native.genrule(
        name = name + "_libfuzzer_bundle",
        outs = [name + "_libfuzzer_bundle.zip"],
        srcs = [
            ":" + name + "_with_libfuzzer",
            ":" + name + "_options",
        ],
        cmd = "cp $(location :" + name + "_with_libfuzzer) fuzzer; " +
              "cp $(location :" + name + "_options) fuzzer.options; " +
              "$(location @bazel_tools//tools/zip:zipper) cf $@ fuzzer fuzzer.options",
        tools = ["@bazel_tools//tools/zip:zipper"],
        testonly = 1,
        tags = ["manual"] + tags,
    )

    upload_to_gcp(
        name = name + "_uploader",
        gcp_bucket = gcp_bucket,
        afl_bundle = ":" + name + "_afl_bundle",
        libfuzzer_bundle = ":" + name + "_libfuzzer_bundle",
        tags = ["manual"] + tags,
    )
