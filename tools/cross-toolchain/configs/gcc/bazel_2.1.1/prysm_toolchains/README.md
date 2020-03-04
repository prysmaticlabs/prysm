# Multiarch Cross Compiling Toolchain

### Containerized Builds
This project declares a c++ toolchain suite with cross compilers for targeting four platforms:
* linux_amd64
* linux_arm64
* osx_amd64
* windows_amd64

The toolchain suite describes cross compilers defined in a docker container described by a Dockerfile, also included in this project.

### Using Published Docker Container for Cross Compilation Targets:
At the time of this writing linux_amd64, linux_arm64, osx_amd64, and windows_amd64 are working targets.

#### If your host machine is linux_amd64
If you are on linux_amd64 and you have docker configured, you can simply use bazel with the docker target configs.  See the table below.

#### Otherwise run the cross compiler image

1. checkout prysm, `git clone https://github.com/prysmaticlabs/prysm`
2. cd prysm
3. `docker run -it -v $(pwd):/workdir gcr.io/prysmaticlabs/rbe-worker` 

From here you can run builds inside the linux x86_64 container image, e.g.:

|    arch |   os    |    config     | working? | bazel docker config (for linux_arm64 hosts) |
|---------|---------|---------------|----------|---------------|
| arm64   | linux   | linux_arm64   |  Y       | `RBE_AUTOCONF_ROOT=$(bazel info workspace) bazel build --config=linux_arm64_docker //beacon-chain`   |
| x86_64  | linux   | linux_amd64   |  Y       | `RBE_AUTOCONF_ROOT=$(bazel info workspace) bazel build --config=linux_amd64_docker //beacon-chain`   |
| x86_64  | osx     | osx_amd64     |  Y       | `RBE_AUTOCONF_ROOT=$(bazel info workspace) bazel build --config=osx_amd64_docker //beacon-chain`     |
| x86_64  | windows | windows_amd64 |  y       | `RBE_AUTOCONF_ROOT=$(bazel info workspace) bazel build --config=windows_amd64_docker //beacon-chain` |


#### Or, if you just want to run a particular target, this is handy:
For example, to build the beacon chain for linux_arm64: 
`docker run -it -v $(pwd):/workdir gcr.io/prysmaticlabs/rbe-worker bazel build --config=linux_arm64 //beacon-chain`
 

Also fun, if you are on OSX or windows, you can build and run a linux_amd64 beacon-chain:
`docker run -it -v $(pwd):/workdir gcr.io/prysmaticlabs/rbe-worker bazel run //beacon-chain` 
            
### Rebuilding Remote Build Execution (RBE) Configuration

```
RBE_AUTOCONF_ROOT=$(bazel info workspace) bazel build @rbe_default//...
```

