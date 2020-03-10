load(
    "@io_bazel_rules_go//go:def.bzl",
    "GoArchive",
    "GoLibrary",
    "GoSource",
)
load(
    "@io_bazel_rules_go//go/private:context.bzl",
    "go_context",
)
load(
    "@io_bazel_rules_go//go/private:rules/rule.bzl",
    "go_rule",
)

def _ssz_go_proto_library_impl(ctx):
    go_proto = ctx.attr.go_proto
    go = go_context(ctx)

    generated_pb_go_files = go_proto[OutputGroupInfo].go_generated_srcs

    # Run the tool on the generated files
    package_path = generated_pb_go_files.to_list()[0].dirname

    # TODO: name = go_proto's name
    output = go.declare_file(go = go, name = "v1_go_proto", ext = ".ssz.go", path = package_path)
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

    # Update providers, maybe recompile the go archive?
    library = go_proto[GoLibrary]
    source = go_proto[GoSource]
    f = {
        # TODO: Gross!
        "library": source.library,
        "mode": source.mode,
        "srcs": source.srcs + [output],  # Add generated file.
        "orig_srcs": source.srcs,
        "orig_src_map": source.orig_src_map,
        "cover": source.cover,
        "x_defs": source.x_defs,
        "deps": source.deps + ctx.attr._deps,  # Add generated file's deps.
        "gc_goopts": source.gc_goopts,
        "runfiles": source.runfiles,
        "cgo": source.cgo,
        "cdeps": source.cdeps,
        "cppopts": source.cppopts,
        "copts": source.copts,
        "cxxopts": source.cxxopts,
        "clinkopts": source.clinkopts,
        "cgo_deps": source.cgo_deps,
        "cgo_exports": source.cgo_exports,
    }
    foo = GoSource(**f)
    archive = go.archive(go, foo)

    return [
        library,
        source,
        archive,
        DefaultInfo(
            files = depset([archive.data.file]),
        ),
        OutputGroupInfo(
            cgo_exports = archive.cgo_exports,
            compilation_outputs = [archive.data.file],
        ),
    ]

ssz_go_proto_library = go_rule(
    implementation = _ssz_go_proto_library_impl,
    attrs = {
        "go_proto": attr.label(providers = [GoLibrary]),
        "sszgen": attr.label(
            default = Label("@com_github_ferranbt_fastssz//sszgen:sszgen"),
            executable = True,
            cfg = "host",
        ),
        "objs": attr.string_list(),
        "_deps": attr.label_list(
            default = [Label("@com_github_ferranbt_fastssz//:go_default_library")],
        ),
    },
)
