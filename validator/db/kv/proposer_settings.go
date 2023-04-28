package kv

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	validator_service_config "github.com/prysmaticlabs/prysm/v4/config/validator/service"
	bolt "go.etcd.io/bbolt"
)

// UpdateProposerSettings
func (s *Store) UpdateProposerSettings(_ context.Context, pubkey string, options *validator_service_config.ProposerOptionPayload) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(proposerSettingsBucket)
		if bkt == nil {
			if _, err := tx.CreateBucketIfNotExists(proposerSettingsBucket); err != nil {
				return err
			}
		}
		b := bkt.Get(proposerSettingsKey)
		if len(b) != 0 {
			return fmt.Errorf("no proposer settings found in bucket")
		}
		to := &validator_service_config.ProposerSettingsPayload{}
		if err := json.Unmarshal(b, to); err != nil {
			return errors.Wrap(err, "failed to unmarshal proposer settings")
		}
		if to.ProposerConfig == nil {
			to.ProposerConfig = make(map[string]*validator_service_config.ProposerOptionPayload)
		}
		to.ProposerConfig[pubkey] = options
		m, err := json.Marshal(to)
		if err != nil {
			return errors.Wrap(err, "failed to marshal proposer settings")
		}
		return bkt.Put(proposerSettingsKey, m)
	})
	return err
}

//

// ProposerSettings
func (s *Store) ProposerSettings(ctx context.Context) (*validator_service_config.ProposerSettingsPayload, error) {
	return nil, nil
}

// SaveProposerSettings
func (s *Store) SaveProposerSettings(ctx context.Context, settings *validator_service_config.ProposerSettingsPayload) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(genesisInfoBucket)
		enc := bkt.Get(genesisValidatorsRootKey)
		if len(enc) == 0 {
			return nil
		}
		return nil
	})
}
