package db

import "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/kv"

// ErrNotFound can be used to determine if an error from a method in the database package
// represents a "not found" error. These often require different handling than a low-level
// i/o error. This variable copies the value in the kv package to the same scope as the Database interfaces,
// so that it is available to code paths that do not interact directly with the kv package.
var ErrNotFound = kv.ErrNotFound

// ErrNotFoundState wraps ErrNotFound for an error specific to a state not being found in the database.
var ErrNotFoundState = kv.ErrNotFoundState

// ErrNotFoundOriginBlockRoot wraps ErrNotFound for an error specific to the origin block root.
var ErrNotFoundOriginBlockRoot = kv.ErrNotFoundOriginBlockRoot

// ErrNotFoundBackfillBlockRoot wraps ErrNotFound for an error specific to the backfill block root.
var ErrNotFoundBackfillBlockRoot = kv.ErrNotFoundBackfillBlockRoot

// ErrNotFoundGenesisBlockRoot means no genesis block root was found, indicating the db was not initialized with genesis
var ErrNotFoundGenesisBlockRoot = kv.ErrNotFoundGenesisBlockRoot
