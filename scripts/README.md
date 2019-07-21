# Bash Scripts

This subproject contains useful bash scripts for working with our repository. We have a simple tool that outputs coverage, a simple tool to check for gazelle requirements, and visibility rules tools for Bazel packages.

### Instructions to run a single beacon chain node and 8 validators locally using the scripts.

1. Ensure your private key path is correct in all the files below.

2. Run `./deploy-deposit-contract.sh`

3. Put the resulting contract address in `start-beacon-chain.sh` and `setup-8-validators.sh`.

4. Run `./start-beacon-chain.sh`

5. Run `./setup-8-validators.sh`

6. You can use `tail -f /tmp/data/validator#.log` with # as a number from 1 - 8 to view the output of the validators.
