package bytesutil

import (
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

// EpochToBytesLittleEndian conversion.
func EpochToBytesLittleEndian(i types.Epoch) []byte {
	return Uint64ToBytesLittleEndian(uint64(i))
}

// EpochToBytesBigEndian conversion.
func EpochToBytesBigEndian(i types.Epoch) []byte {
	return Uint64ToBytesBigEndian(uint64(i))
}

// BytesToEpochBigEndian conversion.
func BytesToEpochBigEndian(b []byte) types.Epoch {
	return types.Epoch(BytesToUint64BigEndian(b))
}

// SlotToBytesBigEndian conversion.
func SlotToBytesBigEndian(i types.Slot) []byte {
	return Uint64ToBytesBigEndian(uint64(i))
}

// BytesToSlotBigEndian conversion.
func BytesToSlotBigEndian(b []byte) types.Slot {
	return types.Slot(BytesToUint64BigEndian(b))
}

// ZeroRoot returns whether or not a root is of proper length and non-zero hash.
func ZeroRoot(root []byte) bool {
	return string(make([]byte, fieldparams.RootLength)) == string(root)
}

// IsRoot checks whether the byte array is a root.
func IsRoot(root []byte) bool {
	return len(root) == fieldparams.RootLength
}

// IsValidRoot checks whether the byte array is a valid root.
func IsValidRoot(root []byte) bool {
	return IsRoot(root) && !ZeroRoot(root)
}
