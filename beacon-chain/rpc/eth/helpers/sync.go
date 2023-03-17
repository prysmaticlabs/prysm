package helpers

import (
	"context"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/grpc"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/statefetcher"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ValidateSync checks whether the node is currently syncing and returns an error if it is.
// It also appends syncing info to gRPC headers.
func ValidateSync(
	ctx context.Context,
	syncChecker sync.Checker,
	headFetcher blockchain.HeadFetcher,
	timeFetcher blockchain.TimeFetcher,
	optimisticModeFetcher blockchain.OptimisticModeFetcher,
) error {
	if !syncChecker.Syncing() {
		return nil
	}
	headSlot := headFetcher.HeadSlot()
	isOptimistic, err := optimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "Could not check optimistic status: %v", err)
	}

	syncDetailsContainer := &syncDetailsContainer{
		SyncDetails: &SyncDetailsJson{
			HeadSlot:     strconv.FormatUint(uint64(headSlot), 10),
			SyncDistance: strconv.FormatUint(uint64(timeFetcher.CurrentSlot()-headSlot), 10),
			IsSyncing:    true,
			IsOptimistic: isOptimistic,
		},
	}

	err = grpc.AppendCustomErrorHeader(ctx, syncDetailsContainer)
	if err != nil {
		return status.Errorf(
			codes.Internal,
			"Syncing to latest head, not ready to respond. Could not prepare sync details: %v",
			err,
		)
	}
	return status.Error(codes.Unavailable, "Syncing to latest head, not ready to respond")
}

// IsOptimistic checks whether the beacon state's block is optimistic. The algorithm works as follows:
//   - For head, return the node's optimistic status.
//   - For genesis and finalized states, immediately return false.
//   - For justified state and a state root:
//     # If state's slot is equal to head's slot, return the node's optimistic status.
//     # If state's block is not canonical, return true.
//     # If state's block is canonical and node is not optimistic, return false.
//     # If state's block is canonical and node is optimistic and state's slot is not after head's finalized slot, return false.
//     # Otherwise return true.
//   - For slot number:
//     # If node is not optimistic, return false.
//     # If slot is equal to head's slot, return the node's optimistic status.
//     # If slot is not after head's finalized slot, return false.
//     # Otherwise fetch the state's ancestor root and return its optimistic status.
func IsOptimistic(
	ctx context.Context,
	stateId []byte,
	optimisticModeFetcher blockchain.OptimisticModeFetcher,
	stateFetcher statefetcher.Fetcher,
	chainInfo blockchain.ChainInfoFetcher,
	database db.ReadOnlyDatabase,
) (bool, error) {
	stateIdString := strings.ToLower(string(stateId))
	switch stateIdString {
	case "head":
		return optimisticModeFetcher.IsOptimistic(ctx)
	case "genesis", "finalized":
		return false, nil
	case "justified":
		return isStateRootOptimistic(ctx, stateId, optimisticModeFetcher, stateFetcher, chainInfo, database)
	default:
		if len(stateId) == 32 {
			return isStateRootOptimistic(ctx, stateId, optimisticModeFetcher, stateFetcher, chainInfo, database)
		} else {
			slotNumber, parseErr := strconv.ParseUint(stateIdString, 10, 64)
			if parseErr != nil {
				// ID format does not match any valid options.
				e := statefetcher.NewStateIdParseError(parseErr)
				return true, &e
			}
			optimistic, err := optimisticModeFetcher.IsOptimistic(ctx)
			if err != nil {
				return true, errors.Wrap(err, "could not check optimistic status")
			}
			if !optimistic {
				return false, nil
			}
			finalizedSlot, err := slots.EpochStart(chainInfo.FinalizedCheckpt().Epoch)
			if err != nil {
				return true, errors.Wrap(err, "could not get head state's finalized slot")
			}
			if primitives.Slot(slotNumber) <= finalizedSlot {
				return false, nil
			}
			if primitives.Slot(slotNumber) == chainInfo.HeadSlot() {
				return optimisticModeFetcher.IsOptimistic(ctx)
			}
			headRoot, err := chainInfo.HeadRoot(ctx)
			if err != nil {
				return true, errors.Wrap(err, "could not get head root")
			}
			r, err := chainInfo.ForkChoicer().AncestorRoot(ctx, bytesutil.ToBytes32(headRoot), primitives.Slot(slotNumber))
			if err != nil {
				return true, errors.Wrap(err, "could not get ancestor root")
			}
			return optimisticModeFetcher.IsOptimisticForRoot(ctx, r)
		}
	}
}

func isStateRootOptimistic(
	ctx context.Context,
	stateId []byte,
	optimisticModeFetcher blockchain.OptimisticModeFetcher,
	stateFetcher statefetcher.Fetcher,
	chainInfo blockchain.ChainInfoFetcher,
	database db.ReadOnlyDatabase,
) (bool, error) {
	st, err := stateFetcher.State(ctx, stateId)
	if err != nil {
		return true, errors.Wrap(err, "could not fetch state")
	}
	if st.Slot() == chainInfo.HeadSlot() {
		return optimisticModeFetcher.IsOptimistic(ctx)
	}
	_, roots, err := database.HighestRootsBelowSlot(ctx, st.Slot()+1)
	if err != nil {
		return true, errors.Wrapf(err, "could not get block roots for slot %d", st.Slot())
	}
	if len(roots) == 0 {
		return false, errors.New("no blocks returned from the database")
	}
	for _, r := range roots {
		canonical, err := chainInfo.IsCanonical(ctx, r)
		if err != nil {
			return true, errors.Wrapf(err, "could not check canonical status")
		}
		if canonical {
			optimistic, err := optimisticModeFetcher.IsOptimistic(ctx)
			if err != nil {
				return true, errors.Wrap(err, "could not check optimistic status")
			}
			if !optimistic {
				return false, nil
			}
			finalizedSlot, err := slots.EpochStart(chainInfo.FinalizedCheckpt().Epoch)
			if err != nil {
				return true, errors.Wrap(err, "could not get head state's finalized slot")
			}
			if st.Slot() <= finalizedSlot {
				return false, nil
			}
			return true, nil
		}
	}
	// No block is canonical, return true
	return true, nil
}

// SyncDetailsJson contains information about node sync status.
type SyncDetailsJson struct {
	HeadSlot     string `json:"head_slot"`
	SyncDistance string `json:"sync_distance"`
	IsSyncing    bool   `json:"is_syncing"`
	IsOptimistic bool   `json:"is_optimistic"`
	ElOffline    bool   `json:"el_offline"`
}

// SyncDetailsContainer is a wrapper for SyncDetails.
type syncDetailsContainer struct {
	SyncDetails *SyncDetailsJson `json:"sync_details"`
}
