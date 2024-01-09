package testing

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api/client/builder"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	v1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

// Config defines a config struct for dependencies into the service.
type Config struct {
	BeaconDB db.HeadAccessDatabase
}

// MockBuilderService to mock builder.
type MockBuilderService struct {
	HasConfigured         bool
	Cfg                   *Config
	Payload               *v1.ExecutionPayload
	PayloadCapella        *v1.ExecutionPayloadCapella
	PayloadDeneb          *v1.ExecutionPayloadDeneb
	BlobBundle            *v1.BlobsBundle
	Bid                   *util.FakeBid
	BidCapella            *util.FakeBidCapella
	BidDeneb              *util.FakeBidDeneb
	RegistrationCache     *cache.RegistrationCache
	ErrSubmitBlindedBlock error
	ErrGetHeader          error
	ErrRegisterValidator  error
}

func DefaultBuilderService(t testing.TB, ver int, useBuilder bool) *MockBuilderService {
	switch ver {
	case version.Bellatrix:
		bid, err := util.DefaultBid()
		require.NoError(t, err)
		return &MockBuilderService{
			HasConfigured: useBuilder,
			Payload:       util.DefaultPayload(),
			Bid:           bid,
		}
	case version.Capella:
		bid, err := util.DefaultBidCapella()
		require.NoError(t, err)
		return &MockBuilderService{
			HasConfigured:  useBuilder,
			PayloadCapella: util.DefaultPayloadCapella(),
			BidCapella:     bid,
		}
	case version.Deneb:
		bid, err := util.DefaultBidDeneb()
		require.NoError(t, err)
		return &MockBuilderService{
			HasConfigured: useBuilder,
			PayloadDeneb:  util.DefaultPayloadDeneb(),
			BidDeneb:      bid,
		}
	default:
		t.Fatal("Mock builder service does not support version " + version.String(ver))
		return nil
	}
}

// Configured for mocking.
func (s *MockBuilderService) Configured() bool {
	return s.HasConfigured
}

// SubmitBlindedBlock for mocking.
func (s *MockBuilderService) SubmitBlindedBlock(_ context.Context, b interfaces.ReadOnlySignedBeaconBlock) (interfaces.ExecutionData, *v1.BlobsBundle, error) {
	switch b.Version() {
	case version.Bellatrix:
		w, err := blocks.WrappedExecutionPayload(s.Payload)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not wrap payload")
		}
		return w, nil, s.ErrSubmitBlindedBlock
	case version.Capella:
		w, err := blocks.WrappedExecutionPayloadCapella(s.PayloadCapella, 0)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not wrap capella payload")
		}
		return w, nil, s.ErrSubmitBlindedBlock
	case version.Deneb:
		w, err := blocks.WrappedExecutionPayloadDeneb(s.PayloadDeneb, 0)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not wrap deneb payload")
		}
		return w, s.BlobBundle, s.ErrSubmitBlindedBlock
	default:
		return nil, nil, errors.New("unknown block version for mocking")
	}
}

// GetHeader for mocking.
func (s *MockBuilderService) GetHeader(_ context.Context, slot primitives.Slot, _ [32]byte, _ [48]byte) (builder.SignedBid, error) {
	if s.ErrGetHeader != nil {
		return nil, s.ErrGetHeader
	}

	if slots.ToEpoch(slot) >= params.BeaconConfig().DenebForkEpoch || s.BidDeneb != nil {
		sBid, err := s.BidDeneb.Sign()
		if err != nil {
			return nil, err
		}
		return builder.WrappedSignedBuilderBidDeneb(sBid)
	}
	if slots.ToEpoch(slot) >= params.BeaconConfig().CapellaForkEpoch || s.BidCapella != nil {
		sBid, err := s.BidCapella.Sign()
		if err != nil {
			return nil, err
		}
		return builder.WrappedSignedBuilderBidCapella(sBid)
	}
	sBid, err := s.Bid.Sign()
	if err != nil {
		return nil, err
	}
	w, err := builder.WrappedSignedBuilderBid(sBid)
	if err != nil {
		return nil, errors.Wrap(err, "could not wrap capella bid")
	}
	return w, nil
}

// RegistrationByValidatorID returns either the values from the cache or db.
func (s *MockBuilderService) RegistrationByValidatorID(ctx context.Context, id primitives.ValidatorIndex) (*ethpb.ValidatorRegistrationV1, error) {
	if s.RegistrationCache != nil {
		return s.RegistrationCache.RegistrationByIndex(id)
	}
	if s.Cfg.BeaconDB != nil {
		return s.Cfg.BeaconDB.RegistrationByValidatorID(ctx, id)
	}
	return nil, cache.ErrNotFoundRegistration
}

// RegisterValidator for mocking.
func (s *MockBuilderService) RegisterValidator(context.Context, []*ethpb.SignedValidatorRegistrationV1) error {
	return s.ErrRegisterValidator
}
