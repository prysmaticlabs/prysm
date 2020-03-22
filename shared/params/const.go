package params

type KeyBytes = [48]byte // type for pub/ priv key bytes
type ValidatorRoleBytes = [48]byte  // validator epch role
type RootBytes = [32]byte // hash tree root for data structs for the validator to sign
type AttestationHashBytes = [32]byte // attestation hash bytes
type BalanceBytes = [48]byte // account balance in bytes
type DomainBytes = []byte // https://github.com/ethereum/eth2.0-specs/blob/dev/specs/phase0/beacon-chain.md#domain-types
type GraffitiBytes = []byte // Arbitrary data in block struct
type SlotNumber = uint64  // slot number
type EpochNumber = uint64 // epoch number