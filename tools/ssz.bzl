load(
    "@io_bazel_rules_go//go:def.bzl",
    "GoLibrary",
)

def _ssz_go_proto_library_impl(ctx):
    go_proto = ctx.attr.go_proto

    generated_pb_go_files = go_proto[OutputGroupInfo].go_generated_srcs

    # Run the tool on the generated files
    package_path = generated_pb_go_files.to_list()[0].dirname
    output = ctx.outputs.out
    args = [
        "--output=%s" % output.path,
        "--path=%s" % package_path,
    ]
    if len(ctx.attr.objs) > 0:
        args += ["--objs=%s" % ",".join(ctx.attr.objs)]
    ctx.actions.run(
        executable = ctx.executable.sszgen,
        progress_message = "Generating ssz marshal and unmarshal functions",
        inputs = generated_pb_go_files,
        arguments = args,
        outputs = [output],
    )

"""
A rule that extends a go_proto_library rule with generated ssz marshal functions.
TODO: Update this documentation before merge.
"""
ssz_gen_marshal = rule(
    implementation = _ssz_go_proto_library_impl,
    attrs = {
        "go_proto": attr.label(providers = [GoLibrary]),
        "sszgen": attr.label(
            default = Label("@com_github_ferranbt_fastssz//sszgen:sszgen"),
            executable = True,
            cfg = "host",
        ),
        "objs": attr.string_list(),
    },
    outputs = {"out": "generated.ssz.go"},
)

SSZ_DEPS = ["@com_github_ferranbt_fastssz//:go_default_library"]
