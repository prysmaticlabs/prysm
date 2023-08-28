package backfill

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/network/forks"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

var errInvalidBatchChain = errors.New("parent_root of block does not match root of previous")
var errProposerIndexTooHigh = errors.New("proposer index not present in origin state")
var errUnknownDomain = errors.New("runtime error looking up signing domain for fork")

// VerifiedROBlocks represents a slice of blocks that have passed signature verification.
type VerifiedROBlocks []blocks.ROBlock

type verifier struct {
	// chkptVals is the set of validators from the state used to initialize the node via checkpoint sync.
	keys        [][fieldparams.BLSPubkeyLength]byte
	maxVal      primitives.ValidatorIndex
	vr          []byte
	fsched      forks.OrderedSchedule
	dt          [bls.DomainByteLength]byte
	forkDomains map[[4]byte][]byte
}

func (bs verifier) verify(blks []interfaces.ReadOnlySignedBeaconBlock) (VerifiedROBlocks, error) {
	var err error
	result := make([]blocks.ROBlock, len(blks))
	sigSet := bls.NewSet()
	for i := range blks {
		result[i], err = blocks.NewROBlock(blks[i])
		if err != nil {
			return nil, err
		}
		if i > 0 && result[i-1].Root() != result[i].Block().ParentRoot() {
			p, b := result[i-1], result[i]
			return nil, errors.Wrapf(errInvalidBatchChain,
				"slot %d parent_root=%#x, slot %d root = %#x",
				b.Block().Slot(), b.Block().ParentRoot(),
				p.Block().Slot(), p.Root())
		}
		set, err := bs.blockSignatureBatch(result[i])
		if err != nil {
			return nil, err
		}
		sigSet.Join(set)
	}
	v, err := sigSet.Verify()
	if err != nil {
		return nil, errors.Wrap(err, "block signature verification error")
	}
	if !v {
		return nil, errors.New("batch block signature verification failed")
	}
	return result, nil
}

func (bs verifier) blockSignatureBatch(b blocks.ROBlock) (*bls.SignatureBatch, error) {
	pidx := b.Block().ProposerIndex()
	if pidx > bs.maxVal {
		return nil, errProposerIndexTooHigh
	}
	dom, err := bs.domainAtEpoch(slots.ToEpoch(b.Block().Slot()))
	if err != nil {
		return nil, err
	}
	sig := b.Signature()
	pk := bs.keys[pidx][:]
	root := b.Root()
	rootF := func() ([32]byte, error) { return root, nil }
	return signing.BlockSignatureBatch(pk, sig[:], dom, rootF)
}

func (bs verifier) domainAtEpoch(e primitives.Epoch) ([]byte, error) {
	fork, err := bs.fsched.VersionForEpoch(e)
	if err != nil {
		return nil, err
	}
	d, ok := bs.forkDomains[fork]
	if !ok {
		return nil, errors.Wrapf(errUnknownDomain, "fork version=%#x, epoch=%d", fork, e)
	}
	return d, nil
}

func newBackfillVerifier(st state.BeaconState) (*verifier, error) {
	fsched := forks.NewOrderedSchedule(params.BeaconConfig())
	v := &verifier{
		keys:        st.PublicKeys(),
		vr:          st.GenesisValidatorsRoot(),
		fsched:      fsched,
		dt:          params.BeaconConfig().DomainBeaconProposer,
		forkDomains: make(map[[4]byte][]byte, len(fsched)),
	}
	v.maxVal = primitives.ValidatorIndex(len(v.keys) - 1)
	// Precompute signing domains for known forks at startup.
	for _, entry := range fsched {
		d, err := signing.ComputeDomain(v.dt, entry.Version[:], v.vr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to pre-compute signing domain for fork version=%#x", entry.Version)
		}
		v.forkDomains[entry.Version] = d
	}
	return v, nil
}
