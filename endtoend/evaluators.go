package main

import (
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// Evaluator defines the function signature for function to run during the E2E.
type Evaluator func(client *eth.BeaconChainClient) error
