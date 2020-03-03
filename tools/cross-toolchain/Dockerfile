# docker build -t gcr.io/prysmaticlabs/rbe-worker:latest .
# gcloud docker -- push gcr.io/prysmaticlabs/rbe-worker:latest

# containerized build environment
FROM gcr.io/cloud-marketplace/google/rbe-ubuntu18-04 as build

# install gnu/gcc cross-build toolchain (gcc 8.3)
ENV DOCKER_CLI_EXPERIMENTAL=enabled
RUN apt-get update && \
    apt-get install -y \
        curl xz-utils \
        gcc g++ mingw-w64 \
        gcc-aarch64-linux-gnu g++-aarch64-linux-gnu \
        cmake libssl-dev libxml2-dev vim apt-transport-https \
        zip unzip libtinfo5 patch zlib1g-dev autoconf libtool \
        pkg-config make docker.io gnupg2

# install bazel
RUN curl https://bazel.build/bazel-release.pub.gpg | apt-key add - && \
    echo "deb [arch=amd64] https://storage.googleapis.com/bazel-apt stable jdk1.8" | tee /etc/apt/sources.list.d/bazel.list && \
    apt update && apt install -y bazel && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* 

# install llvm/clang cross-build toolchains
ENV INSTALL_LLVM_VERSION=10.0.0-rc2
ADD install_clang_cross10.sh /tmp/install_clang_cross.sh
RUN /tmp/install_clang_cross.sh

# install osxcross
ADD install_osxcross.sh /tmp/install_osxcross.sh
RUN /tmp/install_osxcross.sh

# containerized development environment
FROM build as devel
RUN mkdir /workdir
WORKDIR /workdir
RUN echo 'PS1="\[$(tput setaf 3)$(tput bold)[\]devel@\\h:\\W]#\[$(tput sgr0) \]"' >> /root/.bashrc
