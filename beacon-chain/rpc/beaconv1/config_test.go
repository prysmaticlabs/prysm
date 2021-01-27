package beaconv1

import (
	"context"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestGetDepositContract(t *testing.T) {
	const chainId = 99
	const address = "0x0000000000000000000000000000000000000009"
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.DepositChainID = chainId
	config.DepositContractAddress = address
	params.OverrideBeaconConfig(config)

	s := Server{}
	resp, err := s.GetDepositContract(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	assert.Equal(t, uint64(chainId), resp.Data.ChainId)
	assert.Equal(t, address, resp.Data.Address)
}

func TestForkSchedule_Ok(t *testing.T) {
	genesisForkVersion := []byte("Genesis")
	firstForkVersion, firstForkEpoch := []byte("First"), uint64(100)
	secondForkVersion, secondForkEpoch := []byte("Second"), uint64(200)
	thirdForkVersion, thirdForkEpoch := []byte("Third"), uint64(300)

	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.GenesisForkVersion = genesisForkVersion
	// Create fork schedule adding keys in non-sorted order.
	schedule := make(map[uint64][]byte, 3)
	schedule[secondForkEpoch] = secondForkVersion
	schedule[firstForkEpoch] = firstForkVersion
	schedule[thirdForkEpoch] = thirdForkVersion
	config.ForkVersionSchedule = schedule
	params.OverrideBeaconConfig(config)

	s := &Server{}
	resp, err := s.GetForkSchedule(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	require.Equal(t, 3, len(resp.Data))
	fork := resp.Data[0]
	assert.DeepEqual(t, genesisForkVersion, fork.PreviousVersion)
	assert.DeepEqual(t, firstForkVersion, fork.CurrentVersion)
	assert.Equal(t, firstForkEpoch, fork.Epoch)
	fork = resp.Data[1]
	assert.DeepEqual(t, firstForkVersion, fork.PreviousVersion)
	assert.DeepEqual(t, secondForkVersion, fork.CurrentVersion)
	assert.Equal(t, secondForkEpoch, fork.Epoch)
	fork = resp.Data[2]
	assert.DeepEqual(t, secondForkVersion, fork.PreviousVersion)
	assert.DeepEqual(t, thirdForkVersion, fork.CurrentVersion)
	assert.Equal(t, thirdForkEpoch, fork.Epoch)
}

func TestForkSchedule_NoForks(t *testing.T) {
	s := &Server{}
	resp, err := s.GetForkSchedule(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(resp.Data))
}
