"""
SSZ proto templating rules.

These rules allow for variable substitution for hardcoded tag values like ssz-size and ssz-max.

"""

####### Configuration #######

mainnet = {
    "block_roots.size": "8192,32",
    "state_roots.size": "8192,32",
    "eth1_data_votes.size": "1024",
    "randao_mixes.size": "65536,32",
    "active_index_roots.size": "65536,32",
    "compact_committees_roots.size": "65536,32",
    "previous_epoch_attestations.max": "8192",
    "current_epoch_attestations.max": "8192",
    "previous_crosslinks.size": "1024",
    "current_crosslinks.size": "1024",
    "slashings.size": "8192",
}

minimal = {
    "block_roots.size": "64,32",
    "state_roots.size": "64,32",
    "eth1_data_votes.size": "16",
    "randao_mixes.size": "64,32",
    "active_index_roots.size": "64,32",
    "compact_committees_roots.size": "64,32",
    "previous_epoch_attestations.max": "1024",
    "current_epoch_attestations.max": "1024",
    "previous_crosslinks.size": "8",
    "current_crosslinks.size": "8",
    "slashings.size": "64",
}

###### Rules definitions #######

def _ssz_proto_files_impl(ctx):
    """
    ssz_proto_files implementation performs expand_template based on the value of "config".
    """
    outputs = []
    if (ctx.attr.config.lower() == "mainnet"):
        subs = mainnet
    elif (ctx.attr.config.lower() == "minimal"):
        subs = minimal
    else:
        fail("%s is an unknown configuration" % ctx.attr.config)

    for src in ctx.attr.srcs:
        output = ctx.actions.declare_file(src.files.to_list()[0].basename)
        outputs.append(output)
        ctx.actions.expand_template(
            template = src.files.to_list()[0],
            output = output,
            substitutions = subs,
        )

    return [DefaultInfo(files = depset(outputs))]

ssz_proto_files = rule(
    implementation = _ssz_proto_files_impl,
    attrs = {
        "srcs": attr.label_list(mandatory = True, allow_files = [".proto"]),
        "config": attr.string(mandatory = True),
    },
)
