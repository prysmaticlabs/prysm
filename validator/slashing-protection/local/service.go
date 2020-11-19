package local

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/db"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "local-slashing-protection")

// Service to manage validator slashing protection. Local slashing
// protection is mandatory at runtime.
type Service struct {
	ctx                     context.Context
	cancel                  context.CancelFunc
	validatorDB             db.Database
	attesterHistoryByPubKey *sync.Map
}

// Config for the slashing protection service.
type Config struct {
	ValidatorDB db.Database
}

// NewService creates a new validator service for the service registry.
func NewService(ctx context.Context, cfg *Config) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	srv := &Service{
		ctx:                     ctx,
		cancel:                  cancel,
		attesterHistoryByPubKey: &sync.Map{},
		validatorDB:             cfg.ValidatorDB,
	}
	return srv, nil
}

// Start the slashing protection service.
func (s *Service) Start() {
}

// Stop the slashing protection service.
func (s *Service) Stop() error {
	s.cancel()
	return nil
}

// Status of the slashing protection service.
func (s *Service) Status() error {
	return nil
}

// SaveAttestingHistoryForPubKey persists current, in-memory attesting history for
// a public key to the database.
func (s *Service) SaveAttestingHistoryForPubKey(ctx context.Context, pubKey [48]byte) error {
	val, ok := s.attesterHistoryByPubKey.Load(pubKey)
	if !ok {
		return fmt.Errorf("no attesting history found for pubkey %#x", pubKey)
	}
	history, ok := val.(kv.EncHistoryData)
	if !ok {
		return fmt.Errorf("value in map for %#x is not attesting history data", pubKey)
	}
	if err := s.validatorDB.SaveAttestationHistoryForPubKeyV2(ctx, pubKey, history); err != nil {
		return errors.Wrapf(err, "could not save attesting history to db for public key %#x", pubKey)
	}
	return nil
}

// LoadAttestingHistoryForPubKeys retrieves histories from disk for the specified
// attesting public keys and loads them into an in-memory map.
func (s *Service) LoadAttestingHistoryForPubKeys(ctx context.Context, attestingPubKeys [][48]byte) error {
	attHistoryByPubKey, err := s.validatorDB.AttestationHistoryForPubKeysV2(ctx, attestingPubKeys)
	if err != nil {
		return errors.Wrap(err, "could not get attester history")
	}
	s.attesterHistoryByPubKey = &sync.Map{}
	for pubKey, history := range attHistoryByPubKey {
		s.attesterHistoryByPubKey.Store(pubKey, history)
	}
	return nil
}

// ResetAttestingHistoryForEpoch empties out the in-memory attesting histories.
func (s *Service) ResetAttestingHistoryForEpoch(ctx context.Context) {
	s.attesterHistoryByPubKey = &sync.Map{}
}

// AttestingHistoryForPubKey retrieves a history from the in-memory map of histories.
func (s *Service) AttestingHistoryForPubKey(ctx context.Context, pubKey [48]byte) (kv.EncHistoryData, error) {
	val, ok := s.attesterHistoryByPubKey.Load(pubKey)
	if !ok {
		return nil, fmt.Errorf("no attesting history found for pubkey %#x", pubKey)
	}
	history, ok := val.(kv.EncHistoryData)
	if !ok {
		return nil, fmt.Errorf("value in map for %#x is not attesting history data", pubKey)
	}
	return history, nil
}
