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
	ctx                          context.Context
	cancel                       context.CancelFunc
	validatorDB                  db.Database
	attestingHistoryByPubKeyLock sync.RWMutex
	attesterHistoryByPubKey      map[[48]byte]kv.EncHistoryData
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
		attesterHistoryByPubKey: make(map[[48]byte]kv.EncHistoryData),
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

// LoadAttestingHistoryForPubKeys retrieves histories from disk for the specified
// attesting public keys and loads them into an in-memory map.
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

// ResetAttestingHistoryForEpoch empties out the in-memory attesting histories.
func (s *Service) ResetAttestingHistoryForEpoch(ctx context.Context) {
	s.attestingHistoryByPubKeyLock.Lock()
	s.attesterHistoryByPubKey = make(map[[48]byte]kv.EncHistoryData)
	s.attestingHistoryByPubKeyLock.Unlock()
}

// AttestingHistoryForPubKey retrieves a history from the in-memory map of histories.
func (s *Service) AttestingHistoryForPubKey(ctx context.Context, pubKey [48]byte) (kv.EncHistoryData, error) {
	s.attestingHistoryByPubKeyLock.RLock()
	defer s.attestingHistoryByPubKeyLock.RUnlock()
	history, ok := s.attesterHistoryByPubKey[pubKey]
	if !ok {
		return kv.EncHistoryData{}, fmt.Errorf("no attesting history found for pubkey %#x", pubKey)
	}
	return history, nil
}
