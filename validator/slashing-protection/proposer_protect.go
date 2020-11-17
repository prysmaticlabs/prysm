package slashingprotection

//import (
//	"bytes"
//	"context"
//
//	"github.com/pkg/errors"
//	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
//
//	"github.com/prysmaticlabs/prysm/shared/params"
//	"github.com/prysmaticlabs/prysm/validator/db"
//)
//
//var SlashableBlockErr = errors.New("attempted to sign a double proposal, block rejected by slashing protection")
//
//func (lp *Service) IsSlashableBlock(
//	ctx context.Context, header *ethpb.BeaconBlockHeader, pubKey [48]byte,
//) error {
//	signingRoot, err := lp.db.ProposalHistoryForSlot(ctx, pubKey[:], header.Slot)
//	if err != nil {
//		return errors.Wrap(err, "failed to get proposal history")
//	}
//	// If the bit for the current slot is marked, do not propose.
//	if !bytes.Equal(signingRoot, params.BeaconConfig().ZeroHash[:]) {
//		return SlashableBlockErr
//	}
//	return nil
//}

func hi() {}
