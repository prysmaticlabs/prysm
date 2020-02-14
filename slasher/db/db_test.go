package db

import "github.com/prysmaticlabs/prysm/slasher/db/kv"

var _ = Database(&kv.Store{})
