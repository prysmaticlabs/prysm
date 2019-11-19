package validator

import (
	"context"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

var log logrus.FieldLogger

func init() {
	log = logrus.WithField("prefix", "rpc/validator")
}

// Server defines a server implementation of the gRPC Validator service,
// providing RPC endpoints for obtaining validator assignments per epoch, the slots
// and committees in which particular validators need to perform their responsibilities,
// and more.
type Server struct {
	Ctx                context.Context
	BeaconDB           db.Database
	HeadFetcher        blockchain.HeadFetcher
	ForkFetcher        blockchain.ForkFetcher
	CanonicalStateChan chan *pbp2p.BeaconState
	BlockFetcher       powchain.POWBlockFetcher
	DepositFetcher     depositcache.DepositFetcher
	ChainStartFetcher  powchain.ChainStartFetcher
	Eth1InfoFetcher    powchain.ChainInfoFetcher
	SyncChecker        sync.Checker
	StateFeedListener  blockchain.ChainFeeds
	ChainStartChan     chan time.Time
}

// WaitForActivation checks if a validator public key exists in the active validator registry of the current
// beacon state, if not, then it creates a stream which listens for canonical states which contain
// the validator with the public key as an active validator record.
func (vs *Server) WaitForActivation(req *pb.ValidatorActivationRequest, stream pb.ValidatorService_WaitForActivationServer) error {
	activeValidatorExists, validatorStatuses, err := vs.multipleValidatorStatus(stream.Context(), req.PublicKeys)
	if err != nil {
		return status.Errorf(codes.Internal, "Could not fetch validator status: %v", err)
	}
	res := &pb.ValidatorActivationResponse{
		Statuses: validatorStatuses,
	}
	if activeValidatorExists {
		return stream.Send(res)
	}
	if err := stream.Send(res); err != nil {
		return status.Errorf(codes.Internal, "Could not send response over stream: %v", err)
	}

	for {
		select {
		case <-time.After(6 * time.Second):
			activeValidatorExists, validatorStatuses, err := vs.multipleValidatorStatus(stream.Context(), req.PublicKeys)
			if err != nil {
				return status.Errorf(codes.Internal, "Could not fetch validator status: %v", err)
			}
			res := &pb.ValidatorActivationResponse{
				Statuses: validatorStatuses,
			}
			if activeValidatorExists {
				return stream.Send(res)
			}
			if err := stream.Send(res); err != nil {
				return status.Errorf(codes.Internal, "Could not send response over stream: %v", err)
			}
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Stream context canceled")
		case <-vs.Ctx.Done():
			return status.Error(codes.Canceled, "RPC context canceled")
		}
	}
}

// ValidatorIndex is called by a validator to get its index location in the beacon state.
func (vs *Server) ValidatorIndex(ctx context.Context, req *pb.ValidatorIndexRequest) (*pb.ValidatorIndexResponse, error) {
	index, ok, err := vs.BeaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(req.PublicKey))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not fetch validator index: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.Internal, "Could not find validator index for public key %#x not found", req.PublicKey)
	}

	return &pb.ValidatorIndexResponse{Index: index}, nil
}

// ValidatorPerformance reports the validator's latest balance along with other important metrics on
// rewards and penalties throughout its lifecycle in the beacon chain.
func (vs *Server) ValidatorPerformance(
	ctx context.Context, req *pb.ValidatorPerformanceRequest,
) (*pb.ValidatorPerformanceResponse, error) {
	var err error
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}

	// Advance state with empty transitions up to the requested epoch start slot.
	if req.Slot > headState.Slot {
		headState, err = state.ProcessSlots(ctx, headState, req.Slot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not process slots up to %d: %v", req.Slot, err)
		}
	}

	balances := make([]uint64, len(req.PublicKeys))
	missingValidators := make([][]byte, 0)
	for i, key := range req.PublicKeys {
		index, ok, err := vs.BeaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(key))
		if err != nil || !ok {
			missingValidators = append(missingValidators, key)
			balances[i] = 0
			continue
		}
		balances[i] = headState.Balances[index]
	}

	activeCount, err := helpers.ActiveValidatorCount(headState, helpers.SlotToEpoch(req.Slot))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve active validator count: %v", err)
	}

	totalActiveBalance, err := helpers.TotalActiveBalance(headState)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve total active balance: %v", err)
	}

	avgBalance := float32(totalActiveBalance / activeCount)
	return &pb.ValidatorPerformanceResponse{
		Balances:                      balances,
		AverageActiveValidatorBalance: avgBalance,
		MissingValidators:             missingValidators,
		TotalValidators:               uint64(len(headState.Validators)),
		TotalActiveValidators:         uint64(activeCount),
	}, nil
}

// ExitedValidators queries validator statuses for a give list of validators
// and returns a filtered list of validator keys that are exited.
func (vs *Server) ExitedValidators(
	ctx context.Context,
	req *pb.ExitedValidatorsRequest) (*pb.ExitedValidatorsResponse, error) {

	_, statuses, err := vs.multipleValidatorStatus(ctx, req.PublicKeys)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve validator statuses: %v", err)
	}

	exitedKeys := make([][]byte, 0)
	for _, st := range statuses {
		s := st.Status.Status
		if s == pb.ValidatorStatus_EXITED ||
			s == pb.ValidatorStatus_EXITED_SLASHED ||
			s == pb.ValidatorStatus_INITIATED_EXIT {
			exitedKeys = append(exitedKeys, st.PublicKey)
		}
	}

	resp := &pb.ExitedValidatorsResponse{
		PublicKeys: exitedKeys,
	}

	return resp, nil
}

// DomainData fetches the current domain version information from the beacon state.
func (vs *Server) DomainData(ctx context.Context, request *pb.DomainRequest) (*pb.DomainResponse, error) {
	fork := vs.ForkFetcher.CurrentFork()
	dv := helpers.Domain(fork, request.Epoch, request.Domain)
	return &pb.DomainResponse{
		SignatureDomain: dv,
	}, nil
}

// CanonicalHead of the current beacon chain. This method is requested on-demand
// by a validator when it is their time to propose or attest.
func (vs *Server) CanonicalHead(ctx context.Context, req *ptypes.Empty) (*ethpb.BeaconBlock, error) {
	return vs.HeadFetcher.HeadBlock(), nil
}

// WaitForChainStart queries the logs of the Deposit Contract in order to verify the beacon chain
// has started its runtime and validators begin their responsibilities. If it has not, it then
// subscribes to an event stream triggered by the powchain service whenever the ChainStart log does
// occur in the Deposit Contract on ETH 1.0.
func (vs *Server) WaitForChainStart(req *ptypes.Empty, stream pb.ValidatorService_WaitForChainStartServer) error {
	head, err := vs.BeaconDB.HeadState(context.Background())
	if err != nil {
		return status.Errorf(codes.Internal, "Could not retrieve head state: %v", err)
	}
	if head != nil {
		res := &pb.ChainStartResponse{
			Started:     true,
			GenesisTime: head.GenesisTime,
		}
		return stream.Send(res)
	}

	sub := vs.StateFeedListener.StateInitializedFeed().Subscribe(vs.ChainStartChan)
	defer sub.Unsubscribe()
	for {
		select {
		case chainStartTime := <-vs.ChainStartChan:
			log.Info("Sending genesis time notification to connected validator clients")
			res := &pb.ChainStartResponse{
				Started:     true,
				GenesisTime: uint64(chainStartTime.Unix()),
			}
			return stream.Send(res)
		case <-sub.Err():
			return status.Error(codes.Aborted, "Subscriber closed, exiting goroutine")
		case <-vs.Ctx.Done():
			return status.Error(codes.Canceled, "Context canceled")
		}
	}
}
