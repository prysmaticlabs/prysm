package main

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"sync"
	"time"

	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"

	"github.com/pkg/errors"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// Required fields
	datadir = flag.String("datadir", "/Users/shayzluf/Library/Eth2/beaconchaindata/", "Path to data directory.")
)

// sortableAttestations implements the Sort interface to sort attestations
// by slot as the canonical sorting attribute.
type sortableAttestations []*ethpb.Attestation

func (s sortableAttestations) Len() int      { return len(s) }
func (s sortableAttestations) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortableAttestations) Less(i, j int) bool {
	return s[i].Data.Slot < s[j].Data.Slot
}

func init() {
	fc := featureconfig.Get()
	fc.EnableSnappyDBCompression = true
	params.UseMinimalConfig()
	featureconfig.Init(fc)
}

func main() {
	flag.Parse()
	fmt.Println("Starting process...")
	d, err := db.NewDB(*datadir)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	cp, _ := d.FinalizedCheckpoint(ctx)
	_, slasherClient, err := startSlasherClient(ctx, "localhost:5000")
	if err != nil {
		panic(err)
	}
	for e := uint64(0); e < cp.Epoch; e++ {
		atts, err := d.Attestations(ctx, filters.NewFilter().SetTargetEpoch(e))
		bcs, err := ListBeaconCommittees(ctx, d, &ethpb.ListCommitteesRequest{
			QueryFilter: &ethpb.ListCommitteesRequest_Epoch{
				Epoch: e,
			},
		})
		if err != nil {
			panic(err)
		}
		start := time.Now()

		for _, attestation := range atts {
			scs, ok := bcs.Committees[attestation.Data.Slot]
			if !ok {
				var keys []uint64
				for k := range bcs.Committees {
					keys = append(keys, k)
				}
				log.Errorf("committees doesnt contain the attestation slot: %d, actual first slot: %v", attestation.Data.Slot, keys)
				continue
			}
			if attestation.Data.CommitteeIndex >= uint64(len(scs.Committees)) {
				log.Errorf("committee index is out of range in committee index wanted: %v, actual: %v", attestation.Data.CommitteeIndex, len(scs.Committees))
				continue
			}
			sc := scs.Committees[attestation.Data.CommitteeIndex]
			c := sc.ValidatorIndices
			ia, err := ConvertToIndexed(ctx, attestation, c)
			if err != nil {
				log.Error(err)
				continue
			}
			sar, err := slasherClient.IsSlashableAttestation(ctx, ia)
			if err != nil {
				log.Error(err)
				continue
			}
			if len(sar.AttesterSlashing) > 0 {
				log.Infof("slashing response: %v", sar.AttesterSlashing)
			}

		}
		elapsed := time.Since(start)
		log.Infof("detecting slashable events on: %d attestations from epoch: %d took: %d on average: %d per attestation", len(atts), e, elapsed.Milliseconds(), elapsed.Milliseconds()/int64(len(atts)))

	}
	//errorWg.Wait()
	//close(errOut)
	//for err := range errOut {
	//	log.Error(errors.Wrap(err, "error while writing to db in background"))
	//}
	fmt.Println("done")
}

func startSlasherClient(ctx context.Context, slasherProvider string) (*grpc.ClientConn, slashpb.SlasherClient, error) {
	var dialOpt grpc.DialOption

	dialOpt = grpc.WithInsecure()
	log.Warn("You are using an insecure gRPC connection! Please provide a certificate and key to use a secure connection.")

	slasherOpts := []grpc.DialOption{
		dialOpt,
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		grpc.WithStreamInterceptor(middleware.ChainStreamClient(
			grpc_opentracing.StreamClientInterceptor(),
			grpc_prometheus.StreamClientInterceptor,
		)),
		grpc.WithUnaryInterceptor(middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(),
			grpc_prometheus.UnaryClientInterceptor,
		)),
	}
	conn, err := grpc.DialContext(ctx, slasherProvider, slasherOpts...)
	if err != nil {
		log.Errorf("Could not dial endpoint: %s, %v", slasherProvider, err)
		return nil, nil, err
	}
	log.Info("Successfully started gRPC connection")
	slasherClient := slashpb.NewSlasherClient(conn)
	return conn, slasherClient, nil

}

// ListBeaconCommittees for a given epoch.
//
// If no filter criteria is specified, the response returns
// all beacon committees for the current epoch. The results are paginated by default.
func ListBeaconCommittees(
	ctx context.Context,
	db db.Database,
	req *ethpb.ListCommitteesRequest,
) (*ethpb.BeaconCommittees, error) {
	headState, err := db.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}

	var requestingGenesis bool
	var startSlot uint64
	switch q := req.QueryFilter.(type) {
	case *ethpb.ListCommitteesRequest_Epoch:
		startSlot = helpers.StartSlot(q.Epoch)
	case *ethpb.ListCommitteesRequest_Genesis:
		requestingGenesis = q.Genesis
	default:
		startSlot = headState.Slot
	}

	var attesterSeed [32]byte
	var activeIndices []uint64
	// This is the archival condition, if the requested epoch is < current epoch or if we are
	// requesting data from the genesis epoch.
	if requestingGenesis || helpers.SlotToEpoch(startSlot) < helpers.SlotToEpoch(headState.Slot) {
		activeIndices, err = helpers.ActiveValidatorIndices(headState, helpers.SlotToEpoch(startSlot))
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"Could not retrieve active indices for epoch %d: %v",
				helpers.SlotToEpoch(startSlot),
				err,
			)
		}
		archivedCommitteeInfo, err := db.ArchivedCommitteeInfo(ctx, helpers.SlotToEpoch(startSlot))
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"Could not request archival data for epoch %d: %v",
				helpers.SlotToEpoch(startSlot),
				err,
			)
		}
		if archivedCommitteeInfo == nil {
			return nil, status.Errorf(
				codes.NotFound,
				"Could not retrieve data for epoch %d, perhaps --archive in the running beacon node is disabled",
				helpers.SlotToEpoch(startSlot),
			)
		}
		attesterSeed = bytesutil.ToBytes32(archivedCommitteeInfo.AttesterSeed)
	} else if !requestingGenesis && helpers.SlotToEpoch(startSlot) == helpers.SlotToEpoch(headState.Slot) {
		// Otherwise, we use data from the current epoch.
		currentEpoch := helpers.SlotToEpoch(headState.Slot)
		activeIndices, err = helpers.ActiveValidatorIndices(headState, currentEpoch)
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"Could not retrieve active indices for current epoch %d: %v",
				currentEpoch,
				err,
			)
		}
		attesterSeed, err = helpers.Seed(headState, currentEpoch, params.BeaconConfig().DomainBeaconAttester)
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"Could not retrieve attester seed for current epoch %d: %v",
				currentEpoch,
				err,
			)
		}
	} else {
		// Otherwise, we are requesting data from the future and we return an error.
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
			helpers.CurrentEpoch(headState),
			helpers.SlotToEpoch(startSlot),
		)
	}

	committeesList := make(map[uint64]*ethpb.BeaconCommittees_CommitteesList)
	for slot := startSlot; slot < startSlot+params.BeaconConfig().SlotsPerEpoch; slot++ {
		var countAtSlot = uint64(len(activeIndices)) / params.BeaconConfig().SlotsPerEpoch / params.BeaconConfig().TargetCommitteeSize
		if countAtSlot > params.BeaconConfig().MaxCommitteesPerSlot {
			countAtSlot = params.BeaconConfig().MaxCommitteesPerSlot
		}
		if countAtSlot == 0 {
			countAtSlot = 1
		}
		committeeItems := make([]*ethpb.BeaconCommittees_CommitteeItem, countAtSlot)
		for i := uint64(0); i < countAtSlot; i++ {
			epochOffset := i + (slot%params.BeaconConfig().SlotsPerEpoch)*countAtSlot
			totalCount := countAtSlot * params.BeaconConfig().SlotsPerEpoch
			committee, err := helpers.ComputeCommittee(activeIndices, attesterSeed, epochOffset, totalCount)
			if err != nil {
				return nil, status.Errorf(
					codes.Internal,
					"Could not compute committee for slot %d: %v",
					slot,
					err,
				)
			}
			committeeItems[i] = &ethpb.BeaconCommittees_CommitteeItem{
				ValidatorIndices: committee,
			}
		}
		committeesList[slot] = &ethpb.BeaconCommittees_CommitteesList{
			Committees: committeeItems,
		}
	}

	return &ethpb.BeaconCommittees{
		Epoch:                helpers.SlotToEpoch(startSlot),
		Committees:           committeesList,
		ActiveValidatorCount: uint64(len(activeIndices)),
	}, nil
}

// ConvertToIndexed converts attestation to (almost) indexed-verifiable form.
//
// Note about spec pseudocode definition. The state was used by get_attesting_indices to determine
// the attestation committee. Now that we provide this as an argument, we no longer need to provide
// a state.
//
// Spec pseudocode definition:
//   def get_indexed_attestation(state: BeaconState, attestation: Attestation) -> IndexedAttestation:
//    """
//    Return the indexed attestation corresponding to ``attestation``.
//    """
//    attesting_indices = get_attesting_indices(state, attestation.data, attestation.aggregation_bits)
//    custody_bit_1_indices = get_attesting_indices(state, attestation.data, attestation.custody_bits)
//    assert custody_bit_1_indices.issubset(attesting_indices)
//    custody_bit_0_indices = attesting_indices.difference(custody_bit_1_indices)
//
//    return IndexedAttestation(
//        custody_bit_0_indices=sorted(custody_bit_0_indices),
//        custody_bit_1_indices=sorted(custody_bit_1_indices),
//        data=attestation.data,
//        signature=attestation.signature,
//    )
func ConvertToIndexed(ctx context.Context, attestation *ethpb.Attestation, committee []uint64) (*ethpb.IndexedAttestation, error) {
	attIndices, err := helpers.AttestingIndices(attestation.AggregationBits, committee)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attesting indices")
	}

	cb1i, err := helpers.AttestingIndices(attestation.CustodyBits, committee)
	if err != nil {
		return nil, err
	}
	if !sliceutil.SubsetUint64(cb1i, attIndices) {
		return nil, fmt.Errorf("%v is not a subset of %v", cb1i, attIndices)
	}
	cb1Map := make(map[uint64]bool)
	for _, idx := range cb1i {
		cb1Map[idx] = true
	}
	cb0i := []uint64{}
	for _, idx := range attIndices {
		if !cb1Map[idx] {
			cb0i = append(cb0i, idx)
		}
	}
	sort.Slice(cb0i, func(i, j int) bool {
		return cb0i[i] < cb0i[j]
	})

	sort.Slice(cb1i, func(i, j int) bool {
		return cb1i[i] < cb1i[j]
	})
	inAtt := &ethpb.IndexedAttestation{
		Data:                attestation.Data,
		Signature:           attestation.Signature,
		CustodyBit_0Indices: cb0i,
		CustodyBit_1Indices: cb1i,
	}
	return inAtt, nil
}

func mergeChannels(cs []chan error, out chan error, wg *sync.WaitGroup) {
	wg.Add(len(cs))
	for _, c := range cs {
		go func(c <-chan error) {
			for v := range c {
				out <- v
			}
			wg.Done()
		}(c)
	}

	return
}
