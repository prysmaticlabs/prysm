// Package store implements the store object defined in the phase0 fork choice spec.
// It serves as a helpful middleware layer in between blockchain pkg and fork choice protoarray pkg.
// All the getters and setters are concurrent thread safe
package store
