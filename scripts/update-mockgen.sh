#!/bin/bash

# Script to update mock files after proto/beacon/rpc/v1/services.proto changes.
# Use a space to separate mock destination from its interfaces.

mocks=("./validator/internal/attester_service_mock.go AttesterServiceClient"
       "./validator/internal/beacon_service_mock.go BeaconServiceClient,BeaconService_LatestAttestationClient,BeaconService_WaitForChainStartClient"
       "./validator/internal/proposer_service_mock.go ProposerServiceClient"
       "./validator/internal/validator_service_mock.go ValidatorServiceClient")

for ((i = 0; i < ${#mocks[@]}; i++)); do
    file=${mocks[i]% *};
    interfaces=${mocks[i]#* };
    echo "generating $file for interfaces: $interfaces";
    mockgen -package=internal -destination=$file github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1 $interfaces
done