# Prysmatic Labs Remote Build Execution Image with Cross Compile Support.
#
# Update instructions.
# - Build the docker image.
# - Push to gcr.io.
# - Update _PRYSM_BUILD_IMAGE_DIGEST in //tools/cross-toolchain/rbe_toolchains_config.bzl
# - Run ./tools/cross-toolchain/regenerate.sh
# - Add and commit the newly generated configs.
# - Done!
#
# docker build -t gcr.io/prysmaticlabs/rbe-worker:latest .
# gcloud docker -- push gcr.io/prysmaticlabs/rbe-worker:latest 

FROM debian:bullseye-slim as build

# install gnu/gcc cross-build toolchain (gcc 8.3)
ENV DOCKER_CLI_EXPERIMENTAL=enabled
RUN apt-get update && \
    apt-get install -y \
        curl xz-utils \
        gcc g++ mingw-w64 \
        gcc-aarch64-linux-gnu g++-aarch64-linux-gnu \
        cmake libssl-dev libxml2-dev vim apt-transport-https \
        zip unzip libtinfo5 patch zlib1g-dev autoconf libtool \
        pkg-config make docker.io gnupg2 libgmp-dev python

# install llvm/clang cross-build toolchains
ENV INSTALL_LLVM_VERSION=12.0.0
ADD install_clang_cross.sh /tmp/install_clang_cross.sh
RUN /tmp/install_clang_cross.sh

# install osxcross
ADD install_osxcross.sh /tmp/install_osxcross.sh
ADD common_osxcross.sh /tmp/common_osxcross.sh
RUN /tmp/install_osxcross.sh
ADD link_osxcross.sh /tmp/link_osxcross.sh
RUN /tmp/link_osxcross.sh

# containerized development environment
FROM build as devel
RUN mkdir /workdir
WORKDIR /workdir
RUN echo 'PS1="\[$(tput setaf 3)$(tput bold)[\]devel@\\h:\\W]#\[$(tput sgr0) \]"' >> /root/.bashrc
