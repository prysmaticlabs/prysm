#!/bin/sh

set -eu

OSXCROSS_REPO=tpoechtrager/osxcross
OSXCROSS_SHA1=bee9df6
DARWIN_SDK_URL=https://www.dropbox.com/s/yfbesd249w10lpc/MacOSX10.10.sdk.tar.xz

# darwin
mkdir -p /usr/x86_64-apple-darwin/osxcross
mkdir -p /tmp/osxcross && cd "/tmp/osxcross"
curl -sLo osxcross.tar.gz "https://codeload.github.com/${OSXCROSS_REPO}/tar.gz/${OSXCROSS_SHA1}"
tar --strip=1 -xzf osxcross.tar.gz
rm -f osxcross.tar.gz
curl -sLo tarballs/MacOSX10.10.sdk.tar.xz "${DARWIN_SDK_URL}"
yes "" | SDK_VERSION=10.10 OSX_VERSION_MIN=10.10 ./build.sh
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
