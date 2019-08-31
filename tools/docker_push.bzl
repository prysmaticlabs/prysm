load("@io_bazel_rules_docker//container:new_push.bzl", "new_container_push")

def docker_push(*args, **kwargs):
    if "format" in kwargs:
        fail(
            "Cannot override 'format' attribute on docker_push",
            attr = "format",
        )
    kwargs["format"] = "Docker"
    new_container_push(*args, **kwargs)