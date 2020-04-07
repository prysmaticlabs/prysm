load("@io_bazel_rules_docker//container:providers.bzl", "ImageInfo")

# Defines the debug transition implementation to enforce dbg mode.
def _debug_transition_impl(settings, attr):
    return {
        "//command_line_option:compilation_mode": "dbg",
    }

# Defines a starlark transition which enforces dbg compilation mode.
build_in_debug_mode = transition(
    implementation = _debug_transition_impl,
    inputs = [],
    outputs = ["//command_line_option:compilation_mode"],
)

def _alpine_transition_impl(settings, attr):
    return {
        "//tools:base_image": "alpine",
    }

use_alpine = transition(
    implementation = _alpine_transition_impl,
    inputs = [],
    outputs = ["//tools:base_image"],
)

# Defines a rule implementation that essentially returns all of the providers from the image attr.
def _go_image_debug_impl(ctx):
    img = ctx.attr.image[0]

    return [
        img[ImageInfo],
        img[OutputGroupInfo],
    ]

# Defines a rule that specifies a starlark transition to enforce debug compilation mode for debug
# images.
go_image_debug = rule(
    implementation = _go_image_debug_impl,
    attrs = {
        "image": attr.label(
            cfg = build_in_debug_mode,
            executable = True,
        ),
        # Whitelist is required or bazel complains.
        "_whitelist_function_transition": attr.label(default = "@bazel_tools//tools/whitelists/function_transition_whitelist"),
    },
)
go_image_alpine = rule(
    _go_image_debug_impl,
    attrs = {
        "image": attr.label(
            cfg = use_alpine,
            executable = True,
        ),
        # Whitelist is required or bazel complains.
        "_whitelist_function_transition": attr.label(default = "@bazel_tools//tools/whitelists/function_transition_whitelist"),
    },
)
