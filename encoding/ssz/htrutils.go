package ssz

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// Uint64Root computes the HashTreeRoot Merkleization of
// a simple uint64 value according to the Ethereum
// Simple Serialize specification.
func Uint64Root(val uint64) [32]byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, val)
	root := bytesutil.ToBytes32(buf)
	return root
}

// ForkRoot computes the HashTreeRoot Merkleization of
// a Fork struct value according to the Ethereum
// Simple Serialize specification.
func ForkRoot(fork *ethpb.Fork) ([32]byte, error) {
	fieldRoots := make([][]byte, 3)
	if fork != nil {
		prevRoot := bytesutil.ToBytes32(fork.PreviousVersion)
		fieldRoots[0] = prevRoot[:]
		currRoot := bytesutil.ToBytes32(fork.CurrentVersion)
		fieldRoots[1] = currRoot[:]
		forkEpochBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(forkEpochBuf, uint64(fork.Epoch))
		epochRoot := bytesutil.ToBytes32(forkEpochBuf)
		fieldRoots[2] = epochRoot[:]
	}
	return BitwiseMerkleize(hash.CustomSHA256Hasher(), fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

// CheckpointRoot computes the HashTreeRoot Merkleization of
// a InitWithReset struct value according to the Ethereum
// Simple Serialize specification.
func CheckpointRoot(hasher HashFn, checkpoint *ethpb.Checkpoint) ([32]byte, error) {
	fieldRoots := make([][]byte, 2)
	if checkpoint != nil {
		epochBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(epochBuf, uint64(checkpoint.Epoch))
		epochRoot := bytesutil.ToBytes32(epochBuf)
		fieldRoots[0] = epochRoot[:]
		ckpRoot := bytesutil.ToBytes32(checkpoint.Root)
		fieldRoots[1] = ckpRoot[:]
	}
	return BitwiseMerkleize(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

// ByteArrayRootWithLimit computes the HashTreeRoot Merkleization of
// a list of [32]byte roots according to the Ethereum Simple Serialize
// specification.
func ByteArrayRootWithLimit(roots [][]byte, limit uint64) ([32]byte, error) {
	result, err := BitwiseMerkleize(hash.CustomSHA256Hasher(), roots, uint64(len(roots)), limit)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute byte array merkleization")
	}
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(roots))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal byte array length")
	}
	// We need to mix in the length of the slice.
	output := make([]byte, 32)
	copy(output, buf.Bytes())
	mixedLen := MixInLength(result, output)
	return mixedLen, nil
}

// SlashingsRoot computes the HashTreeRoot Merkleization of
// a list of uint64 slashing values according to the Ethereum
// Simple Serialize specification.
func SlashingsRoot(slashings []uint64) ([32]byte, error) {
	slashingMarshaling := make([][]byte, fieldparams.SlashingsLength)
	for i := 0; i < len(slashings) && i < len(slashingMarshaling); i++ {
		slashBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(slashBuf, slashings[i])
		slashingMarshaling[i] = slashBuf
	}
	slashingChunks, err := Pack(slashingMarshaling)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not pack slashings into chunks")
	}
	return BitwiseMerkleize(hash.CustomSHA256Hasher(), slashingChunks, uint64(len(slashingChunks)), uint64(len(slashingChunks)))
}
