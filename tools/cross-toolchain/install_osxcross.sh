#!/bin/sh

set -eu

OSXCROSS_REPO=tpoechtrager/osxcross
OSXCROSS_SHA1=bee9df6
OSX_SDK=MacOSX10.10.sdk
OSX_SDK_SUM=631b4144c6bf75bf7a4d480d685a9b5bda10ee8d03dbf0db829391e2ef858789

# darwin
mkdir -p /usr/x86_64-apple-darwin/osxcross
mkdir -p /tmp/osxcross && cd "/tmp/osxcross"
curl -sLo osxcross.tar.gz "https://codeload.github.com/${OSXCROSS_REPO}/tar.gz/${OSXCROSS_SHA1}"
tar --strip=1 -xzf osxcross.tar.gz
rm -f osxcross.tar.gz
curl -sLo tarballs/${OSX_SDK}.tar.xz "https://s3.dockerproject.org/darwin/v2/${OSX_SDK}.tar.xz"
echo "${OSX_SDK_SUM}"  "tarballs/${OSX_SDK}.tar.xz" | sha256sum -c -
yes "" | SDK_VERSION=10.10 OSX_VERSION_MIN=10.10 OCDEBUG=1 ./build.sh
mv target/* /usr/x86_64-apple-darwin/osxcross/
mv tools /usr/x86_64-apple-darwin/osxcross/
cd /usr/x86_64-apple-darwin/osxcross/include
ln -s ../SDK/MacOSX10.10.sdk/System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/CarbonCore.framework/Versions/A/Headers/ CarbonCore
ln -s ../SDK/MacOSX10.10.sdk/System/Library/Frameworks/CoreFoundation.framework/Versions/A/Headers/ CoreFoundation
ln -s ../SDK/MacOSX10.10.sdk/System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/ Frameworks
ln -s ../SDK/MacOSX10.10.sdk/System/Library/Frameworks/Security.framework/Versions/A/Headers/ Security
rm -rf /tmp/osxcross
rm -rf "/usr/x86_64-apple-darwin/osxcross/SDK/MacOSX10.10.sdk/usr/share/man"
# symlink ld64.lld
ln -s /usr/x86_64-apple-darwin/osxcross/bin/x86_64-apple-darwin14-ld /usr/x86_64-apple-darwin/osxcross/bin/ld64.lld
ln -s /usr/x86_64-apple-darwin/osxcross/lib/libxar.so.1 /usr/lib
