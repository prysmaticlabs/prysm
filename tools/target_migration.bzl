def moved_targets(targets, new_package):
  for target in targets:
    native.alias(
      name=target[1:],
      actual=new_package+target,
      deprecation="This target has moved to %s%s"%(new_package,target),
      tags = ["manual"],
    )
