def sszgenerator(name, srcs, outs, **kwargs):
  """
  Generate custom marshal/unmarshal functions for any SSZ-capable
  data structure using the fastssz library by ferranbt.
  """
  native.genrule(
    name = name,
    srcs = srcs,
    outs = outs,
    cmd = "$(location @com_github_ferranbt_fastssz//sszgen:sszgen) $< $@",
    **kwargs
  )
