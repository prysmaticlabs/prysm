package light

import (
	ssz "github.com/ferranbt/fastssz"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/network/forks"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
)

// Precomputed values for generalized indices.
const (
	FinalizedRootIndex              = 105
	FinalizedRootIndexFloorLog2     = 6
	NextSyncCommitteeIndex          = 55
	NextSyncCommitteeIndexFloorLog2 = 5
	PREV_DATA_MAX_SIZE              = 64
)

type Service struct {
	prevHeadData map[[32]byte]*update
}

type update struct {
	header                  *ethpb.BeaconBlockHeader
	finalityCheckpoint      *ethpb.Checkpoint
	finalityBranch          *ssz.Proof
	nextSyncCommittee       *ethpb.SyncCommittee
	nextSyncCommitteeBranch *ssz.Proof
}

type signatureData struct {
	slot          types.Slot
	forkVersion   []byte
	syncAggregate *ethpb.SyncAggregate
}

/**
 * To be called in API route GET /eth/v1/lightclient/best_update/:periods
 */
//async getBestUpdates(periods: SyncPeriod[]): Promise<altair.LightClientUpdate[]> {
//const updates: altair.LightClientUpdate[] = [];
//for (const period of periods) {
//const update = await this.db.bestUpdatePerCommitteePeriod.get(period);
//if (update) updates.push(update);
//}
//return updates;
//}
//
///**
// * To be called in API route GET /eth/v1/lightclient/latest_update_finalized/
// */
//async getLatestUpdateFinalized(): Promise<altair.LightClientUpdate | null> {
//return this.db.latestFinalizedUpdate.get();
//}
//
///**
// * To be called in API route GET /eth/v1/lightclient/latest_update_nonfinalized/
// */
//async getLatestUpdateNonFinalized(): Promise<altair.LightClientUpdate | null> {
//return this.db.latestNonFinalizedUpdate.get();
//}

func (s *Service) onHead(head block.BeaconBlock, postState state.BeaconStateAltair) error {
	innerState, ok := postState.InnerStateUnsafe().(*ethpb.BeaconStateAltair)
	if !ok {
		return errors.New("not altair")
	}
	tr, err := innerState.GetTree()
	if err != nil {
		return err
	}
	header, err := block.BeaconBlockHeaderFromBlockInterface(head)
	if err != nil {
		return err
	}
	finalityBranch, err := tr.Prove(FinalizedRootIndex)
	if err != nil {
		return err
	}
	nextSyncCommitteeBranch, err := tr.Prove(NextSyncCommitteeIndex)
	if err != nil {
		return err
	}
	blkRoot, err := head.HashTreeRoot()
	if err != nil {
		return err
	}
	s.prevHeadData[blkRoot] = &update{
		header:                  header,
		finalityCheckpoint:      innerState.FinalizedCheckpoint,
		finalityBranch:          finalityBranch,
		nextSyncCommittee:       innerState.NextSyncCommittee,
		nextSyncCommitteeBranch: nextSyncCommitteeBranch,
	}
	syncAttestedBlockRoot, err := helpers.BlockRootAtSlot(postState, innerState.Slot-1)
	if err != nil {
		return err
	}

	fork, err := forks.Fork(slots.ToEpoch(head.Slot()))
	if err != nil {
		return err
	}
	syncAggregate, err := head.Body().SyncAggregate()
	if err != nil {
		return err
	}
	sigData := &signatureData{
		slot:          head.Slot(),
		forkVersion:   fork.CurrentVersion,
		syncAggregate: syncAggregate,
	}
	// Recover attested data from prevData cache. If not found, this SyncAggregate is useless
	syncAttestedData, ok := s.prevHeadData[bytesutil.ToBytes32(syncAttestedBlockRoot)]
	if !ok {
		return errors.New("useless")
	}
	commmitteePeriodWithFinalized, err := s.persistBestFinalizedUpdate(syncAttestedData, sigData)
	if err != nil {
		return err
	}

	// Then, store the best non finalized update per period
	if err := s.persistBestNonFinalizedUpdate(syncAttestedData, sigData, commmitteePeriodWithFinalized); err != nil {
		return err
	}
	// Prune old prevHeadData
	if len(s.prevHeadData) > PREV_DATA_MAX_SIZE {
		for k := range s.prevHeadData {
			delete(s.prevHeadData, k)
			if len(s.prevHeadData) <= PREV_DATA_MAX_SIZE {
				break
			}
		}
	}
	return nil
}

func (s *Service) persistBestFinalizedUpdate(data *update, sigData *signatureData) (types.Epoch, error) {
	return 0, nil
}

func (s Service) persistBestNonFinalizedUpdate(data *update, sigData *signatureData, period types.Epoch) error {
	return nil
}
