package epoch_test

import (
	"context"
	"fmt"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var ValidatorCount = 8192
var RunAmount = 67108864 / ValidatorCount

func BenchmarkProcessEth1Data(b *testing.B) {
	deposits := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}
	requiredVoteCount := params.BeaconConfig().EpochsPerEth1VotingPeriod *
		params.BeaconConfig().SlotsPerEpoch
	beaconState.Slot = 15 * params.BeaconConfig().SlotsPerEpoch
	beaconState.LatestEth1Data = &pb.Eth1Data{
		DepositRootHash32: nil,
		BlockHash32:       nil,
	}
	beaconState.Eth1DataVotes = []*pb.Eth1DataVote{
		{
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte{'A'},
				BlockHash32:       []byte{'B'},
			},
			VoteCount: 0,
		},
		{
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte{'C'},
				BlockHash32:       []byte{'D'},
			},
			VoteCount: requiredVoteCount/2 + 1,
		},
		{
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte{'E'},
				BlockHash32:       []byte{'F'},
			},
			VoteCount: requiredVoteCount / 2,
		},
	}

	b.ResetTimer()
	b.N = RunAmount
	for i := 0; i < b.N; i++ {
		_ = epoch.ProcessEth1Data(context.Background(), beaconState)
	}
}

func BenchmarkProcessCrosslinks(b *testing.B) {
	deposits := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}
	beaconState.Slot = params.BeaconConfig().GenesisSlot + 5*params.BeaconConfig().SlotsPerEpoch

	byteLength := int(uint64(ValidatorCount) / params.BeaconConfig().TargetCommitteeSize / 8 / 8)
	var participationBitfield []byte
	for i := 0; i < byteLength; i++ {
		participationBitfield = append(participationBitfield, byte(0xff))
	}

	var attestations []*pb.PendingAttestation
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                    beaconState.Slot,
				CrosslinkDataRootHash32: []byte{'A'},
			},
			// All validators attested to the above roots.
			AggregationBitfield: participationBitfield,
		}
		attestations = append(attestations, attestation)
	}

	b.N = 10
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := epoch.ProcessCrosslinks(
			context.Background(),
			beaconState,
			attestations,
			nil,
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func setupBenchmarkInitialDeposits(numDeposits int) []*pb.Deposit {
	setBenchmarkConfig()
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		depositInput := &pb.DepositInput{
			Pubkey:                      []byte(fmt.Sprintf("%d", i)),
			WithdrawalCredentialsHash32: []byte{1, 2, 3},
		}
		balance := params.BeaconConfig().MaxDepositAmount
		depositData, err := helpers.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			panic(err)
		}
		deposits[i] = &pb.Deposit{
			DepositData: depositData,
		}
	}
	return deposits
}

func setBenchmarkConfig() {
	c := params.BeaconConfig()
	// From Danny Ryan's "Minimal Config"
	// c.SlotsPerEpoch = 8
	// c.MinAttestationInclusionDelay = 2
	// c.TargetCommitteeSize = 4
	// c.GenesisEpoch = c.GenesisSlot / 8
	// c.LatestRandaoMixesLength = 64
	// c.LatestActiveIndexRootsLength = 64
	// c.LatestSlashedExitLength = 64
	params.OverrideBeaconConfig(c)
}
