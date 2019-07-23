"""
"""

mainnet = {
    "attestation.aggregation_bits.max": "4096",
}

minimal = {
    "attestation.aggregation_bits.max": "64",
}

def _ssz_proto_files_impl(ctx):
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
            substitutions = mainnet,
        )

    return [DefaultInfo(files = depset(outputs))]

ssz_proto_files = rule(
    implementation = _ssz_proto_files_impl,
    attrs = {
        "srcs": attr.label_list(mandatory = True, allow_files = [".proto"]),
        "config": attr.string(mandatory = True),
    },
)
