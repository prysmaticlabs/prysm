# Slasher Implementation

This is the main project folder for a slasher implementation for Ethereum written in Go by [Prysmatic Labs](https://prysmaticlabs.com). A slasher listens for all broadcasted messages using a running beacon node in order to detect slashable attestations and block proposals. 
It uses the [min-max-surround](https://github.com/protolambda/eth2-surround#min-max-surround) method by Protolambda.

The slasher requires a connection to a synced beacon node in order to listen for attestations and block proposals. To run the slasher, type:
```
bazel run //slasher -- \
    --datadir PATH/FOR/DB \
    --span-map-cache \
    --beacon-rpc-provider localhost:4000
```

The beacon node entered in `beacon-rpc-provider` will then receive slashings from the slasher client and send them to any requesting proposer to be put into a block. You can read more about configuration options for our slasher in our [documentation portal](https://docs.prylabs.network/docs/prysm-usage/slasher)
