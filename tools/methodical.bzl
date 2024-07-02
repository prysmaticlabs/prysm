"""
A rule that uses the generated pb.go files from a go_proto_library target to generate SSZ marshal
and unmarshal functions as pointer receivers on the specified objects. To use this rule, provide a
go_proto_library target and specify the structs to generate methods in the "objs" field. Lastly,
include your new target as a source for the go_library that embeds the go_proto_library.
Example:
go_proto_library(
  name = "example_go_proto",
   ...
)
ssz_gen_marshal(
  name = "ssz_generated_sources",
  go_proto = ":example_go_proto",
  objs = [ # omit this field to generate for all structs in the package.
    "AddressBook",
    "Person",
  ],
)
go_library(
  name = "go_default_library",
  srcs = [":ssz_generated_sources"],
  embed = [":example_go_proto"],
  deps = SSZ_DEPS,
)
"""

load(
    "@io_bazel_rules_go//go:def.bzl",
    "GoLibrary",
    "GoSource",
    "go_context",
)

load("@io_bazel_rules_go//go/tools/gopackagesdriver:aspect.bzl", "go_pkg_info_aspect", "GoPkgInfo")

"""
def _ssz_go_proto_library_impl(ctx):
    if ctx.var.get("ssz"):
        fail("--define=ssz=<value> is no longer supported, please use --//proto:network=<value>")

    if ctx.attr.go_proto != None:
        go_proto = ctx.attr.go_proto
        input_files = go_proto[OutputGroupInfo].go_generated_srcs.to_list()
        package_path = input_files[0].dirname
    elif hasattr(ctx.attr, "srcs") and len(ctx.attr.srcs) > 0:
        package_path = ctx.attr.srcs[0].files.to_list()[0].dirname
        input_files = ctx.attr.srcs[0].files.to_list()
    else:
        fail("Must have go_proto or srcs")

    # Run the tool.
    output = ctx.outputs.out
    args = [
        "--output=%s" % output.path,
        "--path=%s" % package_path,
    ]
    if hasattr(ctx.attr, "includes") and len(ctx.attr.includes) > 0:
        incs = []
        for include in ctx.attr.includes:
            incs.append(include[GoSource].srcs[0].dirname)
            input_files += include[GoSource].srcs
        args.append("--include=%s" % ",".join(incs))

    if len(ctx.attr.objs) > 0:
        args.append("--objs=%s" % ",".join(ctx.attr.objs))

    if len(ctx.attr.exclude_objs) > 0:
        args.append("--exclude-objs=%s" % ",".join(ctx.attr.exclude_objs))

    ctx.actions.run(
        executable = ctx.executable.sszgen,
        progress_message = "Generating ssz marshal and unmarshal functions",
        inputs = input_files,
        arguments = args,
        outputs = [output],
    )

ssz_gen_marshal = rule(
    implementation = _ssz_go_proto_library_impl,
    attrs = {
        "srcs": attr.label_list(allow_files = True),
        "go_proto": attr.label(providers = [GoLibrary]),
        "sszgen": attr.label(
            default = Label("@com_github_prysmaticlabs_fastssz//sszgen:sszgen"),
            executable = True,
            cfg = "exec",
        ),
        "objs": attr.string_list(),
        "exclude_objs": attr.string_list(),
        "includes": attr.label_list(providers = [GoLibrary]),
        "out": attr.output(),
    },
)

SSZ_DEPS = ["@com_github_prysmaticlabs_fastssz//:go_default_library"]
"""

def _ssz_methodical_impl(ctx):
    go_ctx = go_context(ctx)
    all_json_files = {}
    stdlib = ''
    inputs = []
    inputs += go_ctx.sdk.srcs
    sample = go_ctx.sdk.srcs[0].path
    # ctx.attr.deps should contain the GoPkgInfo from applying 'go_pkg_info_aspect'
    for dep in ctx.attr.deps:
        pkginfo = dep[OutputGroupInfo]
        # sources from the protobuf compiler
        #inputs = pkginfo.go_generated_srcs.to_list()
        # collect all the paths to json files dict keys for uniqueness
        #if hasattr(pkginfo, "go_pkg_driver_json_file"):
        json_files = pkginfo.go_pkg_driver_json_file.to_list()
        inputs += json_files
        if len(json_files) > 0:
            for jf in json_files:
                # presumably path is full path from exec root
                all_json_files[jf.path] = ""
        # we just ned to get the stdlib once
        #if stdlib == '' and hasattr(pkginfo, "go_pkg_driver_stdlib_json_file"):
        if stdlib == '':
            std_ds = pkginfo.go_pkg_driver_stdlib_json_file.to_list()
            if len(std_ds) > 0:
                stdlib = std_ds[0].path
                inputs += std_ds
        inputs += pkginfo.go_pkg_driver_srcs.to_list()
    # concat the stdlib with all the other json file paths and write to disk
    json_out = [stdlib] + all_json_files.keys()
    all_pkg_list = ctx.actions.declare_file("methodical-pkg-list.json")
    ctx.actions.write(all_pkg_list, content = json.encode(json_out))
    ctx.actions.run_shell(
        env = {
            "PACKAGEJSONINVENTORY": all_pkg_list.path,
        },
        inputs =  [all_pkg_list] + inputs,
        outputs = [ctx.outputs.out],
        command = """
        echo "sample = {sample}" &&
        echo "{out_base}" &&
        env &&
        echo {json_list} && 
        cat {json_list} &&
        echo $PACKAGEJSONINVENTORY &&
        echo "\n" &&
        exit 1
        """.format(
            sample = sample,
            out_base = ctx.outputs.out.root.path,
            json_list = all_pkg_list.path,
        ),
        #wc -l bazel-out/darwin_arm64-fastbuild/bin/proto/prysm/v1alpha1/methodical-pkg-list.json &&
    )

ssz_methodical = rule(
    implementation = _ssz_methodical_impl,
    attrs = {
        'deps' : attr.label_list(aspects = [go_pkg_info_aspect]),
        "out": attr.output(
            doc = "The new Go file to emit the generated mocks into",
        ),
        "_go_context_data": attr.label(
            default = "@io_bazel_rules_go//:go_context_data",
        ),
    },
    toolchains = ["@io_bazel_rules_go//go:toolchain"],
)