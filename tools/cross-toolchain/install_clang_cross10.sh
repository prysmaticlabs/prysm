#!/bin/sh

set -eu

curl -L https://github.com/llvm/llvm-project/releases/download/llvmorg-${INSTALL_LLVM_VERSION}/clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-linux-gnu-ubuntu-18.04.tar.xz \
    -o clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-linux-gnu-ubuntu-18.04.tar.xz
tar xf clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-linux-gnu-ubuntu-18.04.tar.xz --strip-components=1 -C /usr
rm -f clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-linux-gnu-ubuntu-18.04.tar.xz
# arm64
curl -L https://github.com/llvm/llvm-project/releases/download/llvmorg-${INSTALL_LLVM_VERSION}/clang+llvm-${INSTALL_LLVM_VERSION}-aarch64-linux-gnu.tar.xz \
    -o clang+llvm-${INSTALL_LLVM_VERSION}-aarch64-linux-gnu.tar.xz
tar xf clang+llvm-${INSTALL_LLVM_VERSION}-aarch64-linux-gnu.tar.xz
rm -f clang+llvm-${INSTALL_LLVM_VERSION}-aarch64-linux-gnu.tar.xz
mkdir -p /usr/aarch64-linux-gnu/lib/clang/10.0.0
mv /clang+llvm-${INSTALL_LLVM_VERSION}-aarch64-linux-gnu/include/c++/v1 /usr/aarch64-linux-gnu/include/c++/
mv /clang+llvm-${INSTALL_LLVM_VERSION}-aarch64-linux-gnu/lib/clang/10.0.0/include /usr/aarch64-linux-gnu/lib/clang/10.0.0
mv /clang+llvm-${INSTALL_LLVM_VERSION}-aarch64-linux-gnu/lib/libc++.a /usr/aarch64-linux-gnu/lib/
mv /clang+llvm-${INSTALL_LLVM_VERSION}-aarch64-linux-gnu/lib/libc++abi.a /usr/aarch64-linux-gnu/lib/
mv /clang+llvm-${INSTALL_LLVM_VERSION}-aarch64-linux-gnu/lib/libunwind.a /usr/aarch64-linux-gnu/lib/
mv /clang+llvm-${INSTALL_LLVM_VERSION}-aarch64-linux-gnu/lib/clang/10.0.0/lib/linux/libclang_rt.builtins-aarch64.a /usr/lib/clang/10.0.0/lib/linux/
mv /clang+llvm-${INSTALL_LLVM_VERSION}-aarch64-linux-gnu/lib/clang/10.0.0/lib/linux/clang_rt.crtbegin-aarch64.o /usr/lib/clang/10.0.0/lib/linux/
mv /clang+llvm-${INSTALL_LLVM_VERSION}-aarch64-linux-gnu/lib/clang/10.0.0/lib/linux/clang_rt.crtend-aarch64.o /usr/lib/clang/10.0.0/lib/linux/
rm -rf /clang+llvm-${INSTALL_LLVM_VERSION}-aarch64-linux-gnu

# darwin
curl -L https://github.com/llvm/llvm-project/releases/download/llvmorg-${INSTALL_LLVM_VERSION}/clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-apple-darwin.tar.xz \
    -o clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-apple-darwin.tar.xz
tar xf clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-apple-darwin.tar.xz
rm -f clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-apple-darwin.tar.xz
mkdir -p /usr/x86_64-apple-darwin/lib/clang/10.0.0
mkdir -p /usr/x86_64-apple-darwin/include/c++
mkdir -p /usr/lib/clang/10.0.0/lib/darwin/
mv /clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-apple-darwin/include/c++/v1 /usr/x86_64-apple-darwin/include/c++/
mv /clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-apple-darwin/lib/clang/10.0.0/include /usr/x86_64-apple-darwin/lib/clang/10.0.0/
mv /clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-apple-darwin/lib/libc++.a /usr/x86_64-apple-darwin/lib/
mv /clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-apple-darwin/lib/libc++abi.a /usr/x86_64-apple-darwin/lib/
mv /clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-apple-darwin/lib/libunwind.a /usr/x86_64-apple-darwin/lib/
mv /clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-apple-darwin/lib/clang/10.0.0/lib/darwin/libclang_rt.osx.a /usr/lib/clang/10.0.0/lib/darwin/
mv /clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-apple-darwin/lib/clang/10.0.0/lib/darwin/libclang_rt.cc_kext.a /usr/lib/clang/10.0.0/lib/darwin/
rm -rf /clang+llvm-${INSTALL_LLVM_VERSION}-x86_64-apple-darwin
