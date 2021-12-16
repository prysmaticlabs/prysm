package db

import "github.com/prysmaticlabs/prysm/beacon-chain/db/kv"

// ErrNotFound can be used to determine if an error from a method in the database package
// represents a "not found" error. These often require different handling than a low-level
// i/o error. This variable copies the value in the kv package to the same scope as the Database interfaces,
// so that it is available to code paths that do not interact directly with the kv package.
var ErrNotFound = kv.ErrNotFound
