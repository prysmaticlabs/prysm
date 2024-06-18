#!/bin/bash

# Script to update mock files after proto/prysm/v1alpha1/services.proto changes.
# Use a space to separate mock destination from its interfaces.
# Be sure to install mockgen before use: https://github.com/uber-go/mock

# github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1
# ------------------------------------------------------
mock_path="testing/mock"
mocks=(
      "$mock_path/beacon_service_mock.go BeaconChainClient"
      "$mock_path/beacon_validator_server_mock.go BeaconNodeValidatorServer,BeaconNodeValidator_WaitForActivationServer,BeaconNodeValidator_WaitForChainStartServer,BeaconNodeValidator_StreamSlotsServer"
      "$mock_path/beacon_validator_client_mock.go BeaconNodeValidatorClient,BeaconNodeValidator_WaitForChainStartClient,BeaconNodeValidator_WaitForActivationClient,BeaconNodeValidator_StreamSlotsClient"
      "$mock_path/node_service_mock.go NodeClient"
)

for ((i = 0; i < ${#mocks[@]}; i++)); do
    file=${mocks[i]% *};
    interfaces=${mocks[i]#* };
    echo "generating $file for interfaces: $interfaces";
    echo
    GO11MODULE=on mockgen -package=mock -destination="$file" github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1 "$interfaces"
done

goimports -w "$mock_path/."
gofmt -s -w "$mock_path/."

# github.com/prysmaticlabs/prysm/v5/validator/client/iface
# --------------------------------------------------------
mock_path="testing/validator-mock"
mocks=(
      "$mock_path/chain_client_mock.go ChainClient"
      "$mock_path/prysm_chain_client_mock.go PrysmChainClient"
      "$mock_path/node_client_mock.go NodeClient"
      "$mock_path/validator_client_mock.go ValidatorClient"
)

for ((i = 0; i < ${#mocks[@]}; i++)); do
    file=${mocks[i]% *};
    interfaces=${mocks[i]#* };
    echo "generating $file for interfaces: $interfaces";
    GO11MODULE=on mockgen -package=validator_mock -destination="$file" github.com/prysmaticlabs/prysm/v5/validator/client/iface "$interfaces"
done

goimports -w "$mock_path/."
gofmt -s -w "$mock_path/."

# github.com/prysmaticlabs/prysm/v5/validator/client/beacon-api
# -------------------------------------------------------------
mock_path="validator/client/beacon-api/mock"
mocks=(
      "$mock_path/genesis_mock.go genesis.go"
      "$mock_path/duties_mock.go duties.go"
      "$mock_path/json_rest_handler_mock.go json_rest_handler.go"
      "$mock_path/state_validators_mock.go state_validators.go"
      "$mock_path/beacon_block_converter_mock.go beacon_block_converter.go"
)

for ((i = 0; i < ${#mocks[@]}; i++)); do
    file=${mocks[i]% *};
    source=${mocks[i]#* };
    echo "generating $file for file: $source";
    GO11MODULE=on mockgen -package=mock -source="validator/client/beacon-api/$source" -destination="$file"
done

goimports -w "$mock_path/."
gofmt -s -w "$mock_path/."

# github.com/prysmaticlabs/prysm/v5/crypto/bls
# --------------------------------------------
mock_path="crypto/bls/common/mock"
mocks=(
      "$mock_path/interface_mock.go interface.go"
)

for ((i = 0; i < ${#mocks[@]}; i++)); do
    file=${mocks[i]% *};
    source=${mocks[i]#* };
    echo "generating $file for file: $source";
    GO11MODULE=on mockgen -package=mock -source="crypto/bls/common/$source" -destination="$file"
done

goimports -w "$mock_path/."
gofmt -s -w "$mock_path/."

# github.com/prysmaticlabs/prysm/v5/api/client/beacon
# -------------------------------------------------------------
mock_path="api/client/beacon/mock"
mocks=(
      "$mock_path/health_mock.go health.go"
)

for ((i = 0; i < ${#mocks[@]}; i++)); do
    file=${mocks[i]% *};
    source=${mocks[i]#* };
    echo "generating $file for file: $source";
    GO11MODULE=on mockgen -package=mock -source="api/client/beacon/iface/$source" -destination="$file"
done

goimports -w "$mock_path/."
gofmt -s -w "$mock_path/."