# Ethereum 2.0 E2E Test Suite

This is a test-suite for conformity end-2-end tests for Prysm's implementation of the Ethereum 2.0 specification. Implementation teams have decided to utilize YAML as a general conformity test format for the current beacon chain's runtime functionality.

The test suite opts for YAML due to wide language support and support for inline comments.

## Testing Format

The testing format follows the official ETH2.0 Specification created [here](https://github.com/ethereum/eth2.0-specs/blob/master/specs/test-format.md)

### Core, Chain Tests

Chain tests check for conformity of a certain client to the beacon chain specification for items such as the fork choice rule and Casper FFG validator rewards & penalties. Stateful tests need to specify a certain configuration of a beacon chain, with items such as the number validators, in the YAML file. Sample tests will all required fields are shown below.

**State Transition Tests**

The most important use case for this test format is to verify the ins and outs of the Ethereum Phase 0 Beacon Chain:

TODO


### Stateless Tests

Stateless tests represent simple unit test definitions for important invariants in the ETH2.0 runtime. In particular, these test conformity across clients with respect to items such as Simple Serialize (SSZ), Signature Aggregation (BLS), and Validator Shuffling

**Simple Serialize**

TODO

**Signature Aggregation**

TODO

**Validator Shuffling**

```yaml
title: Shuffling Algorithm Tests
summary: Test vectors for shuffling a list based upon a seed using `shuffle`
test_suite: shuffle
fork: tchaikovsky
version: 1.0

test_cases:
- input: []
  output: []
  seed: !!binary ""
- name: boring_list
  description: List with a single element, 0
  input: [0]
  output: [0]
  seed: !!binary ""
- input: [255]
  output: [255]
  seed: !!binary ""
- input: [4, 6, 2, 6, 1, 4, 6, 2, 1, 5]
  output: [1, 6, 4, 1, 6, 6, 2, 2, 4, 5]
  seed: !!binary ""
- input: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13]
  output: [4, 7, 10, 13, 3, 1, 2, 9, 12, 6, 11, 8, 5]
  seed: !!binary ""
- input: [65, 6, 2, 6, 1, 4, 6, 2, 1, 5]
  output: [6, 65, 2, 5, 4, 2, 6, 6, 1, 1]
  seed: !!binary |
    JlAYJ5H2j8g7PLiPHZI/rTS1uAvKiieOrifPN6Moso0=
```

## Using the Runner

First, create a directory containing the YAML files you wish to test (or use the default `./sampletests` directory included with Prysm). Then, navigate to the test runner's directory and use the go tool as follows:

```bash
go run main.go -tests-dir /path/to/your/yamlfiles
```

The runner will then start up a simulated backend and run all your specified YAML tests.

```bash
[2018-11-06 15:01:44]  INFO ----Running Chain Tests----
[2018-11-06 15:01:44]  INFO Running 4 YAML Tests
[2018-11-06 15:01:44]  INFO Title: Sample Ethereum 2.0 Beacon Chain Test
[2018-11-06 15:01:44]  INFO Summary: Basic, functioning fork choice rule for Ethereum 2.0
[2018-11-06 15:01:44]  INFO Test Suite: prysm
[2018-11-06 15:01:44]  INFO Test Runs Finished In: 0.000643545 Seconds
```
