package db

import "github.com/prysmaticlabs/prysm/beacon-chain/db/iface"

// Database defines the necessary methods for Prysm's eth2 backend which may
// be implemented by any key-value or relational database in practice.
type Database = iface.Database
