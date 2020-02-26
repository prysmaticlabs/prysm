# Copyright 2019 Erik Maciejewski
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# containerized build environment
FROM debian:buster as build

# install gnu/gcc cross-build toolchain (gcc 8.3)
ENV DOCKER_CLI_EXPERIMENTAL=enabled
RUN apt-get update && \
    apt-get install -y \
        curl xz-utils \
        gcc g++ \
        gcc-arm-linux-gnueabihf g++-arm-linux-gnueabihf \
        gcc-aarch64-linux-gnu g++-aarch64-linux-gnu \
        cmake libssl-dev libxml2-dev vim apt-transport-https \
        zip unzip libtinfo5 patch zlib1g-dev autoconf libtool \
        pkg-config make docker.io gnupg2 && \
    curl https://bazel.build/bazel-release.pub.gpg | apt-key add - && \
    echo "deb [arch=amd64] https://storage.googleapis.com/bazel-apt stable jdk1.8" | tee /etc/apt/sources.list.d/bazel.list && \
    apt update && apt install -y bazel && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* 

# install llvm/clang cross-build toolchains
ENV INSTALL_LLVM_VERSION=10.0.0-rc2
ADD install_clang_cross10.sh /tmp/install_clang_cross.sh
RUN /tmp/install_clang_cross.sh

# install osxcross
RUN git clone https://github.com/tpoechtrager/osxcross.git && \
    cd osxcross/tarballs && \
    curl -LO https://www.dropbox.com/s/yfbesd249w10lpc/MacOSX10.10.sdk.tar.xz && \
    cd .. && chmod 755 build.sh tools/*.sh && UNATTENDED=1 ./build.sh && \
    mv target /usr/x86_64-apple-darwin/osxcross
# containerized development environment
FROM build as devel
RUN mkdir /workdir
WORKDIR /workdir
RUN echo 'PS1="\[$(tput setaf 3)$(tput bold)[\]devel@\\h:\\W]#\[$(tput sgr0) \]"' >> /root/.bashrc
