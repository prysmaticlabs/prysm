package db

import "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/kv"

var _ Database = (*kv.Store)(nil)
