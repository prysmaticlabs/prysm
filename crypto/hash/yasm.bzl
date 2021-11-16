load("@rules_cc//cc:toolchain_utils.bzl", "find_cpp_toolchain")


def _obj_yasm(ctx, arch, opts, src):
  yasm_bin = ctx.attr.yasm_bin
  out = ctx.actions.declare_file(src.basename.replace(src.extension, "o"))
  opts = arch + ['-o', out.path] + opts + [src.path]
  inputs = []

  for i in ctx.attr.srcs + ctx.attr.hdrs + ctx.attr.deps:
    if hasattr(i, "files"):
      inputs += i.files.to_list()
    else:
      inputs.append(i)

  ctx.actions.run(
      outputs = [out],
      inputs = inputs,
      arguments = opts,
      executable = yasm_bin,
      mnemonic = 'YasmCompile',
  )

  return out

def _library_yasm(ctx, mysrc):
  output_file = ctx.actions.declare_file(ctx.label.name + ".a")

  cc_toolchain = find_cpp_toolchain(ctx)

  feature_configuration = cc_common.configure_features(
      ctx = ctx,
      cc_toolchain = cc_toolchain,
      requested_features = ctx.features,
      unsupported_features = ctx.disabled_features,
  )

  linker_input = cc_common.create_linker_input(
      owner = ctx.label,
      libraries = depset(direct = [
          cc_common.create_library_to_link(
              actions = ctx.actions,
              static_library = output_file,
	      cc_toolchain = cc_toolchain,
	      feature_configuration = feature_configuration,
          ),
      ]),
  )

  compilation_context = cc_common.create_compilation_context()
  linking_context = cc_common.create_linking_context(linker_inputs = depset(direct = [linker_input]))

  ctx.actions.run(
    executable = ctx.attr.ar_bin,
    arguments = ['r', output_file.path] + [i.path for i in mysrc],
    inputs = mysrc,
    outputs = [output_file],
    mnemonic = "Archiving",
  )

  return CcInfo(compilation_context = compilation_context, linking_context = linking_context)

def _yasm_library_impl(ctx):
  opts = ctx.attr.copts 
  deps = [_obj_yasm(ctx, ctx.attr.yasm_arch, opts, src)
          for target in ctx.attr.srcs for src in target.files.to_list()]
  for i in ctx.attr.hdrs:
    if hasattr(i, "files"):
      deps += i.files.to_list()
    else:
      deps.append(i)

  cc_info =  _library_yasm(ctx, deps)

  return [cc_info]


YASM_BIN_DEFAULT = "/usr/bin/yasm"
AR_BIN_DEFAULT = "/usr/bin/ar"
YASM_ARCH_OPTS = ["-f", "elf64", "-m", "amd64"]


_yasm_library = rule(
  implementation=_yasm_library_impl,
  attrs={
    "srcs": attr.label_list(allow_files=True),
    "hdrs": attr.label_list(allow_files=True),
    "deps": attr.label_list(allow_files=True),
    "copts": attr.string_list(),
    "yasm_bin": attr.string(default=""),
    "ar_bin": attr.string(default=""),
    "yasm_arch": attr.string_list(),
    "_cc_toolchain": attr.label(default = Label("@bazel_tools//tools/cpp:current_cc_toolchain")),
  },
  fragments = ["cpp"],
  toolchains = ["@bazel_tools//tools/cpp:toolchain_type"],
  )


def yasm_library(name, srcs, hdrs=[], deps=[], copts=[],
                 yasm_bin=YASM_BIN_DEFAULT, ar_bin=AR_BIN_DEFAULT):
  _yasm_library(
      name = name,
      srcs = srcs,
      hdrs = hdrs,
      copts = copts,
      yasm_bin = yasm_bin,
      ar_bin = ar_bin,
      yasm_arch = YASM_ARCH_OPTS,
  )
