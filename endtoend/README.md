# End-to-end Testing Package

This is the main project folder of the end-to-end testing suite for Prysm. This performs a full end-to-end test for Prysm, including spinning up an ETH1 dev chain, sending deposits to the deposit contract, and making sure the beacon node and its validators are running and performing properly for a few epochs.
It also performs a test on a syncing node, and supports featureflags to allow easy E2E testing of experimental features. 

## How it works
Through the `end2EndConfig` struct, you can declare several options such as how many epochs the test should run for, and what `BeaconConfig` the test should use. You can also declare how many beacon nodes and validator clients are run, the E2E will automatically divide the validators evently among the beacon nodes.

In order to "evaluate" the state of the beacon chain while the E2E is running, there are `Evaluators`  that use the beacon chain node API to determine if the network is performing as it should. This can evaluate for conditions like validator activation, finalization, validator participation and more.

Evaluators have 3 parts, the name for it's test name, a `policy` which declares which epoch(s) the evaluator should run, and then the `evaluation` which uses the beacon chain API to determine if the beacon chain passes certain conditions like finality.

## Current end-to-end tests
* Minimal Config - 4 beacon nodes, 64 validators, running for 6 epochs
* ~~Mainnet Config - 2 beacon nodes, 16,384 validators, running for 5 epochs~~ Disabled for now

## Instructions
If you wish to run all the E2E tests, you can run them through bazel with:

```
bazel test //endtoend:go_default_test --test_output=streamed --test_arg=-test.v --nocache_test_results
```

To run the anti-flake E2E tests, run:
```
bazel test //endtoend:go_default_test --test_output=streamed --test_filter=TestEndToEnd_AntiFlake_MinimalConfig --test_arg=-test.v --nocache_test_results
```