#!/bin/sh
. "$(dirname "$0")"/common_osxcross.sh

# SymLink includes and clean up 

cd "/tmp/osxcross"
mv target/* /usr/x86_64-apple-darwin/osxcross/
mv tools /usr/x86_64-apple-darwin/osxcross/
cd /usr/x86_64-apple-darwin/osxcross/include
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/CarbonCore.framework/Versions/A/Headers/ CarbonCore
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CoreFoundation.framework/Versions/A/Headers/ CoreFoundation
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/ Frameworks
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/Security.framework/Versions/A/Headers/ Security
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CoreServices.framework/Headers/ CoreServices
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/DiskArbitration.framework/Headers/ DiskArbitration
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CoreServices.framework/Frameworks/AE.framework/Headers/ AE
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/IOKit.framework/Headers/ IOKit
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CFNetwork.framework/Headers/ CFNetwork
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CoreServices.framework/Frameworks/DictionaryServices.framework/Headers/ DictionaryServices
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Headers/ LaunchServices
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CoreServices.framework/Frameworks/Metadata.framework/Headers/ Metadata
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CoreServices.framework/Frameworks/OSServices.framework/Headers/ OSServices
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CoreServices.framework/Frameworks/SearchKit.framework/Headers/ SearchKit
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CoreServices.framework/Versions/Current/Frameworks/FSEvents.framework/Headers/ FSEvents
ln -s ../SDK/${OSX_SDK}/System/Library/Frameworks/CoreServices.framework/Versions/Current/Frameworks/SharedFileList.framework/Headers/ SharedFileList

rm -rf /tmp/osxcross
rm -rf "/usr/x86_64-apple-darwin/osxcross/SDK/${OSX_SDK}/usr/share/man"
# symlink ld64.lld
ln -s /usr/x86_64-apple-darwin/osxcross/bin/x86_64-apple-darwin19-ld /usr/x86_64-apple-darwin/osxcross/bin/ld64.lld
ln -s /usr/x86_64-apple-darwin/osxcross/lib/libxar.so.1 /usr/lib
ln -s /usr/x86_64-apple-darwin/osxcross/lib/libtapi.so* /usr/lib