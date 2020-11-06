package db

import "github.com/prysmaticlabs/prysm/beacon-chain/db/kv"

var _ Database = (*kv.Store)(nil)
