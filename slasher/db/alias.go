package db

import "github.com/prysmaticlabs/prysm/slasher/db/iface"

// ReadOnlyDatabase exposes the Slasher's DB read only functions for all slasher related buckets.
type ReadOnlyDatabase = iface.ReadOnlyDatabase

// WriteAccessDatabase exposes the Slasher's DB writing functions for all slasher related buckets.
type WriteAccessDatabase = iface.WriteAccessDatabase

// FullAccessDatabase exposes Slasher's DB write and read functions for all slasher related buckets.
type FullAccessDatabase = iface.FullAccessDatabase

// Database defines the necessary methods for the Slasher's DB which may be implemented by any
// key-value or relational database in practice. This is the full database interface which should
// not be used often. Prefer a more restrictive interface in this package.
type Database = iface.Database
