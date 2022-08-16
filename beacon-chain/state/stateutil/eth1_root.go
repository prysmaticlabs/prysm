package stateutil

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// Eth1DataRootWithHasher returns the hash tree root of input `eth1Data`.
func Eth1DataRootWithHasher(hasher ssz.HashFn, eth1Data *ethpb.Eth1Data) ([32]byte, error) {
	if eth1Data == nil {
		return [32]byte{}, errors.New("nil eth1 data")
	}

	fieldRoots := make([][32]byte, 3)
	for i := 0; i < len(fieldRoots); i++ {
		fieldRoots[i] = [32]byte{}
	}

	if len(eth1Data.DepositRoot) > 0 {
		fieldRoots[0] = bytesutil.ToBytes32(eth1Data.DepositRoot)
	}

	eth1DataCountBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DataCountBuf, eth1Data.DepositCount)
	fieldRoots[1] = bytesutil.ToBytes32(eth1DataCountBuf)
	if len(eth1Data.BlockHash) > 0 {
		fieldRoots[2] = bytesutil.ToBytes32(eth1Data.BlockHash)
	}
	root, err := ssz.BitwiseMerkleize(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, err
	}
	return root, nil
}

// Eth1DatasRoot returns the hash tree root of input `eth1Datas`.
func Eth1DatasRoot(eth1Datas []*ethpb.Eth1Data) ([32]byte, error) {
	hasher := hash.CustomSHA256Hasher()
	eth1VotesRoots := make([][32]byte, 0, len(eth1Datas))
	for i := 0; i < len(eth1Datas); i++ {
		eth1, err := Eth1DataRootWithHasher(hasher, eth1Datas[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute eth1data merkleization")
		}
		eth1VotesRoots = append(eth1VotesRoots, eth1)
	}

	eth1VotesRootsRoot, err := ssz.BitwiseMerkleize(
		hasher,
		eth1VotesRoots,
		uint64(len(eth1VotesRoots)),
		fieldparams.Eth1DataVotesLength,
	)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	eth1VotesRootBuf := new(bytes.Buffer)
	if err := binary.Write(eth1VotesRootBuf, binary.LittleEndian, uint64(len(eth1Datas))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal eth1data votes length")
	}
	// We need to mix in the length of the slice.
	eth1VotesRootBufRoot := make([]byte, 32)
	copy(eth1VotesRootBufRoot, eth1VotesRootBuf.Bytes())
	root := ssz.MixInLength(eth1VotesRootsRoot, eth1VotesRootBufRoot)

	return root, nil
}
