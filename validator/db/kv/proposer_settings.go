package kv

// UpdateProposerSettings
func (s *Store) UpdateProposerSettings(_ context.Context, genValRoot []byte) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(genesisInfoBucket)
		enc := bkt.Get(genesisValidatorsRootKey)
		if len(enc) != 0 {
			return fmt.Errorf("cannot overwite existing genesis validators root: %#x", enc)
		}
		return bkt.Put(genesisValidatorsRootKey, genValRoot)
	})
	return err
}

//

// ProposerSettings
func (s *Store) ProposerSettings(_ context.Context, genValRoot []byte) error {

}
