package backfill

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/network/forks"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

// VerifiedROBlocks represents a slice of blocks that have passed signature verification.
type VerifiedROBlocks []blocks.ROBlock

type backfillValidator struct {
	// chkptVals is the set of validators from the state used to initialize the node via checkpoint sync.
	keys   [][fieldparams.BLSPubkeyLength]byte
	vr     []byte
	fsched forks.OrderedSchedule
	dt     [bls.DomainByteLength]byte
}

func (bs backfillValidator) validateBatch(blocks []blocks.ROBlock) ([32]byte, VerifiedROBlocks, error) {
	sigSet := bls.NewSet()
	var tail [32]byte
	for i := range blocks {
		b := blocks[i]
		set, err := bs.blockSignatureBatch(b)
		if err != nil {
			return tail, nil, err
		}
		sigSet.Join(set)
	}
	v, err := sigSet.Verify()
	if err != nil {
		return tail, nil, err
	}
	if !v {
		return [32]byte{}, nil, errors.New("batch block signature verification failed")
	}
	return tail, blocks, nil
}

func (bs backfillValidator) blockSignatureBatch(b blocks.ROBlock) (*bls.SignatureBatch, error) {
	root := b.Root()
	rootFunc := func() ([32]byte, error) { return root, nil }
	sig := b.Signature()
	pk := bs.keys[b.Block().ProposerIndex()][:]
	fork, err := bs.fsched.VersionForEpoch(slots.ToEpoch(b.Block().Slot()))
	if err != nil {
		return nil, err
	}
	domain, err := signing.ComputeDomain(bs.dt, fork[:], bs.vr)
	if err != nil {
		return nil, err
	}
	return signing.BlockSignatureBatch(pk, sig[:], domain, rootFunc)
}

func newBackfillVerifier(st state.BeaconState, genValRoot [32]byte) *backfillValidator {
	return &backfillValidator{
		keys:   st.PublicKeys(),
		vr:     genValRoot[:],
		fsched: forks.NewOrderedSchedule(params.BeaconConfig()),
		dt:     params.BeaconConfig().DomainBeaconProposer,
	}
}
