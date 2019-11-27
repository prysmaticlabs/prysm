# Third Party Package Patching

This directory includes local patches to third party dependencies we use in Prysm. Sometimes,
we need to make a small change to some dependency for ease of use in Prysm without wanting
to maintain our own fork of the dependency ourselves. Our build tool, [Bazel](https://bazel.build)
allows us to include patches in a seamless manner based on simple diff rules.

This README outlines how patching works in Prysm and an explanation of previously
created patches. 

**Given maintaining a patch can be difficult and tedious,
patches are NOT the recommended way of modifying dependencies in Prysm 
unless really needed**

## Table of Contents

- [Prerequisites](#prerequisites)
- [Creating a Patch](#creating-a-patch)
- [Ethereum APIs Patch](#ethereum-apis-patch)
- [Updating Patches](#updating-patches)

## Prerequisites

**Bazel Installation:**
  - The latest release of [Bazel](https://docs.bazel.build/versions/master/install.html)
  - A modern UNIX operating system (MacOS included)

## Creating a Patch

To create a patch, we need an original version of a dependency which we will refer to as `a`
and the patched version referred to as `b`. 

```
cd /tmp
git clone https://github.com/someteam/somerepo a
git clone https://github.com/someteam/somerepo b && cd b
```
Then, make all your changes in `b` and finally create the diff of all your changes as follows:
```
cd ..
diff -ur --exclude=".git" a b > $GOPATH/src/github.com/prysmaticlabs/prysm/third_party/YOURPATCH.patch
```

Next, we need to tell the Bazel [WORKSPACE](https://github.com/prysmaticlabs/prysm/blob/master/WORKSPACE) to patch the specific dependency.
Here's an example for a patch we use today for the [Ethereum APIs](https://github.com/prysmaticlabs/ethereumapis)
dependency:

```
go_repository(
    name = "com_github_prysmaticlabs_ethereumapis",
    commit = "367ca574419a062ae26818f60bdeb5751a6f538",
    patch_args = ["-p1"],
    patches = [
        "//third_party:com_github_prysmaticlabs_ethereumapis-tags.patch",
    ],
    importpath = "github.com/prysmaticlabs/ethereumapis",
)
```

Now, when used in Prysm, the dependency you patched will have the patched modifications
when you run your code.

## Ethereum APIs Patch

As mentioned earlier, patches aren't a recommended approach when needing to modify dependencies
in Prysm save for a few use cases. In particular, all of our public APIs and most canonical
data structures for Prysm are kept in the [Ethereum APIs](https://github.com/prysmaticlabs/ethereumapis) repo.
The purpose of the repo is to serve as a well-documented, well-maintained schema for a full-featured
eth2 API. It is written in protobuf format, and specifies JSON over HTTP mappings as well
as a [Swagger API](https://api.prylabs.network) front-end configuration.

The Prysm repo specifically requires its data structures to have certain struct tags
for serialization purposes as well as other package-related annotations for proper functionality.
Given a protobuf schema is meant to be generic, easily readable, accessible, and language agnostic
(at least for languages which support protobuf generation), it would be wrong for us to include
Go-specific annotations in the Ethereum APIs repo. Instead of maintaining a duplicate of it
within Prysm, we can apply a patch to include those struct tags as needed, while being able
to use the latest changes in the Ethereum APIs repo. This is an appropriate use-case for a patch.

Here's an example:

```
 // The block body of an Ethereum 2.0 beacon block.
 message BeaconBlockBody {
     // The validators RANDAO reveal 96 byte value.
-    bytes randao_reveal = 1;
+    bytes randao_reveal = 1 [(gogoproto.moretags) = "ssz-size:\"96\""];
 
     // A reference to the Ethereum 1.x chain.
     Eth1Data eth1_data = 2;
 
     // 32 byte field of arbitrary data. This field may contain any data and
     // is not used for anything other than a fun message.
-    bytes graffiti = 3; 
+    bytes graffiti = 3 [(gogoproto.moretags) = "ssz-size:\"32\""];
... 
}
```

Above, we're telling Prysm to patch a few lines to include protobuf tags
for SSZ (the serialization library used by Prysm). 

## Updating Patches

Say a new change was pushed out to a dependency you're patching in Prysm. In order to update your
patch or add any other modifications, you would need to:

1. First, clone the repo of the dependency you're patching and the specific commit Prysm is 
currently using for the dependency in the WORKSPACE file. For example, Ethereum APIs could have
the following definition in the WORKSPACE:
```
go_repository(
    name = "com_github_prysmaticlabs_ethereumapis",
    commit = "367ca574419a062ae26818f60bdeb5751a6f538",
    ...
```
Then, checkout that commit
```
git clone https://github.com/prysmaticlabs/ethereumapis && cd ethereumapis
git checkout 367ca574419a062ae26818f60bdeb5751a6f538
```
2. Apply the patch currently in Prysm
```
git apply $GOPATH/src/github.com/prysmaticlabs/prysm/third_party/somepatch.patch
```
3. Resolve any conflicts, commit the changes you want to make

```
git commit -m 'added more changes since patch'
```

4. Generate a new diff by comparing your latest changes to the original commit
the patch was for and output it directly into Prysm's third_party directory
```
git diff 367ca574419a062ae26818f60bdeb5751a6f538 > $GOPATH/src/github.com/prysmaticlabs/prysm/third_party/somepatch.patch
```

5. Build Prysm and ensure tests pass
```
bazel test //...
```
