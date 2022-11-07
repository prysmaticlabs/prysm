#!/bin/bash

# Script to update mock files after proto/prysm/v1alpha1/services.proto changes.
# Use a space to separate mock destination from its interfaces.

mock_path="testing/mock"
proto_mocks=(
      "$mock_path/beacon_service_mock.go BeaconChainClient,BeaconChain_StreamChainHeadClient,BeaconChain_StreamAttestationsClient,BeaconChain_StreamBlocksClient,BeaconChain_StreamValidatorsInfoClient,BeaconChain_StreamIndexedAttestationsClient"
      "$mock_path/beacon_chain_service_mock.go BeaconChain_StreamChainHeadServer,BeaconChain_StreamAttestationsServer,BeaconChain_StreamBlocksServer,BeaconChain_StreamValidatorsInfoServer,BeaconChain_StreamIndexedAttestationsServer"
      "$mock_path/beacon_validator_server_mock.go BeaconNodeValidatorServer,BeaconNodeValidator_WaitForActivationServer,BeaconNodeValidator_WaitForChainStartServer,BeaconNodeValidator_StreamDutiesServer"
      "$mock_path/beacon_validator_client_mock.go BeaconNodeValidatorClient,BeaconNodeValidator_WaitForChainStartClient,BeaconNodeValidator_WaitForActivationClient,BeaconNodeValidator_StreamDutiesClient"
      "$mock_path/slasher_client_mock.go SlasherClient"
      "$mock_path/event_service_mock.go EventsClient,Events_StreamEventsClient,Events_StreamEventsServer"
      "$mock_path/node_service_mock.go NodeClient"
      "$mock_path/keymanager_mock.go RemoteSignerClient"
)

iface_mocks=(
      "$mock_path/validator_client_mock.go ValidatorClient"
)

for ((i = 0; i < ${#proto_mocks[@]}; i++)); do
    file=${proto_mocks[i]% *};
    interfaces=${proto_mocks[i]#* };
    echo "generating $file for interfaces: $interfaces";
    GO11MODULE=on mockgen -package=mock -destination="$file" github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1 "$interfaces"
done

for ((i = 0; i < ${#iface_mocks[@]}; i++)); do
    file=${iface_mocks[i]% *};
    interfaces=${iface_mocks[i]#* };
    echo "generating $file for interfaces: $interfaces";
    GO11MODULE=on mockgen -package=mock -destination="$file" github.com/prysmaticlabs/prysm/v3/validator/client/iface "$interfaces"
done

goimports -w "$mock_path/."
gofmt -s -w "$mock_path/."

