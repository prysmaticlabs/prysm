package(default_visibility = ["//k8s:__subpackages__"])

load("@io_bazel_rules_k8s//k8s:objects.bzl", "k8s_objects")
load("@k8s_deploy//:defaults.bzl", "k8s_deploy")

k8s_objects(
    name = "everything",
    objects = [
        ":deployments",
    ],
)

_deployments = [
    "proposer",
]

k8s_objects(
    name = "deployments",
    objects = [":" + name + ".deploy" for name in _deployments],
)

[k8s_deploy(
    name = name + ".deploy",
    template = name + ".deploy.yaml",
) for name in _deployments]
