package rewards

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/testutil"
	mockstategen "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen/mock"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
)

func TestBlockRewards(t *testing.T) {
	valCount := 64

	st, err := util.NewBeaconStateAltair()
	require.NoError(t, st.SetSlot(1))
	require.NoError(t, err)
	validators := make([]*eth.Validator, 0, valCount)
	balances := make([]uint64, 0, valCount)
	keys := make([]bls.SecretKey, 0, valCount)
	for i := 0; i < valCount; i++ {
		blsKey, err := bls.RandKey()
		require.NoError(t, err)
		keys = append(keys, blsKey)
		validators = append(validators, &eth.Validator{
			PublicKey:         blsKey.PublicKey().Marshal(),
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
		})
		balances = append(balances, params.BeaconConfig().MaxEffectiveBalance)
	}
	require.NoError(t, st.SetValidators(validators))
	require.NoError(t, st.SetBalances(balances))
	require.NoError(t, st.SetCurrentParticipationBits(make([]byte, valCount)))

	b := util.NewBeaconBlockAltair()
	b.Block.Slot = 2
	// we have to set the proposer index to the value that will be randomly chosen (fortunately it's deterministic)
	b.Block.ProposerIndex = 12
	b.Block.Body.Attestations = []*eth.Attestation{
		{
			AggregationBits: bitfield.Bitlist{0b00000111},
			Data:            util.HydrateAttestationData(&eth.AttestationData{}),
		},
		{
			AggregationBits: bitfield.Bitlist{0b00000111},
			Data:            util.HydrateAttestationData(&eth.AttestationData{}),
		},
	}
	attData1 := util.HydrateAttestationData(&eth.AttestationData{BeaconBlockRoot: bytesutil.PadTo([]byte("root1"), 32)})
	attData2 := util.HydrateAttestationData(&eth.AttestationData{BeaconBlockRoot: bytesutil.PadTo([]byte("root2"), 32)})
	domain, err := signing.Domain(st.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, st.GenesisValidatorsRoot())
	require.NoError(t, err)
	sigRoot1, err := signing.ComputeSigningRoot(attData1, domain)
	require.NoError(t, err)
	sigRoot2, err := signing.ComputeSigningRoot(attData2, domain)
	require.NoError(t, err)
	b.Block.Body.AttesterSlashings = []*eth.AttesterSlashing{
		{
			Attestation_1: &eth.IndexedAttestation{
				AttestingIndices: []uint64{0},
				Data:             attData1,
				Signature:        keys[0].Sign(sigRoot1[:]).Marshal(),
			},
			Attestation_2: &eth.IndexedAttestation{
				AttestingIndices: []uint64{0},
				Data:             attData2,
				Signature:        keys[0].Sign(sigRoot2[:]).Marshal(),
			},
		},
	}
	header1 := &eth.BeaconBlockHeader{
		Slot:          0,
		ProposerIndex: 1,
		ParentRoot:    bytesutil.PadTo([]byte("root1"), 32),
		StateRoot:     bytesutil.PadTo([]byte("root1"), 32),
		BodyRoot:      bytesutil.PadTo([]byte("root1"), 32),
	}
	header2 := &eth.BeaconBlockHeader{
		Slot:          0,
		ProposerIndex: 1,
		ParentRoot:    bytesutil.PadTo([]byte("root2"), 32),
		StateRoot:     bytesutil.PadTo([]byte("root2"), 32),
		BodyRoot:      bytesutil.PadTo([]byte("root2"), 32),
	}
	domain, err = signing.Domain(st.Fork(), 0, params.BeaconConfig().DomainBeaconProposer, st.GenesisValidatorsRoot())
	require.NoError(t, err)
	sigRoot1, err = signing.ComputeSigningRoot(header1, domain)
	require.NoError(t, err)
	sigRoot2, err = signing.ComputeSigningRoot(header2, domain)
	require.NoError(t, err)
	b.Block.Body.ProposerSlashings = []*eth.ProposerSlashing{
		{
			Header_1: &eth.SignedBeaconBlockHeader{
				Header:    header1,
				Signature: keys[1].Sign(sigRoot1[:]).Marshal(),
			},
			Header_2: &eth.SignedBeaconBlockHeader{
				Header:    header2,
				Signature: keys[1].Sign(sigRoot2[:]).Marshal(),
			},
		},
	}

	sbb, err := blocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	mockChainService := &mock.ChainService{}
	s := &Server{
		BlockFetcher:          &testutil.MockBlockFetcher{BlockToReturn: sbb},
		OptimisticModeFetcher: mockChainService,
		FinalizationFetcher:   mockChainService,
		ReplayerBuilder:       mockstategen.NewMockReplayerBuilder(mockstategen.WithMockState(st)),
	}

	t.Run("ok", func(t *testing.T) {
		url := "http://only.the.slot.number.at.the.end.is.important/0"
		request := httptest.NewRequest("GET", url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.BlockRewards(writer, request)
	})
}
