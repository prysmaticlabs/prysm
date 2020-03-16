# A genrule to run the beacon-fuzz tool for generating source files with beacon states hard coded as
# hex strings.
def beacon_fuzz_state_gen(name, corpus, package_name):
    native.genrule(
        name = name,
        outs = ["beacon_fuzz_generated_states.go"],
        srcs = [corpus],
        visibility = ["//visibility:private"],
        testonly = 1,
        tools = ["//tools/beacon-fuzz:beacon-fuzz"],
        cmd = "$(location //tools/beacon-fuzz:beacon-fuzz) --output=$@ $(locations %s)" % corpus,
    )
