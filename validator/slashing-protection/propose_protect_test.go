package slashingprotection

import (
	"context"
	"errors"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
)

func TestService_IsSlashableBlock_OK(t *testing.T) {
	ctx := context.Background()
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := privKey.PublicKey()
	validatorDB := dbtest.SetupDB(t, nil)
	slot := uint64(10)
	signedBlock := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:          slot,
			ProposerIndex: 0,
			ParentRoot:    make([]byte, 32),
			StateRoot:     make([]byte, 32),
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: make([]byte, 96),
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot:  make([]byte, 32),
					DepositCount: 0,
					BlockHash:    make([]byte, 32),
				},
				Graffiti: make([]byte, 32),
			},
		},
	}
	dummySigningRoot := [32]byte{}
	copy(dummySigningRoot[:], []byte{1})
	err = validatorDB.SaveProposalHistoryForSlot(ctx, pubKey.Marshal(), slot, dummySigningRoot[:])
	require.NoError(t, err)
	pubKeyBytes := [48]byte{}
	copy(pubKeyBytes[:], pubKey.Marshal())

	srv := &Service{
		validatorDB: validatorDB,
	}
	domainResp := &ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}
	err = srv.IsSlashableBlock(ctx, signedBlock, pubKeyBytes, domainResp)
	assert.Equal(t, true, errors.Is(err, ErrSlashableBlock))

	// Change the slot and now we should not get a slashable block.
	signedBlock.Block.Slot = slot + 1
	err = srv.IsSlashableBlock(ctx, signedBlock, pubKeyBytes, domainResp)
	require.NoError(t, err, "Expected block to not be slashable")
}
