load(
    "@io_bazel_rules_go//go:def.bzl",
    "GoLibrary", "GoArchive"
)
load("@bazel_skylib//:lib.bzl", "shell")
load("@io_bazel_rules_go//go:def.bzl", "go_path")

solidity_filetype = FileType([".sol"])

def _sol2go_impl(ctx):
  in_file = ctx.files.src
  output = ctx.outputs.out
  script = ctx.files._script[0]
  abigen = ctx.executable._abigen
  solc = ctx.file._solc

  archive = ctx.attr._deps[0][GoArchive]
  importmap = archive.data.importmap

  go_files_out = []
  go_files_out.append(output)

  ctx.actions.run_shell(
      tools=[abigen, script, solc],
      inputs=in_file,
      outputs=go_files_out,
      command="%s %s %s %s %s %s %s %s com_github_ethereum_go_ethereum" % (script.path, in_file[0].path, abigen.path, output.path, ctx.attr.pkg, solc.path, importmap, importmap.rpartition('/')[0]))

sol2go = rule(
  _sol2go_impl,
  attrs = {
    "pkg": attr.string(
        mandatory = True,
        doc = "Destination package",
    ),
    "src":  attr.label(allow_files=solidity_filetype),
    "_deps": attr.label_list(
        providers = [GoLibrary],
        default = ["@com_github_ethereum_go_ethereum//:go_default_library"]
    ),
    "_abigen": attr.label(
        default = "@com_github_ethereum_go_ethereum//cmd/abigen:abigen",
        cfg = "host",
        executable = True,
    ),
    "_solc": attr.label(
        default = "@solc_sdk//solidity_compiler:solc",
        allow_single_file = True
    ),
    "_script": attr.label(
        default = "@solc_sdk//solidity_compiler:sol2go.sh",
        allow_single_file = True
    ),
  },
  outputs = {"out": "%{src}.go"},
)
