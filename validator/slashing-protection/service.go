package slashingprotection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/validator/db"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "slashing-protection")

// Protector interface defines a struct which provides methods
// for validator slashing protection.
type Protector interface {
	IsSlashableAttestation(
		ctx context.Context,
		indexedAtt *ethpb.IndexedAttestation,
		pubKey [48]byte,
		domain *ethpb.DomainResponse,
	) (bool, error)
	IsSlashableBlock(
		ctx context.Context, block *ethpb.SignedBeaconBlock, pubKey [48]byte, domain *ethpb.DomainResponse,
	) (bool, error)
	shared.Service
}

type AttestingHistoryManager interface {
	SaveAttestingHistoryForPubKey(ctx context.Context, pubKey [48]byte) error
	LoadAttestingHistoryForPubKeys(ctx context.Context, attestingPubKeys [][48]byte) error
	AttestingHistoryForPubKey(ctx context.Context, pubKey [48]byte) (kv.EncHistoryData, error)
	ResetAttestingHistoryForEpoch(ctx context.Context)
}

// Service to manage validator slashing protection. Local slashing
// protection is mandatory at runtime but remote protection is optional.
type Service struct {
	ctx                          context.Context
	cancel                       context.CancelFunc
	remoteProtector              Protector
	validatorDB                  db.Database
	attestingHistoryByPubKeyLock sync.RWMutex
	attesterHistoryByPubKey      map[[48]byte]kv.EncHistoryData
}

// Config for the slashing protection service.
type Config struct {
	SlasherEndpoint            string
	CertFlag                   string
	GrpcMaxCallRecvMsgSizeFlag int
	GrpcRetriesFlag            uint
	GrpcRetryDelay             time.Duration
	GrpcHeadersFlag            string
	ValidatorDB                db.Database
}

// NewService creates a new validator service for the service registry.
func NewService(ctx context.Context, cfg *Config) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	srv := &Service{
		ctx:                     ctx,
		cancel:                  cancel,
		attesterHistoryByPubKey: make(map[[48]byte]kv.EncHistoryData),
		validatorDB:             cfg.ValidatorDB,
	}
	if cfg.SlasherEndpoint != "" {
		rp, err := NewRemoteProtector(ctx, cfg)
		if err != nil {
			return nil, err
		}
		srv.remoteProtector = rp
	}
	return srv, nil
}

// Start the slashing protection service.
func (s *Service) Start() {
	if s.remoteProtector != nil {
		s.remoteProtector.Start()
	}
}

// Stop the slashing protection service.
func (s *Service) Stop() error {
	s.cancel()
	log.Info("Stopping slashing protection service")
	if s.remoteProtector != nil {
		return s.remoteProtector.Stop()
	}
	return nil
}

// Status of the slashing protection service.
func (s *Service) Status() error {
	if s.remoteProtector != nil {
		return s.remoteProtector.Status()
	}
	return nil
}

func (s *Service) SaveAttestingHistoryForPubKey(ctx context.Context, pubKey [48]byte) error {
	s.attestingHistoryByPubKeyLock.RLock()
	defer s.attestingHistoryByPubKeyLock.RUnlock()
	history, ok := s.attesterHistoryByPubKey[pubKey]
	if !ok {
		return fmt.Errorf("no attesting history found for pubkey %#x", pubKey)
	}
	if err := s.validatorDB.SaveAttestationHistoryForPubKeyV2(ctx, pubKey, history); err != nil {
		return errors.Wrapf(err, "could not save attesting history to db for public key %#x", pubKey)
	}
	return nil
}

func (s *Service) LoadAttestingHistoryForPubKeys(ctx context.Context, attestingPubKeys [][48]byte) error {
	attHistoryByPubKey, err := s.validatorDB.AttestationHistoryForPubKeysV2(ctx, attestingPubKeys)
	if err != nil {
		return errors.Wrap(err, "could not get attester history")
	}
	s.attestingHistoryByPubKeyLock.Lock()
	s.attesterHistoryByPubKey = attHistoryByPubKey
	s.attestingHistoryByPubKeyLock.Unlock()
	return nil
}

func (s *Service) ResetAttestingHistoryForEpoch(ctx context.Context) {
	s.attestingHistoryByPubKeyLock.Lock()
	s.attesterHistoryByPubKey = make(map[[48]byte]kv.EncHistoryData)
	s.attestingHistoryByPubKeyLock.Unlock()
}

func (s *Service) AttestingHistoryForPubKey(ctx context.Context, pubKey [48]byte) (kv.EncHistoryData, error) {
	s.attestingHistoryByPubKeyLock.RLock()
	defer s.attestingHistoryByPubKeyLock.RUnlock()
	history, ok := s.attesterHistoryByPubKey[pubKey]
	if !ok {
		return kv.EncHistoryData{}, fmt.Errorf("no attesting history found for pubkey %#x", pubKey)
	}
	return history, nil
}
