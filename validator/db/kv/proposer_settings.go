package kv

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	validatorServiceConfig "github.com/prysmaticlabs/prysm/v4/config/validator/service"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

var NoProposerSettingsFound = errors.New("no proposer settings found in bucket")

// UpdateProposerSettingsForPubkey updates the existing settings for an internal representation of the proposers settings file at a particular public key
func (s *Store) UpdateProposerSettingsForPubkey(ctx context.Context, pubkey [fieldparams.BLSPubkeyLength]byte, options *validatorServiceConfig.ProposerOption) error {
	_, span := trace.StartSpan(ctx, "validator.db.UpdateProposerSettingsDefault")
	defer span.End()
	err := s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(proposerSettingsBucket)
		b := bkt.Get(proposerSettingsKey)
		if len(b) != 0 {
			return fmt.Errorf("no proposer settings found in bucket")
		}
		to := &validatorServiceConfig.ProposerSettingsPayload{}
		if err := json.Unmarshal(b, to); err != nil {
			return errors.Wrap(err, "failed to unmarshal proposer settings")
		}
		settings, err := to.ToSettings()
		if err != nil {
			return errors.Wrap(err, "failed to convert payload to proposer settings")
		}
		if settings.ProposeConfig == nil {
			settings.ProposeConfig = make(map[[fieldparams.BLSPubkeyLength]byte]*validatorServiceConfig.ProposerOption)
		}
		settings.ProposeConfig[pubkey] = options
		m, err := json.Marshal(settings.ToPayload())
		if err != nil {
			return errors.Wrap(err, "failed to marshal proposer settings")
		}
		return bkt.Put(proposerSettingsKey, m)
	})
	return err
}

// UpdateProposerSettingsDefault updates the existing default settings for proposer settings
func (s *Store) UpdateProposerSettingsDefault(ctx context.Context, options *validatorServiceConfig.ProposerOption) error {
	_, span := trace.StartSpan(ctx, "validator.db.UpdateProposerSettingsDefault")
	defer span.End()
	if options == nil {
		return errors.New("proposer settings option was empty")
	}
	if options.FeeRecipientConfig == nil {
		return errors.New("fee recipient cannot be empty")
	}
	err := s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(proposerSettingsBucket)
		b := bkt.Get(proposerSettingsKey)
		if len(b) != 0 {
			return NoProposerSettingsFound
		}
		to := &validatorServiceConfig.ProposerSettingsPayload{}
		if err := json.Unmarshal(b, to); err != nil {
			return errors.Wrap(err, "failed to unmarshal proposer settings")
		}
		settings, err := to.ToSettings()
		if err != nil {
			return errors.Wrap(err, "failed to convert payload to proposer settings")
		}
		settings.DefaultConfig = options
		m, err := json.Marshal(settings.ToPayload())
		if err != nil {
			return errors.Wrap(err, "failed to marshal proposer settings")
		}
		return bkt.Put(proposerSettingsKey, m)
	})
	return err
}

// ProposerSettings gets the current proposer settings
func (s *Store) ProposerSettings(ctx context.Context) (*validatorServiceConfig.ProposerSettings, error) {
	_, span := trace.StartSpan(ctx, "validator.db.ProposerSettings")
	defer span.End()
	to := &validatorServiceConfig.ProposerSettingsPayload{}
	if err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(proposerSettingsBucket)
		b := bkt.Get(proposerSettingsKey)
		if len(b) != 0 {
			return NoProposerSettingsFound
		}
		if err := json.Unmarshal(b, to); err != nil {
			return errors.Wrap(err, "failed to unmarshal proposer settings")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return to.ToSettings()
}

// ProposerSettingsExists returns true or false if the settings exist or not
func (s *Store) ProposerSettingsExists(ctx context.Context) (bool, error) {
	ps, err := s.ProposerSettings(ctx)
	if err != nil {
		if errors.Is(err, NoProposerSettingsFound) {
			return false, nil
		}
		return false, err
	}
	if ps == nil {
		return false, nil
	}
	return true, nil
}

// SaveProposerSettings saves the entire proposer setting overriding the existing settings
func (s *Store) SaveProposerSettings(ctx context.Context, settings *validatorServiceConfig.ProposerSettings) error {
	_, span := trace.StartSpan(ctx, "validator.db.SaveProposerSettings")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(proposerSettingsBucket)
		m, err := json.Marshal(settings.ToPayload())
		if err != nil {
			return errors.Wrap(err, "failed to marshal proposer settings")
		}
		return bkt.Put(proposerSettingsKey, m)
	})
}
