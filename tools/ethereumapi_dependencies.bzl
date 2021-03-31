load("@bazel_gazelle//:deps.bzl", "go_repository")

def ethereumapi_deps():
    go_repository(
        name = "com_github_protolambda_zssz",
        importpath = "github.com/protolambda/zssz",
        sum = "h1:7fjJjissZIIaa2QcvmhS/pZISMX21zVITt49sW1ouek=",
        version = "v0.1.5",
    )
    go_repository(
        name = "com_github_prysmaticlabs_go_ssz",
        importpath = "github.com/prysmaticlabs/go-ssz",
        sum = "h1:7qd0Af1ozWKBU3c93YW2RH+/09hJns9+ftqWUZyts9c=",
        version = "v0.0.0-20200612203617-6d5c9aa213ae",
    )
