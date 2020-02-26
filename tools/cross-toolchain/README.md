# WIP Multiarch Cross Compiling Toolchain

### Containerized Builds
This project declares a c++ toolchain suite with cross compilers for targeting four platforms:
* linux_amd64
* linux_arm64
* osx_amd64
* windows_amd64

The toolchain suite describes cross compilers defined in a docker container described by a Dockerfile, also included in this project.


### Using Published Docker Container for Cross Compilation Targets:
This is still a WIP, at the time of this writing only linux_amd64 and linux_arm64 are working targets.

#### Run the cross compiler image
1. checkout prysm, `git clone https://github.com/prysmaticlabs/prysm`
2. cd prysm
3. ``docker run -it -v $(pwd):/workdir suburbandad/cross-clang-10:latest` 

From here you can run builds inside the linux x86_64 container image, e.g.:

|    arch |   os    |    config     | working? | example cmd   |
|--------|----------|---------------|----------|---------------|
| arm64   | linux   | linux_arm64   |  Y       | `bazel build --config=linux_arm64 //beacon-chain` |
| x86_64  | linux   | linux_amd64   |  Y       | `bazel build --config=linux_amd64 //beacon-chain` |
| x86_64  | osx     | osx_amd64     |  N       | `bazel build --config=osx_arm64 //beacon-chain` |
| x86_64  | windows | windows_amd64 |  N       | `bazel build --config=windows_arm64 //beacon-chain` |


#### Or, if you just want to run a particular target, this is handy:
For example, to build the beacon chain for linux_arm64: 
`docker run -it -v $(pwd):/workdir suburbandad/cross-clang-10:latest bazel build --config=linux_arm64 //beacon-chain`
` 

Also fun, if you are on OSX can build and run a linux_amd64 beacon-chain:
`docker run -it -v $(pwd):/workdir suburbandad/cross-clang-10:latest bazel run //beacon-chain` 
            

### Coming soon
* Mac OSX x86_64 builds
* Windows x86_64 builds
            