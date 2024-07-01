package electra_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
	consensusblocks "github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestVerifyOperationLengths_Electra(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		s, _ := util.DeterministicGenesisStateElectra(t, 1)
		sb, err := consensusblocks.NewSignedBeaconBlock(util.NewBeaconBlockElectra())
		require.NoError(t, err)
		require.NoError(t, electra.VerifyBlockDepositLength(sb.Block().Body(), s))
	})
	t.Run("eth1depositIndex less than eth1depositIndexLimit & number of deposits incorrect", func(t *testing.T) {
		s, _ := util.DeterministicGenesisStateElectra(t, 1)
		sb, err := consensusblocks.NewSignedBeaconBlock(util.NewBeaconBlockElectra())
		require.NoError(t, err)
		require.NoError(t, s.SetEth1DepositIndex(0))
		require.NoError(t, s.SetDepositRequestsStartIndex(1))
		err = electra.VerifyBlockDepositLength(sb.Block().Body(), s)
		require.ErrorContains(t, "incorrect outstanding deposits in block body", err)
	})
	t.Run("eth1depositIndex more than eth1depositIndexLimit & number of deposits is not 0", func(t *testing.T) {
		s, _ := util.DeterministicGenesisStateElectra(t, 1)
		sb, err := consensusblocks.NewSignedBeaconBlock(util.NewBeaconBlockElectra())
		require.NoError(t, err)
		sb.SetDeposits([]*ethpb.Deposit{
			{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte{1, 2, 3},
					Amount:                1000,
					WithdrawalCredentials: make([]byte, common.AddressLength),
					Signature:             []byte{4, 5, 6},
				},
			},
		})
		require.NoError(t, s.SetEth1DepositIndex(1))
		require.NoError(t, s.SetDepositRequestsStartIndex(1))
		err = electra.VerifyBlockDepositLength(sb.Block().Body(), s)
		require.ErrorContains(t, "incorrect outstanding deposits in block body", err)
	})
}
