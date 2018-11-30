"""TODO: Add doc here"""

load("@k8s_secret//:defaults.bzl", "k8s_secret")

def _k8s_encrypted_secret_impl(ctx):
  ctx.actions.run_shell(
    inputs = [ctx.file.template],
    outputs = [ctx.outputs.out],
    progress_message = "Decrypting %s" % ctx.file.template,
    tools = [ctx.executable._kubesec],
    command = "%s decrypt %s > %s" % (ctx.executable._kubesec.path, ctx.file.template.path, ctx.outputs.out.path)
  )

k8s_encrypted_secret = rule(
    implementation = _k8s_encrypted_secret_impl,
    attrs = {
      "_kubesec": attr.label(
        executable = True,
        cfg = "host",
        default = "//tools:kubesec",
      ),
      "template": attr.label(
          allow_files = True, 
          single_file = True, 
          mandatory = True
      ),
      "out": attr.output(mandatory = True),
    },
)
