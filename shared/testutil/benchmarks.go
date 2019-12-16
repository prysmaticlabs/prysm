package testutil

import (
	"io/ioutil"

	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// ValidatorCount is for declaring how many validators the benchmarks will be
// performed with. Default is 65536 or 2M ETH staked.
var ValidatorCount = uint64(65536)

// AttestationsPerEpoch represents the requested amount attestations in an epoch.
// This affects the amount of attestations in a fully attested for block and the amount
// of attestations in the state per epoch, so a full 2 epochs should result in twice
// this amount of attestations in the state. Default is 128.
var AttestationsPerEpoch = uint64(128)

// GenesisFileName is the generated genesis beacon state file name.
var GenesisFileName = fmt.Sprintf("benchmark_files/bStateGenesis-%dAtts-%dVals.ssz", AttestationsPerEpoch, ValidatorCount)

// BState1EpochFileName is the generated beacon state after 1 skipped epoch file name.
var BState1EpochFileName = fmt.Sprintf("benchmark_files/bState1Epoch-%dAtts-%dVals.ssz", AttestationsPerEpoch, ValidatorCount)

// BState2EpochFileName is the generated beacon state after 2 full epochs file name.
var BState2EpochFileName = fmt.Sprintf("benchmark_files/bState2Epochs-%dAtts-%dVals.ssz", AttestationsPerEpoch, ValidatorCount)

// FullBlockFileName is the generated full block file name.
var FullBlockFileName = fmt.Sprintf("benchmark_files/fullBlock-%dAtts-%dVals.ssz", AttestationsPerEpoch, ValidatorCount)

func PregenState1Epoch() (*pb.BeaconState, error) {
	path, err := bazel.Runfile(BState1EpochFileName)
	if err != nil {
		return nil, err
	}
	beaconBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	beaconState := &pb.BeaconState{}
	if err := ssz.Unmarshal(beaconBytes, beaconState); err != nil {
		return nil, err
	}
	return beaconState, nil
}

func PreGenState2FullEpochs() (*pb.BeaconState, error) {
	path, err := bazel.Runfile(BState2EpochFileName)
	if err != nil {
		return nil, err
	}
	beaconBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	beaconState := &pb.BeaconState{}
	if err := ssz.Unmarshal(beaconBytes, beaconState); err != nil {
		return nil, err
	}
	return beaconState, nil
}

func PregenFullBlock() (*ethpb.BeaconBlock, error) {
	path, err := bazel.Runfile(FullBlockFileName)
	if err != nil {
		return nil, err
	}
	blockBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	beaconBlock := &ethpb.BeaconBlock{}
	if err := ssz.Unmarshal(blockBytes, beaconBlock); err != nil {
		return nil, err
	}
	return beaconBlock, nil
}
