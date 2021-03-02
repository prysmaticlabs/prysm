def moved_targets(targets, new_package):
  for target in targets:
    native.alias(name=target, actual=new_package+target, deprecation="This target has moved to %s%s"%(actual,target))