#!/bin/bash

# Script to update mock files after proto/beacon/rpc/v1/services.proto changes.

mockgen -package=internal -destination=./validator/internal/attester_service_mock.go github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1 AttesterServiceClient
mockgen -package=internal -destination=./validator/internal/beacon_service_mock.go github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1 BeaconServiceClient,BeaconService_LatestAttestationClient,BeaconService_WaitForChainStartClient
mockgen -package=internal -destination=./validator/internal/proposer_service_mock.go github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1 ProposerServiceClient
mockgen -package=internal -destination=./validator/internal/validator_service_mock.go github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1 ValidatorServiceClient