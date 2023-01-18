#!/bin/bash

# Script to update mock files after proto/prysm/v1alpha1/services.proto changes.
# Use a space to separate mock destination from its interfaces.

mock_path="testing/mock"

# github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1
# ------------------------------------------------------
proto_mocks_v1alpha1=(
      "$mock_path/beacon_service_mock.go BeaconChainClient,BeaconChain_StreamChainHeadClient,BeaconChain_StreamAttestationsClient,BeaconChain_StreamBlocksClient,BeaconChain_StreamValidatorsInfoClient,BeaconChain_StreamIndexedAttestationsClient"
      "$mock_path/beacon_chain_service_mock.go BeaconChain_StreamChainHeadServer,BeaconChain_StreamAttestationsServer,BeaconChain_StreamBlocksServer,BeaconChain_StreamValidatorsInfoServer,BeaconChain_StreamIndexedAttestationsServer"
      "$mock_path/beacon_validator_server_mock.go BeaconNodeValidatorServer,BeaconNodeValidator_WaitForActivationServer,BeaconNodeValidator_WaitForChainStartServer,BeaconNodeValidator_StreamDutiesServer"
      "$mock_path/beacon_validator_client_mock.go BeaconNodeValidatorClient,BeaconNodeValidator_WaitForChainStartClient,BeaconNodeValidator_WaitForActivationClient,BeaconNodeValidator_StreamDutiesClient"
      "$mock_path/slasher_client_mock.go SlasherClient"
      "$mock_path/node_service_mock.go NodeClient"
)

for ((i = 0; i < ${#proto_mocks_v1alpha1[@]}; i++)); do
    file=${proto_mocks_v1alpha1[i]% *};
    interfaces=${proto_mocks_v1alpha1[i]#* };
    echo "generating $file for interfaces: $interfaces";
    echo
    GO11MODULE=on mockgen -package=mock -destination="$file" github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1 "$interfaces"
done

# github.com/prysmaticlabs/prysm/v3/proto/eth/service
# ---------------------------------------------------
proto_mocks_service=(
      "$mock_path/event_service_mock.go EventsClient,Events_StreamEventsClient,Events_StreamEventsServer"
)

for ((i = 0; i < ${#proto_mocks_service[@]}; i++)); do
    file=${proto_mocks_service[i]% *};
    interfaces=${proto_mocks_service[i]#* };
    echo "generating $file for interfaces: $interfaces";
    echo
    GO11MODULE=on mockgen -package=mock -destination="$file" github.com/prysmaticlabs/prysm/v3/proto/eth/service "$interfaces"
done

# github.com/prysmaticlabs/prysm/proto/v3/prysm/v1alpha1/validator-client
# -----------------------------------------------------------------------
proto_mocks_v1alpha1_validator_clients=(
    "$mock_path/keymanager_mock.go RemoteSignerClient"
)

for ((i = 0; i < ${#proto_mocks_v1alpha1_validator_clients[@]}; i++)); do
    file=${proto_mocks_v1alpha1_validator_clients[i]% *};
    interfaces=${proto_mocks_v1alpha1_validator_clients[i]#* };
    echo "generating $file for interfaces: $interfaces";
    echo
    GO11MODULE=on mockgen -package=mock -destination="$file" github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client "$interfaces"
done

# github.com/prysmaticlabs/prysm/v3/validator/client/iface
# --------------------------------------------------------
iface_mocks=(
      "$mock_path/validator_client_mock.go ValidatorClient"
)

for ((i = 0; i < ${#iface_mocks[@]}; i++)); do
    file=${iface_mocks[i]% *};
    interfaces=${iface_mocks[i]#* };
    echo "generating $file for interfaces: $interfaces";
    GO11MODULE=on mockgen -package=mock -destination="$file" github.com/prysmaticlabs/prysm/v3/validator/client/iface "$interfaces"
done

goimports -w "$mock_path/."
gofmt -s -w "$mock_path/."

# github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api
# -------------------------------------------------------------
beacon_api_mock_path="validator/client/beacon-api/mock"
beacon_api_mocks=(
      "$beacon_api_mock_path/genesis_mock.go genesis.go"
      "$beacon_api_mock_path/duties_mock.go duties.go"
      "$beacon_api_mock_path/json_rest_handler_mock.go json_rest_handler.go"
      "$beacon_api_mock_path/state_validators_mock.go state_validators.go"
)

for ((i = 0; i < ${#beacon_api_mocks[@]}; i++)); do
    file=${beacon_api_mocks[i]% *};
    source=${beacon_api_mocks[i]#* };
    echo "generating $file for file: $source";
    GO11MODULE=on mockgen -package=mock -source="validator/client/beacon-api/$source" -destination="$file"
done

goimports -w "$beacon_api_mock_path/."
gofmt -s -w "$beacon_api_mock_path/."
