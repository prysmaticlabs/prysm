def sszgenerator(name, srcs, outfile, **kwargs):
  """
  Generate custom marshal/unmarshal functions for any SSZ-capable
  data structure using the fastssz library by ferranbt.
  """
  print(srcs)
  native.genrule(
    name = name,
    srcs = srcs,
    outs = [outfile],
    cmd = "$(location @com_github_ferranbt_fastssz//sszgen:sszgen) $< --path $(@D) --objs AttestationData --output $@",
    tools = [
        "@com_github_ferranbt_fastssz//sszgen:sszgen",
    ],
    **kwargs
  )
