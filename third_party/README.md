# Third Party Package Patching

This directory includes local patches to third party dependencies we use in Prysm. Sometimes,
we need to make a small change to some dependency for ease of use in Prysm without wanting
to maintain our own fork of the dependency ourselves. Our build tool, [Bazel](https://bazel.build)
allows us to include patches in a seamless manner based on simple diff rules.

This README outlines how patching works in Prysm and an explanation of previously
created patches.

## Table of Contents

- [Creating a Patch](#creating-a-patch)
- [Existing Patches](#existing-patches)
    - [Gogo Protobuf](#gogo-protobuf)
    - [Ethereum APIs](#ethereum-apis)
- [Updating Patches](#updating-patches)

## Creating a Patch


## Existing Patches

## Gogo Protobuf

## Ethereum APIs
Prysm can be installed either with Docker **(recommended method)** or using our build tool, Bazel. The below instructions include sections for performing both.

**For Docker installations:**
  - The latest release of [Docker](https://docs.docker.com/install/)

**For Bazel installations:**
  - The latest release of [Bazel](https://docs.bazel.build/versions/master/install.html)
  - A modern UNIX operating system (MacOS included)

## Updating Patches

## Common Errors
