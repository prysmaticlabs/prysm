#!/bin/bash

# Script to update mock files after proto/beacon/rpc/v1/services.proto changes.
# Use a space to separate mock destination from its interfaces.

mock_path="shared/van_mock"
mocks=(
      "$mock_path/van_beacon_chain_service_mock.go BeaconChain_StreamChainHeadServer,BeaconChain_StreamAttestationsServer,BeaconChain_StreamBlocksServer,BeaconChain_StreamValidatorsInfoServer,BeaconChain_StreamIndexedAttestationsServer,BeaconChain_StreamNewPendingBlocksServer,BeaconChain_StreamMinimalConsensusInfoServer"
)

for ((i = 0; i < ${#mocks[@]}; i++)); do
    file=${mocks[i]% *};
    interfaces=${mocks[i]#* };
    echo "generating $file for interfaces: $interfaces";
    GO11MODULE=on mockgen -package=van_mock -destination=$file github.com/prysmaticlabs/ethereumapis/eth/v1alpha1 $interfaces
done

goimports -w "$mock_path/."
gofmt -s -w "$mock_path/."
