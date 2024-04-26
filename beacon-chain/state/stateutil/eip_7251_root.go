package stateutil

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

func PendingBalanceDepositsRoot(slice []*ethpb.PendingBalanceDeposit) ([32]byte, error) {
	max := uint64(fieldparams.PendingBalanceDepositsLimit)
	if uint64(len(slice)) > max {
		return [32]byte{}, fmt.Errorf("pending balance deposits exceeds max length %d", max)
	}

	roots := make([][32]byte, len(slice))
	for i := 0; i < len(slice); i++ {
		r, err := slice[i].HashTreeRoot()
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not merkleize pending balance deposits")
		}
		roots[i] = r
	}

	sliceRoot, err := ssz.BitwiseMerkleize(roots, uint64(len(roots)), fieldparams.PendingBalanceDepositsLimit)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute pending balance deposits merkleization")
	}
	sliceLenBuf := new(bytes.Buffer)
	if err := binary.Write(sliceLenBuf, binary.LittleEndian, uint64(len(slice))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal pending balance deposits length")
	}
	// We need to mix in the length of the slice.
	sliceLenRoot := make([]byte, 32)
	copy(sliceLenRoot, sliceLenBuf.Bytes())
	res := ssz.MixInLength(sliceRoot, sliceLenRoot)
	return res, nil
}

func PendingPartialWithdrawalsRoot(slice []*ethpb.PendingPartialWithdrawal) ([32]byte, error) {
	max := uint64(fieldparams.PendingPartialWithdrawalsLimit)
	if uint64(len(slice)) > max {
		return [32]byte{}, fmt.Errorf("pending partial withdrawals exceeds max length %d", max)
	}

	roots := make([][32]byte, len(slice))
	for i := 0; i < len(slice); i++ {
		r, err := slice[i].HashTreeRoot()
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not merkleize pending partial withdrawals")
		}
		roots[i] = r
	}

	sliceRoot, err := ssz.BitwiseMerkleize(roots, uint64(len(roots)), fieldparams.PendingPartialWithdrawalsLimit)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute pending partial withdrawals merkleization")
	}
	sliceLenBuf := new(bytes.Buffer)
	if err := binary.Write(sliceLenBuf, binary.LittleEndian, uint64(len(slice))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal pending partial withdrawals length")
	}
	// We need to mix in the length of the slice.
	sliceLenRoot := make([]byte, 32)
	copy(sliceLenRoot, sliceLenBuf.Bytes())
	res := ssz.MixInLength(sliceRoot, sliceLenRoot)
	return res, nil
}

func PendingConsolidationsRoot(slice []*ethpb.PendingConsolidation) ([32]byte, error) {
	max := uint64(fieldparams.PendingConsolidationsLimit)
	if uint64(len(slice)) > max {
		return [32]byte{}, fmt.Errorf("pending consolidations exceeds max length %d", max)
	}

	roots := make([][32]byte, len(slice))
	for i := 0; i < len(slice); i++ {
		r, err := slice[i].HashTreeRoot()
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not merkleize pending consolidations")
		}
		roots[i] = r
	}

	sliceRoot, err := ssz.BitwiseMerkleize(roots, uint64(len(roots)), fieldparams.PendingConsolidationsLimit)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute pending consolidations merkleization")
	}
	sliceLenBuf := new(bytes.Buffer)
	if err := binary.Write(sliceLenBuf, binary.LittleEndian, uint64(len(slice))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal pending consolidations length")
	}
	// We need to mix in the length of the slice.
	sliceLenRoot := make([]byte, 32)
	copy(sliceLenRoot, sliceLenBuf.Bytes())
	res := ssz.MixInLength(sliceRoot, sliceLenRoot)
	return res, nil
}
