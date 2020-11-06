#!/bin/sh

set -eu

OSXCROSS_REPO=tpoechtrager/osxcross
OSXCROSS_SHA1=2733413b6847c1489d6230f062d3293e6f42a021
OSX_SDK=MacOSX10.15.sdk
OSX_SDK_SUM=f1703b980d479b367d5bddd299fcad7e0ade2fe5019e571359f52ef2c58872a9

# darwin
mkdir -p /usr/x86_64-apple-darwin/osxcross
mkdir -p /tmp/osxcross && cd "/tmp/osxcross"
curl -sLo osxcross.tar.gz "https://codeload.github.com/${OSXCROSS_REPO}/tar.gz/${OSXCROSS_SHA1}"
tar --strip=1 -xzf osxcross.tar.gz
rm -f osxcross.tar.gz
curl -sLo tarballs/${OSX_SDK}.tar.xz "https://prysmaticlabs.com/uploads/${OSX_SDK}.tar.xz"
echo "${OSX_SDK_SUM}"  "tarballs/${OSX_SDK}.tar.xz" | sha256sum -c -
yes "" | SDK_VERSION=10.15 OSX_VERSION_MIN=10.12 OCDEBUG=1 ./build.sh
mv target/* /usr/x86_64-apple-darwin/osxcross/
mv tools /usr/x86_64-apple-darwin/osxcross/
cd /usr/x86_64-apple-darwin/osxcross/include
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/CarbonCore.framework/Versions/A/Headers/ CarbonCore
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CoreFoundation.framework/Versions/A/Headers/ CoreFoundation
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/ Frameworks
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/Security.framework/Versions/A/Headers/ Security
rm -rf /tmp/osxcross
rm -rf "/usr/x86_64-apple-darwin/osxcross/SDK/${OSX_SDK}/usr/share/man"
# symlink ld64.lld
ln -s /usr/x86_64-apple-darwin/osxcross/bin/x86_64-apple-darwin19-ld /usr/x86_64-apple-darwin/osxcross/bin/ld64.lld
ln -s /usr/x86_64-apple-darwin/osxcross/lib/libxar.so.1 /usr/lib
ln -s /usr/x86_64-apple-darwin/osxcross/lib/libtapi.so* /usr/lib
