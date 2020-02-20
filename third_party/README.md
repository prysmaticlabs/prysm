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

Say we want to update Ethereum APIs in Prysm to its latest master commit `b7452dde4ca361809def4ed5924ab3cb7ad1299a`.
Here are the steps:

1. Go to your Prysm WORKSPACE and look at the commit in there for Ethereum APIs, say it's `e6f60041667fbc3edb22b03735ec111d1a40cd0e`
2. Go to Ethereum APIs and do `git checkout e6f60041667fbc3edb22b03735ec111d1a40cd0e`
3. In the Ethereum APIs repo, do `git apply $GOPATH/src/github.com/prysmaticlabs/prysm/third_party/com_github_prysmaticlabs_ethereumapis-tags.patch`
4. Make any changes you want to make in Ethereum APIs, such as applying ssz struct tags, etc.
5. In the Ethereum APIs repo, do `git commit -m "applied patch and changes"`
6. Do `git merge master`
7. Generate a new diff and update the diff in Prysm `git diff b7452dde4ca361809def4ed5924ab3cb7ad1299a > $GOPATH/src/github.com/prysmaticlabs/prysm/third_party/com_github_prysmaticlabs_ethereumapis-tags.patch`
8. Update the commit in the Prysm WORKSPACE file for Ethereum APIs to `b7452dde4ca361809def4ed5924ab3cb7ad1299a`
9. Build the Prysm project
