load(
    "@bazel_tools//tools/cpp:cc_toolchain_config_lib.bzl",
    "action_config",
    "feature",
    "feature_set",
    "flag_group",
    "flag_set",
    "make_variable",
    "tool",
    "tool_path",
    "with_feature_set",
)

load(
    "@bazel_tools//tools/cpp:cc_toolchain_config.bzl",
    ALL_COMPILE_ACTIONS = "all_compile_actions",
    ALL_CPP_COMPILE_ACTIONS = "all_cpp_compile_actions",
    ALL_LINK_ACTIONS = "all_link_actions",
)

def _impl(ctx):
    toolchain_identifier = "osxcross"
    compiler = "clang"
    abi_version = "darwin_x86_64"
    abi_libc_version = "darwin_x86_64"
    install = "/usr/x86_64-apple-darwin/"
    clang_version = "10.0.0"
    target_libc = "macosx"
    target_cpu = "x86_64"
    osxcross = install + "osxcross/"
    osxcross_binprefix = osxcross + "bin/x86_64-apple-darwin19-"
    sdkroot = osxcross + "SDK/MacOSX10.15.sdk/"
    cross_system_include_dirs = [
        "/usr/lib/clang/10.0.0/include",
        osxcross + "include",
        sdkroot + "usr/include",
    ]

    opt_feature = feature(name = "opt")
    dbg_feature = feature(name = "dbg")
    fastbuild_feature = feature(name = "fastbuild")
    random_seed_feature = feature(name = "random_seed", enabled = True)
    supports_pic_feature = feature(name = "supports_pic", enabled = True)
    supports_dynamic_linker_feature = feature(name = "supports_dynamic_linker", enabled = True)

    unfiltered_compile_flags_feature = feature(
        name = "unfiltered_compile_flags",
        enabled = True,
        flag_sets = [
            flag_set(
                actions = ALL_COMPILE_ACTIONS,
                flag_groups = [
                    flag_group(
                        flags = [
                            "-stdlib=libc++",
                            "-no-canonical-prefixes",
                            "-Wno-builtin-macro-redefined",
                            "-D__DATE__=\"redacted\"",
                            "-D__TIMESTAMP__=\"redacted\"",
                            "-D__TIME__=\"redacted\"",
                        ],
                    ),
                ],
            ),
        ],
    )

    # explicit arch specific system includes
    system_include_flags = []
    for d in cross_system_include_dirs:
        system_include_flags += ["-idirafter", d]

    default_compile_flags_feature = feature(
        name = "default_compile_flags",
        enabled = True,
        flag_sets = [
            flag_set(
                actions = ALL_COMPILE_ACTIONS,
                flag_groups = [
                    flag_group(
                        flags = [
                            "-mlinker-version=400",
                            "-B " + osxcross + "bin",
                            "-nostdinc",
                            "-U_FORTIFY_SOURCE",
                            "-fstack-protector",
                            "-fno-omit-frame-pointer",
                            "-fcolor-diagnostics",
                            "-Wall",
                            "-Wthread-safety",
                            "-Wself-assign",
                        ] + system_include_flags,
                    ),
                ],
            ),
            flag_set(
                actions = ALL_COMPILE_ACTIONS,
                flag_groups = [flag_group(flags = ["-g", "-fstandalone-debug"])],
                with_features = [with_feature_set(features = ["dbg"])],
            ),
            flag_set(
                actions = ALL_COMPILE_ACTIONS,
                flag_groups = [
                    flag_group(
                        flags = [
                            "-g0",
                            "-O2",
                            "-D_FORTIFY_SOURCE=1",
                            "-DNDEBUG",
                            "-ffunction-sections",
                            "-fdata-sections",
                        ],
                    ),
                ],
                with_features = [with_feature_set(features = ["opt"])],
            ),
            flag_set(
                actions = ALL_CPP_COMPILE_ACTIONS,
                flag_groups = [flag_group(flags = ["-std=c++17", "-nostdinc++"])],
            ),
        ],
    )

    default_link_flags_feature = feature(
        name = "default_link_flags",
        enabled = True,
        flag_sets = [
            flag_set(
                actions = ALL_LINK_ACTIONS,
                flag_groups = [
                    flag_group(
                        flags = [
                            "-v",
                            "-lm",
                            "-no-canonical-prefixes",
                            "-lc++",
                            "-lc++abi",
                            "-F" + sdkroot + "System/Library/Frameworks/",
                            "-L"+ sdkroot + "usr/lib",
                            "-undefined",
                            "dynamic_lookup",
                            ],
                    ),
                ],
            ),
        ],
    )

    objcopy_embed_flags_feature = feature(
        name = "objcopy_embed_flags",
        enabled = True,
        flag_sets = [
            flag_set(
                actions = ["objcopy_embed_data"],
                flag_groups = [flag_group(flags = ["-I", "binary"])],
            ),
        ],
    )

    user_compile_flags_feature = feature(
        name = "user_compile_flags",
        enabled = True,
        flag_sets = [
            flag_set(
                actions = ALL_COMPILE_ACTIONS,
                flag_groups = [
                    flag_group(
                        expand_if_available = "user_compile_flags",
                        flags = ["%{user_compile_flags}"],
                        iterate_over = "user_compile_flags",
                    ),
                ],
            ),
        ],
    )

    coverage_feature = feature(
        name = "coverage",
        flag_sets = [
            flag_set(
                actions = ALL_COMPILE_ACTIONS,
                flag_groups = [
                    flag_group(
                        flags = ["-fprofile-instr-generate", "-fcoverage-mapping"],
                    ),
                ],
            ),
            flag_set(
                actions = ALL_LINK_ACTIONS,
                flag_groups = [flag_group(flags = ["-fprofile-instr-generate"])],
            ),
        ],
        provides = ["profile"],
    )

    features = [
        opt_feature,
        fastbuild_feature,
        dbg_feature,
        random_seed_feature,
        supports_pic_feature,
        supports_dynamic_linker_feature,
        unfiltered_compile_flags_feature,
        default_link_flags_feature,
        default_compile_flags_feature,
        objcopy_embed_flags_feature,
        user_compile_flags_feature,
        coverage_feature,
    ]

    tool_paths = [
        tool_path(name = "ld", path = osxcross_binprefix + "ld"),
        tool_path(name = "cpp", path = osxcross + "bin/o64-clang++"),
        tool_path(name = "dwp", path = "/usr/bin/dwp"),
        tool_path(name = "gcov", path = "/usr/bin/gcov"),
        tool_path(name = "nm", path = osxcross_binprefix + "nm"),
        tool_path(name = "objdump", path = osxcross_binprefix + "ObjectDump"),
        tool_path(name = "strip", path = osxcross_binprefix + "strip"),
        tool_path(name = "gcc", path = osxcross + "bin/o64-clang"),
        tool_path(name = "ar", path = osxcross_binprefix + "libtool"),
    ]

    return cc_common.create_cc_toolchain_config_info(
        ctx = ctx,
        features = features,
        abi_version = abi_version,
        abi_libc_version = abi_libc_version,
        compiler = compiler,
        cxx_builtin_include_directories = cross_system_include_dirs,
        host_system_name = "x86_64-unknown-linux-gnu",
        target_cpu = target_cpu,
        target_libc = target_libc,
        target_system_name = ctx.attr.target,
        tool_paths = tool_paths,
        toolchain_identifier = toolchain_identifier,
    )

osx_cc_toolchain_config = rule(
    implementation = _impl,
    attrs = {
        "target": attr.string(mandatory = True),
        "stdlib": attr.string(),
    },
    provides = [CcToolchainConfigInfo],
)
