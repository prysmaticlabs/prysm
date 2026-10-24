//go:build !minimal

package eth

import (
	binary "encoding/binary"
	"fmt"
	go_bitfield "github.com/OffchainLabs/go-bitfield"
	ssz "github.com/OffchainLabs/methodical-ssz/ssz"
	primitives "github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	v1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
)

func (c *HeightCheckpoint) SizeSSZ() int {
	size := 40

	return size
}

func (c *HeightCheckpoint) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *HeightCheckpoint) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Height
	if dst, err = c.Height.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Height: %w", err)
	}

	// Field 1: Root
	if len(c.Root) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Root...)

	return dst, err
}

func (c *HeightCheckpoint) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 40 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:8]  // c.Height
	sszSlice1 := buf[8:40] // c.Root

	// Field 0: Height
	if err = c.Height.UnmarshalSSZ(sszSlice0); err != nil {
		return fmt.Errorf("Height: %w", err)
	}

	// Field 1: Root
	c.Root = make([]byte, 0, 32)
	c.Root = append(c.Root, sszSlice1...)
	return err
}

func (c *HeightCheckpoint) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *HeightCheckpoint) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Height
	if err := c.Height.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Height: %w", err)
	}
	// Field 1: Root
	if len(c.Root) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Root)
	hh.Merkleize(indx)
	return nil
}

func (c *PendingFinalityEIP8453) SizeSSZ() int {
	size := 52
	size += len(c.Participation)
	return size
}

func (c *PendingFinalityEIP8453) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *PendingFinalityEIP8453) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 52

	// Field 0: Checkpoint
	if c.Checkpoint == nil {
		c.Checkpoint = new(HeightCheckpoint)
	}
	if dst, err = c.Checkpoint.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Checkpoint: %w", err)
	}

	// Field 1: Slot
	if dst, err = c.Slot.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Slot: %w", err)
	}

	// Field 2: Participation
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.Participation)

	// Field 2: Participation
	dst = append(dst, c.Participation...)
	return dst, err
}

func (c *PendingFinalityEIP8453) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 52 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:40]  // c.Checkpoint
	sszSlice1 := buf[40:48] // c.Slot

	sszVarOffset2 := ssz.ReadOffset(buf[48:52]) // c.Participation
	if sszVarOffset2 != 52 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset2 > size {
		return ssz.ErrOffset
	}
	sszSlice2 := buf[sszVarOffset2:] // c.Participation

	// Field 0: Checkpoint
	c.Checkpoint = new(HeightCheckpoint)
	if err = c.Checkpoint.UnmarshalSSZ(sszSlice0); err != nil {
		return fmt.Errorf("Checkpoint: %w", err)
	}

	// Field 1: Slot
	if err = c.Slot.UnmarshalSSZ(sszSlice1); err != nil {
		return fmt.Errorf("Slot: %w", err)
	}

	// Field 2: Participation
	c.Participation = append([]byte{}, sszSlice2...)
	return err
}

func (c *PendingFinalityEIP8453) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *PendingFinalityEIP8453) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Checkpoint
	if err := c.Checkpoint.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Checkpoint: %w", err)
	}
	// Field 1: Slot
	if err := c.Slot.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Slot: %w", err)
	}
	// Field 2: Participation
	{
		subIndx := hh.Index()
		hh.AppendBytes32(c.Participation)
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.Participation)))
	}
	hh.Merkleize(indx)
	return nil
}

func (c *HeadVoteData) SizeSSZ() int {
	size := 40

	return size
}

func (c *HeadVoteData) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *HeadVoteData) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Slot
	if dst, err = c.Slot.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Slot: %w", err)
	}

	// Field 1: BeaconBlockRoot
	if len(c.BeaconBlockRoot) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.BeaconBlockRoot...)

	return dst, err
}

func (c *HeadVoteData) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 40 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:8]  // c.Slot
	sszSlice1 := buf[8:40] // c.BeaconBlockRoot

	// Field 0: Slot
	if err = c.Slot.UnmarshalSSZ(sszSlice0); err != nil {
		return fmt.Errorf("Slot: %w", err)
	}

	// Field 1: BeaconBlockRoot
	c.BeaconBlockRoot = make([]byte, 0, 32)
	c.BeaconBlockRoot = append(c.BeaconBlockRoot, sszSlice1...)
	return err
}

func (c *HeadVoteData) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *HeadVoteData) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Slot
	if err := c.Slot.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Slot: %w", err)
	}
	// Field 1: BeaconBlockRoot
	if len(c.BeaconBlockRoot) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.BeaconBlockRoot)
	hh.Merkleize(indx)
	return nil
}

func (c *HeadVote) SizeSSZ() int {
	size := 264

	return size
}

func (c *HeadVote) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *HeadVote) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: AggregationBits
	if len(c.AggregationBits) != 128 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.AggregationBits...)

	// Field 1: Data
	if c.Data == nil {
		c.Data = new(HeadVoteData)
	}
	if dst, err = c.Data.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Data: %w", err)
	}

	// Field 2: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	return dst, err
}

func (c *HeadVote) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 264 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:128]   // c.AggregationBits
	sszSlice1 := buf[128:168] // c.Data
	sszSlice2 := buf[168:264] // c.Signature

	// Field 0: AggregationBits
	c.AggregationBits = make([]byte, 0, 128)
	c.AggregationBits = append(c.AggregationBits, sszSlice0...)

	// Field 1: Data
	c.Data = new(HeadVoteData)
	if err = c.Data.UnmarshalSSZ(sszSlice1); err != nil {
		return fmt.Errorf("Data: %w", err)
	}

	// Field 2: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, sszSlice2...)
	return err
}

func (c *HeadVote) HashTreeRoot() ([32]byte, error) {
	return c.ProgressiveHashTreeRoot()
}

func (c *HeadVote) HashTreeRootWith(hh *ssz.Hasher) error {
	return c.ProgressiveHashTreeRootWith(hh)
}

var activeFieldsHeadVote = []byte{0b00000111}

func (c *HeadVote) ProgressiveHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.ProgressiveHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *HeadVote) ProgressiveHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Position 0: AggregationBits
	if len(c.AggregationBits) != 128 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.AggregationBits)
	// Position 1: Data
	if err := c.Data.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Data: %w", err)
	}
	// Position 2: Signature
	if len(c.Signature) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Signature)
	hh.MerkleizeProgressiveWithActiveFields(indx, activeFieldsHeadVote)
	return nil
}

func (c *AttestationDataEIP8453) SizeSSZ() int {
	size := 81

	return size
}

func (c *AttestationDataEIP8453) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *AttestationDataEIP8453) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Target
	if c.Target == nil {
		c.Target = new(HeightCheckpoint)
	}
	if dst, err = c.Target.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Target: %w", err)
	}

	// Field 1: Timeout
	if c.Timeout {
		dst = append(dst, 1)
	} else {
		dst = append(dst, 0)
	}

	// Field 2: Finality
	if c.Finality == nil {
		c.Finality = new(HeightCheckpoint)
	}
	if dst, err = c.Finality.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Finality: %w", err)
	}

	return dst, err
}

func (c *AttestationDataEIP8453) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 81 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:40]  // c.Target
	sszSlice1 := buf[40:41] // c.Timeout
	sszSlice2 := buf[41:81] // c.Finality

	// Field 0: Target
	c.Target = new(HeightCheckpoint)
	if err = c.Target.UnmarshalSSZ(sszSlice0); err != nil {
		return fmt.Errorf("Target: %w", err)
	}

	// Field 1: Timeout
	if sszSlice1[0] > 1 {
		return ssz.ErrInvalidSerialization
	}
	if sszSlice1[0] == 1 {
		c.Timeout = true
	} else {
		c.Timeout = false
	}

	// Field 2: Finality
	c.Finality = new(HeightCheckpoint)
	if err = c.Finality.UnmarshalSSZ(sszSlice2); err != nil {
		return fmt.Errorf("Finality: %w", err)
	}
	return err
}

func (c *AttestationDataEIP8453) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *AttestationDataEIP8453) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Target
	if err := c.Target.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Target: %w", err)
	}
	// Field 1: Timeout
	hh.PutBool(c.Timeout)
	// Field 2: Finality
	if err := c.Finality.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Finality: %w", err)
	}
	hh.Merkleize(indx)
	return nil
}

func (c *AttestationEIP8453) SizeSSZ() int {
	size := 445
	size += len(c.AggregationBits)
	return size
}

func (c *AttestationEIP8453) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *AttestationEIP8453) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 445

	// Field 0: AggregationBits
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.AggregationBits)

	// Field 1: Data
	if c.Data == nil {
		c.Data = new(AttestationDataEIP8453)
	}
	if dst, err = c.Data.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Data: %w", err)
	}

	// Field 2: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	// Field 3: CommitteeBits
	if len(c.CommitteeBits) != 256 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.CommitteeBits...)

	// Field 4: Epoch
	if dst, err = c.Epoch.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Epoch: %w", err)
	}

	// Field 0: AggregationBits
	dst = append(dst, c.AggregationBits...)
	return dst, err
}

func (c *AttestationEIP8453) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 445 {
		return ssz.ErrSize
	}

	sszSlice1 := buf[4:85]    // c.Data
	sszSlice2 := buf[85:181]  // c.Signature
	sszSlice3 := buf[181:437] // c.CommitteeBits
	sszSlice4 := buf[437:445] // c.Epoch

	sszVarOffset0 := ssz.ReadOffset(buf[0:4]) // c.AggregationBits
	if sszVarOffset0 != 445 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset0 > size {
		return ssz.ErrOffset
	}
	sszSlice0 := buf[sszVarOffset0:] // c.AggregationBits

	// Field 0: AggregationBits
	if err = ssz.ValidateProgressiveBitlist(sszSlice0); err != nil {
		return fmt.Errorf("AggregationBits: %w", err)
	}
	c.AggregationBits = append([]byte{}, go_bitfield.Bitlist(sszSlice0)...)

	// Field 1: Data
	c.Data = new(AttestationDataEIP8453)
	if err = c.Data.UnmarshalSSZ(sszSlice1); err != nil {
		return fmt.Errorf("Data: %w", err)
	}

	// Field 2: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, sszSlice2...)

	// Field 3: CommitteeBits
	c.CommitteeBits = make([]byte, 0, 256)
	c.CommitteeBits = append(c.CommitteeBits, sszSlice3...)

	// Field 4: Epoch
	if err = c.Epoch.UnmarshalSSZ(sszSlice4); err != nil {
		return fmt.Errorf("Epoch: %w", err)
	}
	return err
}

func (c *AttestationEIP8453) HashTreeRoot() ([32]byte, error) {
	return c.ProgressiveHashTreeRoot()
}

func (c *AttestationEIP8453) HashTreeRootWith(hh *ssz.Hasher) error {
	return c.ProgressiveHashTreeRootWith(hh)
}

var activeFieldsAttestationEIP8453 = []byte{0b00011111}

func (c *AttestationEIP8453) ProgressiveHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.ProgressiveHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *AttestationEIP8453) ProgressiveHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Position 0: AggregationBits
	if len(c.AggregationBits) == 0 {
		return ssz.ErrEmptyBitlist
	}
	hh.PutProgressiveBitlist(c.AggregationBits)
	// Position 1: Data
	if err := c.Data.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Data: %w", err)
	}
	// Position 2: Signature
	if len(c.Signature) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Signature)
	// Position 3: CommitteeBits
	if len(c.CommitteeBits) != 256 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.CommitteeBits)
	// Position 4: Epoch
	if err := c.Epoch.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Epoch: %w", err)
	}
	hh.MerkleizeProgressiveWithActiveFields(indx, activeFieldsAttestationEIP8453)
	return nil
}

func (c *IndexedAttestationEIP8453) SizeSSZ() int {
	size := 189
	size += len(c.AttestingIndices) * 8
	return size
}

func (c *IndexedAttestationEIP8453) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *IndexedAttestationEIP8453) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 189

	// Field 0: AttestingIndices
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.AttestingIndices) * 8

	// Field 1: Data
	if c.Data == nil {
		c.Data = new(AttestationDataEIP8453)
	}
	if dst, err = c.Data.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Data: %w", err)
	}

	// Field 2: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	// Field 3: Epoch
	if dst, err = c.Epoch.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Epoch: %w", err)
	}

	// Field 0: AttestingIndices
	for _, o := range c.AttestingIndices {
		dst = binary.LittleEndian.AppendUint64(dst, o)
	}
	return dst, err
}

func (c *IndexedAttestationEIP8453) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 189 {
		return ssz.ErrSize
	}

	sszSlice1 := buf[4:85]    // c.Data
	sszSlice2 := buf[85:181]  // c.Signature
	sszSlice3 := buf[181:189] // c.Epoch

	sszVarOffset0 := ssz.ReadOffset(buf[0:4]) // c.AttestingIndices
	if sszVarOffset0 != 189 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset0 > size {
		return ssz.ErrOffset
	}
	sszSlice0 := buf[sszVarOffset0:] // c.AttestingIndices

	// Field 0: AttestingIndices
	{
		if len(sszSlice0)%8 != 0 {
			return fmt.Errorf("misaligned bytes: c.AttestingIndices length is %d, which is not a multiple of 8: %w", len(sszSlice0), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice0) / 8
		c.AttestingIndices = make([]uint64, numElem)
		for i := 0; i < numElem; i++ {
			var tmp uint64

			tmpSlice := sszSlice0[i*8 : (1+i)*8]
			tmp = binary.LittleEndian.Uint64(tmpSlice)
			c.AttestingIndices[i] = tmp
		}
	}

	// Field 1: Data
	c.Data = new(AttestationDataEIP8453)
	if err = c.Data.UnmarshalSSZ(sszSlice1); err != nil {
		return fmt.Errorf("Data: %w", err)
	}

	// Field 2: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, sszSlice2...)

	// Field 3: Epoch
	if err = c.Epoch.UnmarshalSSZ(sszSlice3); err != nil {
		return fmt.Errorf("Epoch: %w", err)
	}
	return err
}

func (c *IndexedAttestationEIP8453) HashTreeRoot() ([32]byte, error) {
	return c.ProgressiveHashTreeRoot()
}

func (c *IndexedAttestationEIP8453) HashTreeRootWith(hh *ssz.Hasher) error {
	return c.ProgressiveHashTreeRootWith(hh)
}

var activeFieldsIndexedAttestationEIP8453 = []byte{0b00001111}

func (c *IndexedAttestationEIP8453) ProgressiveHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.ProgressiveHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *IndexedAttestationEIP8453) ProgressiveHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Position 0: AttestingIndices
	{
		subIndx := hh.Index()
		for _, o := range c.AttestingIndices {
			hh.AppendUint64(o)
		}
		hh.FillUpTo32()
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.AttestingIndices)))
	}
	// Position 1: Data
	if err := c.Data.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Data: %w", err)
	}
	// Position 2: Signature
	if len(c.Signature) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Signature)
	// Position 3: Epoch
	if err := c.Epoch.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Epoch: %w", err)
	}
	hh.MerkleizeProgressiveWithActiveFields(indx, activeFieldsIndexedAttestationEIP8453)
	return nil
}

func (c *AttesterSlashingEIP8453) SizeSSZ() int {
	size := 8
	if c.Attestation_1 == nil {
		c.Attestation_1 = new(IndexedAttestationEIP8453)
	}
	size += c.Attestation_1.SizeSSZ()
	if c.Attestation_2 == nil {
		c.Attestation_2 = new(IndexedAttestationEIP8453)
	}
	size += c.Attestation_2.SizeSSZ()
	return size
}

func (c *AttesterSlashingEIP8453) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *AttesterSlashingEIP8453) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 8

	// Field 0: Attestation_1
	if c.Attestation_1 == nil {
		c.Attestation_1 = new(IndexedAttestationEIP8453)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Attestation_1.SizeSSZ()

	// Field 1: Attestation_2
	if c.Attestation_2 == nil {
		c.Attestation_2 = new(IndexedAttestationEIP8453)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Attestation_2.SizeSSZ()

	// Field 0: Attestation_1
	if dst, err = c.Attestation_1.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Attestation_1: %w", err)
	}

	// Field 1: Attestation_2
	if dst, err = c.Attestation_2.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Attestation_2: %w", err)
	}
	return dst, err
}

func (c *AttesterSlashingEIP8453) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 8 {
		return ssz.ErrSize
	}

	sszVarOffset0 := ssz.ReadOffset(buf[0:4]) // c.Attestation_1
	if sszVarOffset0 != 8 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset0 > size {
		return ssz.ErrOffset
	}
	sszVarOffset1 := ssz.ReadOffset(buf[4:8]) // c.Attestation_2
	if sszVarOffset1 > size || sszVarOffset1 < sszVarOffset0 {
		return ssz.ErrOffset
	}
	sszSlice0 := buf[sszVarOffset0:sszVarOffset1] // c.Attestation_1
	sszSlice1 := buf[sszVarOffset1:]              // c.Attestation_2

	// Field 0: Attestation_1
	c.Attestation_1 = new(IndexedAttestationEIP8453)
	if err = c.Attestation_1.UnmarshalSSZ(sszSlice0); err != nil {
		return fmt.Errorf("Attestation_1: %w", err)
	}

	// Field 1: Attestation_2
	c.Attestation_2 = new(IndexedAttestationEIP8453)
	if err = c.Attestation_2.UnmarshalSSZ(sszSlice1); err != nil {
		return fmt.Errorf("Attestation_2: %w", err)
	}
	return err
}

func (c *AttesterSlashingEIP8453) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *AttesterSlashingEIP8453) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Attestation_1
	if err := c.Attestation_1.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Attestation_1: %w", err)
	}
	// Field 1: Attestation_2
	if err := c.Attestation_2.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Attestation_2: %w", err)
	}
	hh.Merkleize(indx)
	return nil
}

func (c *ExecutionPayloadBidHeze) SizeSSZ() int {
	size := 226
	size += len(c.BlobKzgCommitments) * 48
	return size
}

func (c *ExecutionPayloadBidHeze) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *ExecutionPayloadBidHeze) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 226

	// Field 0: ParentBlockHash
	if len(c.ParentBlockHash) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.ParentBlockHash...)

	// Field 1: ParentBlockRoot
	if len(c.ParentBlockRoot) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.ParentBlockRoot...)

	// Field 2: BlockHash
	if len(c.BlockHash) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.BlockHash...)

	// Field 3: PrevRandao
	if len(c.PrevRandao) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.PrevRandao...)

	// Field 4: FeeRecipient
	if len(c.FeeRecipient) != 20 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.FeeRecipient...)

	// Field 5: GasLimit
	dst = binary.LittleEndian.AppendUint64(dst, c.GasLimit)

	// Field 6: BuilderIndex
	if dst, err = c.BuilderIndex.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("BuilderIndex: %w", err)
	}

	// Field 7: Slot
	if dst, err = c.Slot.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Slot: %w", err)
	}

	// Field 8: Value
	if dst, err = c.Value.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Value: %w", err)
	}

	// Field 9: ExecutionPayment
	if dst, err = c.ExecutionPayment.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("ExecutionPayment: %w", err)
	}

	// Field 10: BlobKzgCommitments
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.BlobKzgCommitments) * 48

	// Field 11: ExecutionRequestsRoot
	if len(c.ExecutionRequestsRoot) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.ExecutionRequestsRoot...)

	// Field 12: InclusionListBits
	if len(c.InclusionListBits) != 2 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.InclusionListBits...)

	// Field 10: BlobKzgCommitments
	for _, o := range c.BlobKzgCommitments {
		if len(o) != 48 {
			return nil, ssz.ErrBytesLength
		}
		dst = append(dst, o...)
	}
	return dst, err
}

func (c *ExecutionPayloadBidHeze) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 226 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:32]     // c.ParentBlockHash
	sszSlice1 := buf[32:64]    // c.ParentBlockRoot
	sszSlice2 := buf[64:96]    // c.BlockHash
	sszSlice3 := buf[96:128]   // c.PrevRandao
	sszSlice4 := buf[128:148]  // c.FeeRecipient
	sszSlice5 := buf[148:156]  // c.GasLimit
	sszSlice6 := buf[156:164]  // c.BuilderIndex
	sszSlice7 := buf[164:172]  // c.Slot
	sszSlice8 := buf[172:180]  // c.Value
	sszSlice9 := buf[180:188]  // c.ExecutionPayment
	sszSlice11 := buf[192:224] // c.ExecutionRequestsRoot
	sszSlice12 := buf[224:226] // c.InclusionListBits

	sszVarOffset10 := ssz.ReadOffset(buf[188:192]) // c.BlobKzgCommitments
	if sszVarOffset10 != 226 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset10 > size {
		return ssz.ErrOffset
	}
	sszSlice10 := buf[sszVarOffset10:] // c.BlobKzgCommitments

	// Field 0: ParentBlockHash
	c.ParentBlockHash = make([]byte, 0, 32)
	c.ParentBlockHash = append(c.ParentBlockHash, sszSlice0...)

	// Field 1: ParentBlockRoot
	c.ParentBlockRoot = make([]byte, 0, 32)
	c.ParentBlockRoot = append(c.ParentBlockRoot, sszSlice1...)

	// Field 2: BlockHash
	c.BlockHash = make([]byte, 0, 32)
	c.BlockHash = append(c.BlockHash, sszSlice2...)

	// Field 3: PrevRandao
	c.PrevRandao = make([]byte, 0, 32)
	c.PrevRandao = append(c.PrevRandao, sszSlice3...)

	// Field 4: FeeRecipient
	c.FeeRecipient = make([]byte, 0, 20)
	c.FeeRecipient = append(c.FeeRecipient, sszSlice4...)

	// Field 5: GasLimit
	c.GasLimit = binary.LittleEndian.Uint64(sszSlice5)

	// Field 6: BuilderIndex
	if err = c.BuilderIndex.UnmarshalSSZ(sszSlice6); err != nil {
		return fmt.Errorf("BuilderIndex: %w", err)
	}

	// Field 7: Slot
	if err = c.Slot.UnmarshalSSZ(sszSlice7); err != nil {
		return fmt.Errorf("Slot: %w", err)
	}

	// Field 8: Value
	if err = c.Value.UnmarshalSSZ(sszSlice8); err != nil {
		return fmt.Errorf("Value: %w", err)
	}

	// Field 9: ExecutionPayment
	if err = c.ExecutionPayment.UnmarshalSSZ(sszSlice9); err != nil {
		return fmt.Errorf("ExecutionPayment: %w", err)
	}

	// Field 10: BlobKzgCommitments
	{
		if len(sszSlice10)%48 != 0 {
			return fmt.Errorf("misaligned bytes: c.BlobKzgCommitments length is %d, which is not a multiple of 48: %w", len(sszSlice10), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice10) / 48
		if numElem > 4096 {
			return fmt.Errorf("ssz-max exceeded: c.BlobKzgCommitments has %d elements, ssz-max is 4096: %w", numElem, ssz.ErrListTooBig)
		}
		c.BlobKzgCommitments = make([][]byte, numElem)
		for i := 0; i < numElem; i++ {
			var tmp []byte

			tmpSlice := sszSlice10[i*48 : (1+i)*48]
			tmp = make([]byte, 0, 48)
			tmp = append(tmp, tmpSlice...)
			c.BlobKzgCommitments[i] = tmp
		}
	}

	// Field 11: ExecutionRequestsRoot
	c.ExecutionRequestsRoot = make([]byte, 0, 32)
	c.ExecutionRequestsRoot = append(c.ExecutionRequestsRoot, sszSlice11...)

	// Field 12: InclusionListBits
	c.InclusionListBits = make([]byte, 0, 2)
	c.InclusionListBits = append(c.InclusionListBits, sszSlice12...)
	return err
}

func (c *ExecutionPayloadBidHeze) HashTreeRoot() ([32]byte, error) {
	return c.ProgressiveHashTreeRoot()
}

func (c *ExecutionPayloadBidHeze) HashTreeRootWith(hh *ssz.Hasher) error {
	return c.ProgressiveHashTreeRootWith(hh)
}

var activeFieldsExecutionPayloadBidHeze = []byte{0b11111111, 0b00011111}

func (c *ExecutionPayloadBidHeze) ProgressiveHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.ProgressiveHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *ExecutionPayloadBidHeze) ProgressiveHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Position 0: ParentBlockHash
	if len(c.ParentBlockHash) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.ParentBlockHash)
	// Position 1: ParentBlockRoot
	if len(c.ParentBlockRoot) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.ParentBlockRoot)
	// Position 2: BlockHash
	if len(c.BlockHash) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.BlockHash)
	// Position 3: PrevRandao
	if len(c.PrevRandao) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.PrevRandao)
	// Position 4: FeeRecipient
	if len(c.FeeRecipient) != 20 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.FeeRecipient)
	// Position 5: GasLimit
	hh.PutUint64(c.GasLimit)
	// Position 6: BuilderIndex
	if err := c.BuilderIndex.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("BuilderIndex: %w", err)
	}
	// Position 7: Slot
	if err := c.Slot.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Slot: %w", err)
	}
	// Position 8: Value
	if err := c.Value.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Value: %w", err)
	}
	// Position 9: ExecutionPayment
	if err := c.ExecutionPayment.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("ExecutionPayment: %w", err)
	}
	// Position 10: BlobKzgCommitments
	{
		subIndx := hh.Index()
		for _, o := range c.BlobKzgCommitments {
			if len(o) != 48 {
				return ssz.ErrBytesLength
			}
			hh.PutBytes(o)
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.BlobKzgCommitments)))
	}
	// Position 11: ExecutionRequestsRoot
	if len(c.ExecutionRequestsRoot) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.ExecutionRequestsRoot)
	// Position 12: InclusionListBits
	if len(c.InclusionListBits) != 2 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.InclusionListBits)
	hh.MerkleizeProgressiveWithActiveFields(indx, activeFieldsExecutionPayloadBidHeze)
	return nil
}

func (c *SignedExecutionPayloadBidHeze) SizeSSZ() int {
	size := 100
	if c.Message == nil {
		c.Message = new(ExecutionPayloadBidHeze)
	}
	size += c.Message.SizeSSZ()
	return size
}

func (c *SignedExecutionPayloadBidHeze) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *SignedExecutionPayloadBidHeze) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 100

	// Field 0: Message
	if c.Message == nil {
		c.Message = new(ExecutionPayloadBidHeze)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Message.SizeSSZ()

	// Field 1: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	// Field 0: Message
	if dst, err = c.Message.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Message: %w", err)
	}
	return dst, err
}

func (c *SignedExecutionPayloadBidHeze) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 100 {
		return ssz.ErrSize
	}

	sszSlice1 := buf[4:100] // c.Signature

	sszVarOffset0 := ssz.ReadOffset(buf[0:4]) // c.Message
	if sszVarOffset0 != 100 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset0 > size {
		return ssz.ErrOffset
	}
	sszSlice0 := buf[sszVarOffset0:] // c.Message

	// Field 0: Message
	c.Message = new(ExecutionPayloadBidHeze)
	if err = c.Message.UnmarshalSSZ(sszSlice0); err != nil {
		return fmt.Errorf("Message: %w", err)
	}

	// Field 1: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, sszSlice1...)
	return err
}

func (c *SignedExecutionPayloadBidHeze) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *SignedExecutionPayloadBidHeze) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Message
	if err := c.Message.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Message: %w", err)
	}
	// Field 1: Signature
	if len(c.Signature) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Signature)
	hh.Merkleize(indx)
	return nil
}

func (c *BeaconStateEIP8453) SizeSSZ() int {
	size := 3142968
	size += len(c.HistoricalRoots) * 32
	size += len(c.Validators) * 121
	size += len(c.Balances) * 8
	size += len(c.InactivityScores) * 8
	size += len(c.HistoricalSummaries) * 64
	size += len(c.PendingDeposits) * 192
	size += len(c.PendingPartialWithdrawals) * 24
	size += len(c.PendingConsolidations) * 16
	size += len(c.Builders) * 93
	size += len(c.BuilderPendingWithdrawals) * 36
	if c.LatestExecutionPayloadBid == nil {
		c.LatestExecutionPayloadBid = new(ExecutionPayloadBidHeze)
	}
	size += c.LatestExecutionPayloadBid.SizeSSZ()
	size += len(c.PayloadExpectedWithdrawals) * 44
	size += len(c.TargetParticipation)
	size += len(c.ProgressParticipation)
	size += len(c.RoundParticipation)
	for _, o := range c.PendingFinality {
		size += 4
		size += o.SizeSSZ()
	}
	return size
}

func (c *BeaconStateEIP8453) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *BeaconStateEIP8453) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 3142968

	// Field 0: GenesisTime
	dst = binary.LittleEndian.AppendUint64(dst, c.GenesisTime)

	// Field 1: GenesisValidatorsRoot
	if len(c.GenesisValidatorsRoot) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.GenesisValidatorsRoot...)

	// Field 2: Slot
	if dst, err = c.Slot.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Slot: %w", err)
	}

	// Field 3: Fork
	if c.Fork == nil {
		c.Fork = new(Fork)
	}
	if dst, err = c.Fork.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Fork: %w", err)
	}

	// Field 4: LatestBlockHeader
	if c.LatestBlockHeader == nil {
		c.LatestBlockHeader = new(BeaconBlockHeader)
	}
	if dst, err = c.LatestBlockHeader.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("LatestBlockHeader: %w", err)
	}

	// Field 5: BlockRoots
	if len(c.BlockRoots) != 8192 {
		return nil, ssz.ErrBytesLength
	}
	for _, o := range c.BlockRoots {
		if len(o) != 32 {
			return nil, ssz.ErrBytesLength
		}
		dst = append(dst, o...)
	}

	// Field 6: StateRoots
	if len(c.StateRoots) != 8192 {
		return nil, ssz.ErrBytesLength
	}
	for _, o := range c.StateRoots {
		if len(o) != 32 {
			return nil, ssz.ErrBytesLength
		}
		dst = append(dst, o...)
	}

	// Field 7: HistoricalRoots
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.HistoricalRoots) * 32

	// Field 8: Validators
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.Validators) * 121

	// Field 9: Balances
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.Balances) * 8

	// Field 10: RandaoMixes
	if len(c.RandaoMixes) != 65536 {
		return nil, ssz.ErrBytesLength
	}
	for _, o := range c.RandaoMixes {
		if len(o) != 32 {
			return nil, ssz.ErrBytesLength
		}
		dst = append(dst, o...)
	}

	// Field 11: Slashings
	if len(c.Slashings) != 8192 {
		return nil, ssz.ErrBytesLength
	}
	for _, o := range c.Slashings {
		dst = binary.LittleEndian.AppendUint64(dst, o)
	}

	// Field 12: InactivityScores
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.InactivityScores) * 8

	// Field 13: CurrentSyncCommittee
	if c.CurrentSyncCommittee == nil {
		c.CurrentSyncCommittee = new(SyncCommittee)
	}
	if dst, err = c.CurrentSyncCommittee.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("CurrentSyncCommittee: %w", err)
	}

	// Field 14: NextSyncCommittee
	if c.NextSyncCommittee == nil {
		c.NextSyncCommittee = new(SyncCommittee)
	}
	if dst, err = c.NextSyncCommittee.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("NextSyncCommittee: %w", err)
	}

	// Field 15: LatestBlockHash
	if len(c.LatestBlockHash) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.LatestBlockHash...)

	// Field 16: NextWithdrawalIndex
	dst = binary.LittleEndian.AppendUint64(dst, c.NextWithdrawalIndex)

	// Field 17: NextWithdrawalValidatorIndex
	if dst, err = c.NextWithdrawalValidatorIndex.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("NextWithdrawalValidatorIndex: %w", err)
	}

	// Field 18: HistoricalSummaries
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.HistoricalSummaries) * 64

	// Field 19: DepositBalanceToConsume
	if dst, err = c.DepositBalanceToConsume.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("DepositBalanceToConsume: %w", err)
	}

	// Field 20: ExitBalanceToConsume
	if dst, err = c.ExitBalanceToConsume.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("ExitBalanceToConsume: %w", err)
	}

	// Field 21: EarliestExitEpoch
	if dst, err = c.EarliestExitEpoch.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("EarliestExitEpoch: %w", err)
	}

	// Field 22: ConsolidationBalanceToConsume
	if dst, err = c.ConsolidationBalanceToConsume.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("ConsolidationBalanceToConsume: %w", err)
	}

	// Field 23: EarliestConsolidationEpoch
	if dst, err = c.EarliestConsolidationEpoch.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("EarliestConsolidationEpoch: %w", err)
	}

	// Field 24: PendingDeposits
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.PendingDeposits) * 192

	// Field 25: PendingPartialWithdrawals
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.PendingPartialWithdrawals) * 24

	// Field 26: PendingConsolidations
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.PendingConsolidations) * 16

	// Field 27: ProposerLookahead
	if len(c.ProposerLookahead) != 64 {
		return nil, ssz.ErrBytesLength
	}
	for _, o := range c.ProposerLookahead {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("ProposerLookahead: %w", err)
		}
	}

	// Field 28: Builders
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.Builders) * 93

	// Field 29: NextWithdrawalBuilderIndex
	if dst, err = c.NextWithdrawalBuilderIndex.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("NextWithdrawalBuilderIndex: %w", err)
	}

	// Field 30: ExecutionPayloadAvailability
	if len(c.ExecutionPayloadAvailability) != 1024 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.ExecutionPayloadAvailability...)

	// Field 31: BuilderPendingPayments
	if len(c.BuilderPendingPayments) != 64 {
		return nil, ssz.ErrBytesLength
	}
	for _, o := range c.BuilderPendingPayments {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("BuilderPendingPayments: %w", err)
		}
	}

	// Field 32: BuilderPendingWithdrawals
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.BuilderPendingWithdrawals) * 36

	// Field 33: LatestExecutionPayloadBid
	if c.LatestExecutionPayloadBid == nil {
		c.LatestExecutionPayloadBid = new(ExecutionPayloadBidHeze)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.LatestExecutionPayloadBid.SizeSSZ()

	// Field 34: PayloadExpectedWithdrawals
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.PayloadExpectedWithdrawals) * 44

	// Field 35: PtcWindow
	if len(c.PtcWindow) != 96 {
		return nil, ssz.ErrBytesLength
	}
	for _, o := range c.PtcWindow {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("PtcWindow: %w", err)
		}
	}

	// Field 36: TargetCheckpoint
	if c.TargetCheckpoint == nil {
		c.TargetCheckpoint = new(HeightCheckpoint)
	}
	if dst, err = c.TargetCheckpoint.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("TargetCheckpoint: %w", err)
	}

	// Field 37: JustifiedCheckpoint
	if c.JustifiedCheckpoint == nil {
		c.JustifiedCheckpoint = new(HeightCheckpoint)
	}
	if dst, err = c.JustifiedCheckpoint.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("JustifiedCheckpoint: %w", err)
	}

	// Field 38: FinalizedCheckpoint
	if c.FinalizedCheckpoint == nil {
		c.FinalizedCheckpoint = new(HeightCheckpoint)
	}
	if dst, err = c.FinalizedCheckpoint.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("FinalizedCheckpoint: %w", err)
	}

	// Field 39: TargetSlot
	if dst, err = c.TargetSlot.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("TargetSlot: %w", err)
	}

	// Field 40: FinalizedSlot
	if dst, err = c.FinalizedSlot.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("FinalizedSlot: %w", err)
	}

	// Field 41: TargetParticipation
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.TargetParticipation)

	// Field 42: ProgressParticipation
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.ProgressParticipation)

	// Field 43: RoundParticipation
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.RoundParticipation)

	// Field 44: PendingFinality
	dst = ssz.WriteOffset(dst, offset)
	for _, o := range c.PendingFinality {
		offset += 4
		offset += o.SizeSSZ()
	}

	// Field 45: HeadVoteParticipation
	if len(c.HeadVoteParticipation) != 64 {
		return nil, ssz.ErrBytesLength
	}
	for _, o := range c.HeadVoteParticipation {
		if len(o) != 128 {
			return nil, ssz.ErrBytesLength
		}
		dst = append(dst, o...)
	}

	// Field 7: HistoricalRoots
	if len(c.HistoricalRoots) > 16777216 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.HistoricalRoots {
		if len(o) != 32 {
			return nil, ssz.ErrBytesLength
		}
		dst = append(dst, o...)
	}

	// Field 8: Validators
	for _, o := range c.Validators {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("Validators: %w", err)
		}
	}

	// Field 9: Balances
	for _, o := range c.Balances {
		dst = binary.LittleEndian.AppendUint64(dst, o)
	}

	// Field 12: InactivityScores
	for _, o := range c.InactivityScores {
		dst = binary.LittleEndian.AppendUint64(dst, o)
	}

	// Field 18: HistoricalSummaries
	if len(c.HistoricalSummaries) > 16777216 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.HistoricalSummaries {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("HistoricalSummaries: %w", err)
		}
	}

	// Field 24: PendingDeposits
	for _, o := range c.PendingDeposits {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("PendingDeposits: %w", err)
		}
	}

	// Field 25: PendingPartialWithdrawals
	for _, o := range c.PendingPartialWithdrawals {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("PendingPartialWithdrawals: %w", err)
		}
	}

	// Field 26: PendingConsolidations
	for _, o := range c.PendingConsolidations {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("PendingConsolidations: %w", err)
		}
	}

	// Field 28: Builders
	for _, o := range c.Builders {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("Builders: %w", err)
		}
	}

	// Field 32: BuilderPendingWithdrawals
	for _, o := range c.BuilderPendingWithdrawals {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("BuilderPendingWithdrawals: %w", err)
		}
	}

	// Field 33: LatestExecutionPayloadBid
	if dst, err = c.LatestExecutionPayloadBid.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("LatestExecutionPayloadBid: %w", err)
	}

	// Field 34: PayloadExpectedWithdrawals
	for _, o := range c.PayloadExpectedWithdrawals {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("PayloadExpectedWithdrawals: %w", err)
		}
	}

	// Field 41: TargetParticipation
	dst = append(dst, c.TargetParticipation...)

	// Field 42: ProgressParticipation
	dst = append(dst, c.ProgressParticipation...)

	// Field 43: RoundParticipation
	dst = append(dst, c.RoundParticipation...)

	// Field 44: PendingFinality
	if len(c.PendingFinality) > 4 {
		return nil, ssz.ErrListTooBig
	}
	{
		offset = 4 * len(c.PendingFinality)
		for _, o := range c.PendingFinality {
			dst = ssz.WriteOffset(dst, offset)
			offset += o.SizeSSZ()
		}
	}
	for _, o := range c.PendingFinality {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("PendingFinality: %w", err)
		}
	}
	return dst, err
}

func (c *BeaconStateEIP8453) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 3142968 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:8]              // c.GenesisTime
	sszSlice1 := buf[8:40]             // c.GenesisValidatorsRoot
	sszSlice2 := buf[40:48]            // c.Slot
	sszSlice3 := buf[48:64]            // c.Fork
	sszSlice4 := buf[64:176]           // c.LatestBlockHeader
	sszSlice5 := buf[176:262320]       // c.BlockRoots
	sszSlice6 := buf[262320:524464]    // c.StateRoots
	sszSlice10 := buf[524476:2621628]  // c.RandaoMixes
	sszSlice11 := buf[2621628:2687164] // c.Slashings
	sszSlice13 := buf[2687168:2711792] // c.CurrentSyncCommittee
	sszSlice14 := buf[2711792:2736416] // c.NextSyncCommittee
	sszSlice15 := buf[2736416:2736448] // c.LatestBlockHash
	sszSlice16 := buf[2736448:2736456] // c.NextWithdrawalIndex
	sszSlice17 := buf[2736456:2736464] // c.NextWithdrawalValidatorIndex
	sszSlice19 := buf[2736468:2736476] // c.DepositBalanceToConsume
	sszSlice20 := buf[2736476:2736484] // c.ExitBalanceToConsume
	sszSlice21 := buf[2736484:2736492] // c.EarliestExitEpoch
	sszSlice22 := buf[2736492:2736500] // c.ConsolidationBalanceToConsume
	sszSlice23 := buf[2736500:2736508] // c.EarliestConsolidationEpoch
	sszSlice27 := buf[2736520:2737032] // c.ProposerLookahead
	sszSlice29 := buf[2737036:2737044] // c.NextWithdrawalBuilderIndex
	sszSlice30 := buf[2737044:2738068] // c.ExecutionPayloadAvailability
	sszSlice31 := buf[2738068:2741396] // c.BuilderPendingPayments
	sszSlice35 := buf[2741408:3134624] // c.PtcWindow
	sszSlice36 := buf[3134624:3134664] // c.TargetCheckpoint
	sszSlice37 := buf[3134664:3134704] // c.JustifiedCheckpoint
	sszSlice38 := buf[3134704:3134744] // c.FinalizedCheckpoint
	sszSlice39 := buf[3134744:3134752] // c.TargetSlot
	sszSlice40 := buf[3134752:3134760] // c.FinalizedSlot
	sszSlice45 := buf[3134776:3142968] // c.HeadVoteParticipation

	sszVarOffset7 := ssz.ReadOffset(buf[524464:524468]) // c.HistoricalRoots
	if sszVarOffset7 != 3142968 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset7 > size {
		return ssz.ErrOffset
	}
	sszVarOffset8 := ssz.ReadOffset(buf[524468:524472]) // c.Validators
	if sszVarOffset8 > size || sszVarOffset8 < sszVarOffset7 {
		return ssz.ErrOffset
	}
	sszVarOffset9 := ssz.ReadOffset(buf[524472:524476]) // c.Balances
	if sszVarOffset9 > size || sszVarOffset9 < sszVarOffset8 {
		return ssz.ErrOffset
	}
	sszVarOffset12 := ssz.ReadOffset(buf[2687164:2687168]) // c.InactivityScores
	if sszVarOffset12 > size || sszVarOffset12 < sszVarOffset9 {
		return ssz.ErrOffset
	}
	sszVarOffset18 := ssz.ReadOffset(buf[2736464:2736468]) // c.HistoricalSummaries
	if sszVarOffset18 > size || sszVarOffset18 < sszVarOffset12 {
		return ssz.ErrOffset
	}
	sszVarOffset24 := ssz.ReadOffset(buf[2736508:2736512]) // c.PendingDeposits
	if sszVarOffset24 > size || sszVarOffset24 < sszVarOffset18 {
		return ssz.ErrOffset
	}
	sszVarOffset25 := ssz.ReadOffset(buf[2736512:2736516]) // c.PendingPartialWithdrawals
	if sszVarOffset25 > size || sszVarOffset25 < sszVarOffset24 {
		return ssz.ErrOffset
	}
	sszVarOffset26 := ssz.ReadOffset(buf[2736516:2736520]) // c.PendingConsolidations
	if sszVarOffset26 > size || sszVarOffset26 < sszVarOffset25 {
		return ssz.ErrOffset
	}
	sszVarOffset28 := ssz.ReadOffset(buf[2737032:2737036]) // c.Builders
	if sszVarOffset28 > size || sszVarOffset28 < sszVarOffset26 {
		return ssz.ErrOffset
	}
	sszVarOffset32 := ssz.ReadOffset(buf[2741396:2741400]) // c.BuilderPendingWithdrawals
	if sszVarOffset32 > size || sszVarOffset32 < sszVarOffset28 {
		return ssz.ErrOffset
	}
	sszVarOffset33 := ssz.ReadOffset(buf[2741400:2741404]) // c.LatestExecutionPayloadBid
	if sszVarOffset33 > size || sszVarOffset33 < sszVarOffset32 {
		return ssz.ErrOffset
	}
	sszVarOffset34 := ssz.ReadOffset(buf[2741404:2741408]) // c.PayloadExpectedWithdrawals
	if sszVarOffset34 > size || sszVarOffset34 < sszVarOffset33 {
		return ssz.ErrOffset
	}
	sszVarOffset41 := ssz.ReadOffset(buf[3134760:3134764]) // c.TargetParticipation
	if sszVarOffset41 > size || sszVarOffset41 < sszVarOffset34 {
		return ssz.ErrOffset
	}
	sszVarOffset42 := ssz.ReadOffset(buf[3134764:3134768]) // c.ProgressParticipation
	if sszVarOffset42 > size || sszVarOffset42 < sszVarOffset41 {
		return ssz.ErrOffset
	}
	sszVarOffset43 := ssz.ReadOffset(buf[3134768:3134772]) // c.RoundParticipation
	if sszVarOffset43 > size || sszVarOffset43 < sszVarOffset42 {
		return ssz.ErrOffset
	}
	sszVarOffset44 := ssz.ReadOffset(buf[3134772:3134776]) // c.PendingFinality
	if sszVarOffset44 > size || sszVarOffset44 < sszVarOffset43 {
		return ssz.ErrOffset
	}
	sszSlice7 := buf[sszVarOffset7:sszVarOffset8]    // c.HistoricalRoots
	sszSlice8 := buf[sszVarOffset8:sszVarOffset9]    // c.Validators
	sszSlice9 := buf[sszVarOffset9:sszVarOffset12]   // c.Balances
	sszSlice12 := buf[sszVarOffset12:sszVarOffset18] // c.InactivityScores
	sszSlice18 := buf[sszVarOffset18:sszVarOffset24] // c.HistoricalSummaries
	sszSlice24 := buf[sszVarOffset24:sszVarOffset25] // c.PendingDeposits
	sszSlice25 := buf[sszVarOffset25:sszVarOffset26] // c.PendingPartialWithdrawals
	sszSlice26 := buf[sszVarOffset26:sszVarOffset28] // c.PendingConsolidations
	sszSlice28 := buf[sszVarOffset28:sszVarOffset32] // c.Builders
	sszSlice32 := buf[sszVarOffset32:sszVarOffset33] // c.BuilderPendingWithdrawals
	sszSlice33 := buf[sszVarOffset33:sszVarOffset34] // c.LatestExecutionPayloadBid
	sszSlice34 := buf[sszVarOffset34:sszVarOffset41] // c.PayloadExpectedWithdrawals
	sszSlice41 := buf[sszVarOffset41:sszVarOffset42] // c.TargetParticipation
	sszSlice42 := buf[sszVarOffset42:sszVarOffset43] // c.ProgressParticipation
	sszSlice43 := buf[sszVarOffset43:sszVarOffset44] // c.RoundParticipation
	sszSlice44 := buf[sszVarOffset44:]               // c.PendingFinality

	// Field 0: GenesisTime
	c.GenesisTime = binary.LittleEndian.Uint64(sszSlice0)

	// Field 1: GenesisValidatorsRoot
	c.GenesisValidatorsRoot = make([]byte, 0, 32)
	c.GenesisValidatorsRoot = append(c.GenesisValidatorsRoot, sszSlice1...)

	// Field 2: Slot
	if err = c.Slot.UnmarshalSSZ(sszSlice2); err != nil {
		return fmt.Errorf("Slot: %w", err)
	}

	// Field 3: Fork
	c.Fork = new(Fork)
	if err = c.Fork.UnmarshalSSZ(sszSlice3); err != nil {
		return fmt.Errorf("Fork: %w", err)
	}

	// Field 4: LatestBlockHeader
	c.LatestBlockHeader = new(BeaconBlockHeader)
	if err = c.LatestBlockHeader.UnmarshalSSZ(sszSlice4); err != nil {
		return fmt.Errorf("LatestBlockHeader: %w", err)
	}

	// Field 5: BlockRoots
	{
		c.BlockRoots = make([][]byte, 8192)
		for i := 0; i < 8192; i++ {
			var tmp []byte

			tmpSlice := sszSlice5[i*32 : (1+i)*32]
			tmp = make([]byte, 0, 32)
			tmp = append(tmp, tmpSlice...)
			c.BlockRoots[i] = tmp
		}
	}

	// Field 6: StateRoots
	{
		c.StateRoots = make([][]byte, 8192)
		for i := 0; i < 8192; i++ {
			var tmp []byte

			tmpSlice := sszSlice6[i*32 : (1+i)*32]
			tmp = make([]byte, 0, 32)
			tmp = append(tmp, tmpSlice...)
			c.StateRoots[i] = tmp
		}
	}

	// Field 7: HistoricalRoots
	{
		if len(sszSlice7)%32 != 0 {
			return fmt.Errorf("misaligned bytes: c.HistoricalRoots length is %d, which is not a multiple of 32: %w", len(sszSlice7), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice7) / 32
		if numElem > 16777216 {
			return fmt.Errorf("ssz-max exceeded: c.HistoricalRoots has %d elements, ssz-max is 16777216: %w", numElem, ssz.ErrListTooBig)
		}
		c.HistoricalRoots = make([][]byte, numElem)
		for i := 0; i < numElem; i++ {
			var tmp []byte

			tmpSlice := sszSlice7[i*32 : (1+i)*32]
			tmp = make([]byte, 0, 32)
			tmp = append(tmp, tmpSlice...)
			c.HistoricalRoots[i] = tmp
		}
	}

	// Field 8: Validators
	{
		if len(sszSlice8)%121 != 0 {
			return fmt.Errorf("misaligned bytes: c.Validators length is %d, which is not a multiple of 121: %w", len(sszSlice8), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice8) / 121
		if numElem > 1099511627776 {
			return fmt.Errorf("ssz-max exceeded: c.Validators has %d elements, ssz-max is 1099511627776: %w", numElem, ssz.ErrListTooBig)
		}
		c.Validators = make([]*Validator, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *Validator
			tmp = new(Validator)
			tmpSlice := sszSlice8[i*121 : (1+i)*121]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("Validators: %w", err)
			}
			c.Validators[i] = tmp
		}
	}

	// Field 9: Balances
	{
		if len(sszSlice9)%8 != 0 {
			return fmt.Errorf("misaligned bytes: c.Balances length is %d, which is not a multiple of 8: %w", len(sszSlice9), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice9) / 8
		if numElem > 1099511627776 {
			return fmt.Errorf("ssz-max exceeded: c.Balances has %d elements, ssz-max is 1099511627776: %w", numElem, ssz.ErrListTooBig)
		}
		c.Balances = make([]uint64, numElem)
		for i := 0; i < numElem; i++ {
			var tmp uint64

			tmpSlice := sszSlice9[i*8 : (1+i)*8]
			tmp = binary.LittleEndian.Uint64(tmpSlice)
			c.Balances[i] = tmp
		}
	}

	// Field 10: RandaoMixes
	{
		c.RandaoMixes = make([][]byte, 65536)
		for i := 0; i < 65536; i++ {
			var tmp []byte

			tmpSlice := sszSlice10[i*32 : (1+i)*32]
			tmp = make([]byte, 0, 32)
			tmp = append(tmp, tmpSlice...)
			c.RandaoMixes[i] = tmp
		}
	}

	// Field 11: Slashings
	{
		c.Slashings = make([]uint64, 8192)
		for i := 0; i < 8192; i++ {
			var tmp uint64

			tmpSlice := sszSlice11[i*8 : (1+i)*8]
			tmp = binary.LittleEndian.Uint64(tmpSlice)
			c.Slashings[i] = tmp
		}
	}

	// Field 12: InactivityScores
	{
		if len(sszSlice12)%8 != 0 {
			return fmt.Errorf("misaligned bytes: c.InactivityScores length is %d, which is not a multiple of 8: %w", len(sszSlice12), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice12) / 8
		if numElem > 1099511627776 {
			return fmt.Errorf("ssz-max exceeded: c.InactivityScores has %d elements, ssz-max is 1099511627776: %w", numElem, ssz.ErrListTooBig)
		}
		c.InactivityScores = make([]uint64, numElem)
		for i := 0; i < numElem; i++ {
			var tmp uint64

			tmpSlice := sszSlice12[i*8 : (1+i)*8]
			tmp = binary.LittleEndian.Uint64(tmpSlice)
			c.InactivityScores[i] = tmp
		}
	}

	// Field 13: CurrentSyncCommittee
	c.CurrentSyncCommittee = new(SyncCommittee)
	if err = c.CurrentSyncCommittee.UnmarshalSSZ(sszSlice13); err != nil {
		return fmt.Errorf("CurrentSyncCommittee: %w", err)
	}

	// Field 14: NextSyncCommittee
	c.NextSyncCommittee = new(SyncCommittee)
	if err = c.NextSyncCommittee.UnmarshalSSZ(sszSlice14); err != nil {
		return fmt.Errorf("NextSyncCommittee: %w", err)
	}

	// Field 15: LatestBlockHash
	c.LatestBlockHash = make([]byte, 0, 32)
	c.LatestBlockHash = append(c.LatestBlockHash, sszSlice15...)

	// Field 16: NextWithdrawalIndex
	c.NextWithdrawalIndex = binary.LittleEndian.Uint64(sszSlice16)

	// Field 17: NextWithdrawalValidatorIndex
	if err = c.NextWithdrawalValidatorIndex.UnmarshalSSZ(sszSlice17); err != nil {
		return fmt.Errorf("NextWithdrawalValidatorIndex: %w", err)
	}

	// Field 18: HistoricalSummaries
	{
		if len(sszSlice18)%64 != 0 {
			return fmt.Errorf("misaligned bytes: c.HistoricalSummaries length is %d, which is not a multiple of 64: %w", len(sszSlice18), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice18) / 64
		if numElem > 16777216 {
			return fmt.Errorf("ssz-max exceeded: c.HistoricalSummaries has %d elements, ssz-max is 16777216: %w", numElem, ssz.ErrListTooBig)
		}
		c.HistoricalSummaries = make([]*HistoricalSummary, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *HistoricalSummary
			tmp = new(HistoricalSummary)
			tmpSlice := sszSlice18[i*64 : (1+i)*64]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("HistoricalSummaries: %w", err)
			}
			c.HistoricalSummaries[i] = tmp
		}
	}

	// Field 19: DepositBalanceToConsume
	if err = c.DepositBalanceToConsume.UnmarshalSSZ(sszSlice19); err != nil {
		return fmt.Errorf("DepositBalanceToConsume: %w", err)
	}

	// Field 20: ExitBalanceToConsume
	if err = c.ExitBalanceToConsume.UnmarshalSSZ(sszSlice20); err != nil {
		return fmt.Errorf("ExitBalanceToConsume: %w", err)
	}

	// Field 21: EarliestExitEpoch
	if err = c.EarliestExitEpoch.UnmarshalSSZ(sszSlice21); err != nil {
		return fmt.Errorf("EarliestExitEpoch: %w", err)
	}

	// Field 22: ConsolidationBalanceToConsume
	if err = c.ConsolidationBalanceToConsume.UnmarshalSSZ(sszSlice22); err != nil {
		return fmt.Errorf("ConsolidationBalanceToConsume: %w", err)
	}

	// Field 23: EarliestConsolidationEpoch
	if err = c.EarliestConsolidationEpoch.UnmarshalSSZ(sszSlice23); err != nil {
		return fmt.Errorf("EarliestConsolidationEpoch: %w", err)
	}

	// Field 24: PendingDeposits
	{
		if len(sszSlice24)%192 != 0 {
			return fmt.Errorf("misaligned bytes: c.PendingDeposits length is %d, which is not a multiple of 192: %w", len(sszSlice24), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice24) / 192
		if numElem > 134217728 {
			return fmt.Errorf("ssz-max exceeded: c.PendingDeposits has %d elements, ssz-max is 134217728: %w", numElem, ssz.ErrListTooBig)
		}
		c.PendingDeposits = make([]*PendingDeposit, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *PendingDeposit
			tmp = new(PendingDeposit)
			tmpSlice := sszSlice24[i*192 : (1+i)*192]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("PendingDeposits: %w", err)
			}
			c.PendingDeposits[i] = tmp
		}
	}

	// Field 25: PendingPartialWithdrawals
	{
		if len(sszSlice25)%24 != 0 {
			return fmt.Errorf("misaligned bytes: c.PendingPartialWithdrawals length is %d, which is not a multiple of 24: %w", len(sszSlice25), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice25) / 24
		if numElem > 134217728 {
			return fmt.Errorf("ssz-max exceeded: c.PendingPartialWithdrawals has %d elements, ssz-max is 134217728: %w", numElem, ssz.ErrListTooBig)
		}
		c.PendingPartialWithdrawals = make([]*PendingPartialWithdrawal, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *PendingPartialWithdrawal
			tmp = new(PendingPartialWithdrawal)
			tmpSlice := sszSlice25[i*24 : (1+i)*24]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("PendingPartialWithdrawals: %w", err)
			}
			c.PendingPartialWithdrawals[i] = tmp
		}
	}

	// Field 26: PendingConsolidations
	{
		if len(sszSlice26)%16 != 0 {
			return fmt.Errorf("misaligned bytes: c.PendingConsolidations length is %d, which is not a multiple of 16: %w", len(sszSlice26), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice26) / 16
		if numElem > 262144 {
			return fmt.Errorf("ssz-max exceeded: c.PendingConsolidations has %d elements, ssz-max is 262144: %w", numElem, ssz.ErrListTooBig)
		}
		c.PendingConsolidations = make([]*PendingConsolidation, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *PendingConsolidation
			tmp = new(PendingConsolidation)
			tmpSlice := sszSlice26[i*16 : (1+i)*16]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("PendingConsolidations: %w", err)
			}
			c.PendingConsolidations[i] = tmp
		}
	}

	// Field 27: ProposerLookahead
	{
		c.ProposerLookahead = make([]primitives.ValidatorIndex, 64)
		for i := 0; i < 64; i++ {
			var tmp primitives.ValidatorIndex

			tmpSlice := sszSlice27[i*8 : (1+i)*8]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("ProposerLookahead: %w", err)
			}
			c.ProposerLookahead[i] = tmp
		}
	}

	// Field 28: Builders
	{
		if len(sszSlice28)%93 != 0 {
			return fmt.Errorf("misaligned bytes: c.Builders length is %d, which is not a multiple of 93: %w", len(sszSlice28), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice28) / 93
		if numElem > 1099511627776 {
			return fmt.Errorf("ssz-max exceeded: c.Builders has %d elements, ssz-max is 1099511627776: %w", numElem, ssz.ErrListTooBig)
		}
		c.Builders = make([]*Builder, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *Builder
			tmp = new(Builder)
			tmpSlice := sszSlice28[i*93 : (1+i)*93]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("Builders: %w", err)
			}
			c.Builders[i] = tmp
		}
	}

	// Field 29: NextWithdrawalBuilderIndex
	if err = c.NextWithdrawalBuilderIndex.UnmarshalSSZ(sszSlice29); err != nil {
		return fmt.Errorf("NextWithdrawalBuilderIndex: %w", err)
	}

	// Field 30: ExecutionPayloadAvailability
	c.ExecutionPayloadAvailability = make([]byte, 0, 1024)
	c.ExecutionPayloadAvailability = append(c.ExecutionPayloadAvailability, sszSlice30...)

	// Field 31: BuilderPendingPayments
	{
		c.BuilderPendingPayments = make([]*BuilderPendingPayment, 64)
		for i := 0; i < 64; i++ {
			var tmp *BuilderPendingPayment
			tmp = new(BuilderPendingPayment)
			tmpSlice := sszSlice31[i*52 : (1+i)*52]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("BuilderPendingPayments: %w", err)
			}
			c.BuilderPendingPayments[i] = tmp
		}
	}

	// Field 32: BuilderPendingWithdrawals
	{
		if len(sszSlice32)%36 != 0 {
			return fmt.Errorf("misaligned bytes: c.BuilderPendingWithdrawals length is %d, which is not a multiple of 36: %w", len(sszSlice32), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice32) / 36
		if numElem > 1048576 {
			return fmt.Errorf("ssz-max exceeded: c.BuilderPendingWithdrawals has %d elements, ssz-max is 1048576: %w", numElem, ssz.ErrListTooBig)
		}
		c.BuilderPendingWithdrawals = make([]*BuilderPendingWithdrawal, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *BuilderPendingWithdrawal
			tmp = new(BuilderPendingWithdrawal)
			tmpSlice := sszSlice32[i*36 : (1+i)*36]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("BuilderPendingWithdrawals: %w", err)
			}
			c.BuilderPendingWithdrawals[i] = tmp
		}
	}

	// Field 33: LatestExecutionPayloadBid
	c.LatestExecutionPayloadBid = new(ExecutionPayloadBidHeze)
	if err = c.LatestExecutionPayloadBid.UnmarshalSSZ(sszSlice33); err != nil {
		return fmt.Errorf("LatestExecutionPayloadBid: %w", err)
	}

	// Field 34: PayloadExpectedWithdrawals
	{
		if len(sszSlice34)%44 != 0 {
			return fmt.Errorf("misaligned bytes: c.PayloadExpectedWithdrawals length is %d, which is not a multiple of 44: %w", len(sszSlice34), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice34) / 44
		if numElem > 16 {
			return fmt.Errorf("ssz-max exceeded: c.PayloadExpectedWithdrawals has %d elements, ssz-max is 16: %w", numElem, ssz.ErrListTooBig)
		}
		c.PayloadExpectedWithdrawals = make([]*v1.Withdrawal, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *v1.Withdrawal
			tmp = new(v1.Withdrawal)
			tmpSlice := sszSlice34[i*44 : (1+i)*44]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("PayloadExpectedWithdrawals: %w", err)
			}
			c.PayloadExpectedWithdrawals[i] = tmp
		}
	}

	// Field 35: PtcWindow
	{
		c.PtcWindow = make([]*PTCs, 96)
		for i := 0; i < 96; i++ {
			var tmp *PTCs
			tmp = new(PTCs)
			tmpSlice := sszSlice35[i*4096 : (1+i)*4096]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("PtcWindow: %w", err)
			}
			c.PtcWindow[i] = tmp
		}
	}

	// Field 36: TargetCheckpoint
	c.TargetCheckpoint = new(HeightCheckpoint)
	if err = c.TargetCheckpoint.UnmarshalSSZ(sszSlice36); err != nil {
		return fmt.Errorf("TargetCheckpoint: %w", err)
	}

	// Field 37: JustifiedCheckpoint
	c.JustifiedCheckpoint = new(HeightCheckpoint)
	if err = c.JustifiedCheckpoint.UnmarshalSSZ(sszSlice37); err != nil {
		return fmt.Errorf("JustifiedCheckpoint: %w", err)
	}

	// Field 38: FinalizedCheckpoint
	c.FinalizedCheckpoint = new(HeightCheckpoint)
	if err = c.FinalizedCheckpoint.UnmarshalSSZ(sszSlice38); err != nil {
		return fmt.Errorf("FinalizedCheckpoint: %w", err)
	}

	// Field 39: TargetSlot
	if err = c.TargetSlot.UnmarshalSSZ(sszSlice39); err != nil {
		return fmt.Errorf("TargetSlot: %w", err)
	}

	// Field 40: FinalizedSlot
	if err = c.FinalizedSlot.UnmarshalSSZ(sszSlice40); err != nil {
		return fmt.Errorf("FinalizedSlot: %w", err)
	}

	// Field 41: TargetParticipation
	c.TargetParticipation = append([]byte{}, sszSlice41...)

	// Field 42: ProgressParticipation
	c.ProgressParticipation = append([]byte{}, sszSlice42...)

	// Field 43: RoundParticipation
	c.RoundParticipation = append([]byte{}, sszSlice43...)

	// Field 44: PendingFinality
	{
		// empty lists are zero length, so make sure there is room for an offset
		// before attempting to unmarshal it
		if len(sszSlice44) > 3 {
			startOffset := ssz.ReadOffset(sszSlice44[0:4])
			if startOffset == 0 {
				return fmt.Errorf("encountered invalid offset of 0 when decoding c.PendingFinality")
			}
			if startOffset%4 != 0 {
				return fmt.Errorf("misaligned list bytes: when decoding c.PendingFinality, end-of-list offset is %d, which is not a multiple of 4 (offset size)", startOffset)
			}
			listLen := startOffset / 4
			if listLen > 4 {
				return fmt.Errorf("ssz-max exceeded: c.PendingFinality has %d elements, ssz-max is 4: %w", listLen, ssz.ErrListTooBig)
			}
			totalVarBytes := uint64(len(sszSlice44))
			if totalVarBytes < startOffset {
				return fmt.Errorf("list bytes too short to contain an offset when decoding c.PendingFinality")
			}
			c.PendingFinality = make([]*PendingFinalityEIP8453, listLen)
			var tmpSlice []byte
			for i := uint64(0); i < listLen; i++ {
				var tmp *PendingFinalityEIP8453
				tmp = new(PendingFinalityEIP8453)
				endOffset := totalVarBytes
				if i+1 != listLen {
					endOffset = ssz.ReadOffset(sszSlice44[(i+1)*4 : (i+2)*4])
					if totalVarBytes < endOffset {
						return fmt.Errorf("offset %d points past the end of buffer when decoding c.PendingFinality", endOffset)
					}
				}
				if endOffset < startOffset {
					return fmt.Errorf("offset %d is not greater than start offset %d when decoding c.PendingFinality", endOffset, startOffset)
				}
				tmpSlice = sszSlice44[startOffset:endOffset]
				if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
					return fmt.Errorf("PendingFinality: %w", err)
				}
				c.PendingFinality[i] = tmp
				startOffset = endOffset
			}
		} else {
			if len(sszSlice44) > 0 {
				return fmt.Errorf("list bytes too short to contain an offset when decoding c.PendingFinality")
			}
			c.PendingFinality = make([]*PendingFinalityEIP8453, 0)
		}
	}

	// Field 45: HeadVoteParticipation
	{
		c.HeadVoteParticipation = make([][]byte, 64)
		for i := 0; i < 64; i++ {
			var tmp []byte

			tmpSlice := sszSlice45[i*128 : (1+i)*128]
			tmp = make([]byte, 0, 128)
			tmp = append(tmp, tmpSlice...)
			c.HeadVoteParticipation[i] = tmp
		}
	}
	return err
}

func (c *BeaconStateEIP8453) HashTreeRoot() ([32]byte, error) {
	return c.ProgressiveHashTreeRoot()
}

func (c *BeaconStateEIP8453) HashTreeRootWith(hh *ssz.Hasher) error {
	return c.ProgressiveHashTreeRootWith(hh)
}

var activeFieldsBeaconStateEIP8453 = []byte{0b11111111, 0b01111000, 0b11100000, 0b11101111, 0b11111111, 0b11111111, 0b11111111}

func (c *BeaconStateEIP8453) ProgressiveHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.ProgressiveHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BeaconStateEIP8453) ProgressiveHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Position 0: GenesisTime
	hh.PutUint64(c.GenesisTime)
	// Position 1: GenesisValidatorsRoot
	if len(c.GenesisValidatorsRoot) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.GenesisValidatorsRoot)
	// Position 2: Slot
	if err := c.Slot.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Slot: %w", err)
	}
	// Position 3: Fork
	if err := c.Fork.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Fork: %w", err)
	}
	// Position 4: LatestBlockHeader
	if err := c.LatestBlockHeader.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("LatestBlockHeader: %w", err)
	}
	// Position 5: BlockRoots
	{
		if len(c.BlockRoots) != 8192 {
			return ssz.ErrVectorLength
		}
		subIndx := hh.Index()
		for _, o := range c.BlockRoots {
			if len(o) != 32 {
				return ssz.ErrBytesLength
			}
			hh.Append(o)
		}
		hh.Merkleize(subIndx)
	}
	// Position 6: StateRoots
	{
		if len(c.StateRoots) != 8192 {
			return ssz.ErrVectorLength
		}
		subIndx := hh.Index()
		for _, o := range c.StateRoots {
			if len(o) != 32 {
				return ssz.ErrBytesLength
			}
			hh.Append(o)
		}
		hh.Merkleize(subIndx)
	}
	// Position 7: HistoricalRoots
	{
		if len(c.HistoricalRoots) > 16777216 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.HistoricalRoots {
			if len(o) != 32 {
				return ssz.ErrBytesLength
			}
			hh.Append(o)
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.HistoricalRoots)), 16777216)
	}
	// Position 8: inactive (zero chunk)
	hh.PutBool(false)
	// Position 9: inactive (zero chunk)
	hh.PutBool(false)
	// Position 10: inactive (zero chunk)
	hh.PutBool(false)
	// Position 11: Validators
	{
		subIndx := hh.Index()
		for _, o := range c.Validators {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("Validators: %w", err)
			}
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.Validators)))
	}
	// Position 12: Balances
	{
		subIndx := hh.Index()
		for _, o := range c.Balances {
			hh.AppendUint64(o)
		}
		hh.FillUpTo32()
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.Balances)))
	}
	// Position 13: RandaoMixes
	{
		if len(c.RandaoMixes) != 65536 {
			return ssz.ErrVectorLength
		}
		subIndx := hh.Index()
		for _, o := range c.RandaoMixes {
			if len(o) != 32 {
				return ssz.ErrBytesLength
			}
			hh.Append(o)
		}
		hh.Merkleize(subIndx)
	}
	// Position 14: Slashings
	{
		if len(c.Slashings) != 8192 {
			return ssz.ErrVectorLength
		}
		subIndx := hh.Index()
		for _, o := range c.Slashings {
			hh.AppendUint64(o)
		}
		hh.Merkleize(subIndx)
	}
	// Position 15: inactive (zero chunk)
	hh.PutBool(false)
	// Position 16: inactive (zero chunk)
	hh.PutBool(false)
	// Position 17: inactive (zero chunk)
	hh.PutBool(false)
	// Position 18: inactive (zero chunk)
	hh.PutBool(false)
	// Position 19: inactive (zero chunk)
	hh.PutBool(false)
	// Position 20: inactive (zero chunk)
	hh.PutBool(false)
	// Position 21: InactivityScores
	{
		subIndx := hh.Index()
		for _, o := range c.InactivityScores {
			hh.AppendUint64(o)
		}
		hh.FillUpTo32()
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.InactivityScores)))
	}
	// Position 22: CurrentSyncCommittee
	if err := c.CurrentSyncCommittee.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("CurrentSyncCommittee: %w", err)
	}
	// Position 23: NextSyncCommittee
	if err := c.NextSyncCommittee.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("NextSyncCommittee: %w", err)
	}
	// Position 24: LatestBlockHash
	if len(c.LatestBlockHash) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.LatestBlockHash)
	// Position 25: NextWithdrawalIndex
	hh.PutUint64(c.NextWithdrawalIndex)
	// Position 26: NextWithdrawalValidatorIndex
	if err := c.NextWithdrawalValidatorIndex.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("NextWithdrawalValidatorIndex: %w", err)
	}
	// Position 27: HistoricalSummaries
	{
		if len(c.HistoricalSummaries) > 16777216 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.HistoricalSummaries {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("HistoricalSummaries: %w", err)
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.HistoricalSummaries)), 16777216)
	}
	// Position 28: inactive (zero chunk)
	hh.PutBool(false)
	// Position 29: DepositBalanceToConsume
	if err := c.DepositBalanceToConsume.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("DepositBalanceToConsume: %w", err)
	}
	// Position 30: ExitBalanceToConsume
	if err := c.ExitBalanceToConsume.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("ExitBalanceToConsume: %w", err)
	}
	// Position 31: EarliestExitEpoch
	if err := c.EarliestExitEpoch.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("EarliestExitEpoch: %w", err)
	}
	// Position 32: ConsolidationBalanceToConsume
	if err := c.ConsolidationBalanceToConsume.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("ConsolidationBalanceToConsume: %w", err)
	}
	// Position 33: EarliestConsolidationEpoch
	if err := c.EarliestConsolidationEpoch.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("EarliestConsolidationEpoch: %w", err)
	}
	// Position 34: PendingDeposits
	{
		subIndx := hh.Index()
		for _, o := range c.PendingDeposits {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("PendingDeposits: %w", err)
			}
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.PendingDeposits)))
	}
	// Position 35: PendingPartialWithdrawals
	{
		subIndx := hh.Index()
		for _, o := range c.PendingPartialWithdrawals {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("PendingPartialWithdrawals: %w", err)
			}
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.PendingPartialWithdrawals)))
	}
	// Position 36: PendingConsolidations
	{
		subIndx := hh.Index()
		for _, o := range c.PendingConsolidations {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("PendingConsolidations: %w", err)
			}
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.PendingConsolidations)))
	}
	// Position 37: ProposerLookahead
	{
		if len(c.ProposerLookahead) != 64 {
			return ssz.ErrVectorLength
		}
		subIndx := hh.Index()
		for _, o := range c.ProposerLookahead {
			hh.AppendUint64(uint64(o))
		}
		hh.Merkleize(subIndx)
	}
	// Position 38: Builders
	{
		subIndx := hh.Index()
		for _, o := range c.Builders {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("Builders: %w", err)
			}
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.Builders)))
	}
	// Position 39: NextWithdrawalBuilderIndex
	if err := c.NextWithdrawalBuilderIndex.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("NextWithdrawalBuilderIndex: %w", err)
	}
	// Position 40: ExecutionPayloadAvailability
	if len(c.ExecutionPayloadAvailability) != 1024 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.ExecutionPayloadAvailability)
	// Position 41: BuilderPendingPayments
	{
		if len(c.BuilderPendingPayments) != 64 {
			return ssz.ErrVectorLength
		}
		subIndx := hh.Index()
		for _, o := range c.BuilderPendingPayments {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("BuilderPendingPayments: %w", err)
			}
		}
		hh.Merkleize(subIndx)
	}
	// Position 42: BuilderPendingWithdrawals
	{
		subIndx := hh.Index()
		for _, o := range c.BuilderPendingWithdrawals {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("BuilderPendingWithdrawals: %w", err)
			}
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.BuilderPendingWithdrawals)))
	}
	// Position 43: LatestExecutionPayloadBid
	if err := c.LatestExecutionPayloadBid.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("LatestExecutionPayloadBid: %w", err)
	}
	// Position 44: PayloadExpectedWithdrawals
	{
		subIndx := hh.Index()
		for _, o := range c.PayloadExpectedWithdrawals {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("PayloadExpectedWithdrawals: %w", err)
			}
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.PayloadExpectedWithdrawals)))
	}
	// Position 45: PtcWindow
	{
		if len(c.PtcWindow) != 96 {
			return ssz.ErrVectorLength
		}
		subIndx := hh.Index()
		for _, o := range c.PtcWindow {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("PtcWindow: %w", err)
			}
		}
		hh.Merkleize(subIndx)
	}
	// Position 46: TargetCheckpoint
	if err := c.TargetCheckpoint.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("TargetCheckpoint: %w", err)
	}
	// Position 47: JustifiedCheckpoint
	if err := c.JustifiedCheckpoint.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("JustifiedCheckpoint: %w", err)
	}
	// Position 48: FinalizedCheckpoint
	if err := c.FinalizedCheckpoint.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("FinalizedCheckpoint: %w", err)
	}
	// Position 49: TargetSlot
	if err := c.TargetSlot.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("TargetSlot: %w", err)
	}
	// Position 50: FinalizedSlot
	if err := c.FinalizedSlot.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("FinalizedSlot: %w", err)
	}
	// Position 51: TargetParticipation
	{
		subIndx := hh.Index()
		hh.AppendBytes32(c.TargetParticipation)
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.TargetParticipation)))
	}
	// Position 52: ProgressParticipation
	{
		subIndx := hh.Index()
		hh.AppendBytes32(c.ProgressParticipation)
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.ProgressParticipation)))
	}
	// Position 53: RoundParticipation
	{
		subIndx := hh.Index()
		hh.AppendBytes32(c.RoundParticipation)
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.RoundParticipation)))
	}
	// Position 54: PendingFinality
	{
		if len(c.PendingFinality) > 4 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.PendingFinality {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("PendingFinality: %w", err)
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.PendingFinality)), 4)
	}
	// Position 55: HeadVoteParticipation
	{
		if len(c.HeadVoteParticipation) != 64 {
			return ssz.ErrVectorLength
		}
		subIndx := hh.Index()
		for _, o := range c.HeadVoteParticipation {
			if len(o) != 128 {
				return ssz.ErrBytesLength
			}
			hh.PutBytes(o)
		}
		hh.Merkleize(subIndx)
	}
	hh.MerkleizeProgressiveWithActiveFields(indx, activeFieldsBeaconStateEIP8453)
	return nil
}

func (c *BeaconBlockBodyEIP8453) SizeSSZ() int {
	size := 332
	size += len(c.ProposerSlashings) * 416
	for _, o := range c.AttesterSlashings {
		size += 4
		size += o.SizeSSZ()
	}
	for _, o := range c.Attestations {
		size += 4
		size += o.SizeSSZ()
	}
	size += len(c.VoluntaryExits) * 112
	size += len(c.BlsToExecutionChanges) * 172
	if c.SignedExecutionPayloadBid == nil {
		c.SignedExecutionPayloadBid = new(SignedExecutionPayloadBidHeze)
	}
	size += c.SignedExecutionPayloadBid.SizeSSZ()
	size += len(c.PayloadAttestations) * 202
	if c.ParentExecutionRequests == nil {
		c.ParentExecutionRequests = new(v1.ExecutionRequestsGloas)
	}
	size += c.ParentExecutionRequests.SizeSSZ()
	size += len(c.HeadVotes) * 264
	size += len(c.HeadVoteSupportBits)
	for _, o := range c.LegacyAttesterSlashings {
		size += 4
		size += o.SizeSSZ()
	}
	return size
}

func (c *BeaconBlockBodyEIP8453) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *BeaconBlockBodyEIP8453) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 332

	// Field 0: RandaoReveal
	if len(c.RandaoReveal) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.RandaoReveal...)

	// Field 1: Graffiti
	if len(c.Graffiti) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Graffiti...)

	// Field 2: ProposerSlashings
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.ProposerSlashings) * 416

	// Field 3: AttesterSlashings
	dst = ssz.WriteOffset(dst, offset)
	for _, o := range c.AttesterSlashings {
		offset += 4
		offset += o.SizeSSZ()
	}

	// Field 4: Attestations
	dst = ssz.WriteOffset(dst, offset)
	for _, o := range c.Attestations {
		offset += 4
		offset += o.SizeSSZ()
	}

	// Field 5: VoluntaryExits
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.VoluntaryExits) * 112

	// Field 6: SyncAggregate
	if c.SyncAggregate == nil {
		c.SyncAggregate = new(SyncAggregate)
	}
	if dst, err = c.SyncAggregate.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("SyncAggregate: %w", err)
	}

	// Field 7: BlsToExecutionChanges
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.BlsToExecutionChanges) * 172

	// Field 8: SignedExecutionPayloadBid
	if c.SignedExecutionPayloadBid == nil {
		c.SignedExecutionPayloadBid = new(SignedExecutionPayloadBidHeze)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.SignedExecutionPayloadBid.SizeSSZ()

	// Field 9: PayloadAttestations
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.PayloadAttestations) * 202

	// Field 10: ParentExecutionRequests
	if c.ParentExecutionRequests == nil {
		c.ParentExecutionRequests = new(v1.ExecutionRequestsGloas)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.ParentExecutionRequests.SizeSSZ()

	// Field 11: HeadVotes
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.HeadVotes) * 264

	// Field 12: HeadVoteSupportBits
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.HeadVoteSupportBits)

	// Field 13: LegacyAttesterSlashings
	dst = ssz.WriteOffset(dst, offset)
	for _, o := range c.LegacyAttesterSlashings {
		offset += 4
		offset += o.SizeSSZ()
	}

	// Field 2: ProposerSlashings
	for _, o := range c.ProposerSlashings {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("ProposerSlashings: %w", err)
		}
	}

	// Field 3: AttesterSlashings
	{
		offset = 4 * len(c.AttesterSlashings)
		for _, o := range c.AttesterSlashings {
			dst = ssz.WriteOffset(dst, offset)
			offset += o.SizeSSZ()
		}
	}
	for _, o := range c.AttesterSlashings {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("AttesterSlashings: %w", err)
		}
	}

	// Field 4: Attestations
	{
		offset = 4 * len(c.Attestations)
		for _, o := range c.Attestations {
			dst = ssz.WriteOffset(dst, offset)
			offset += o.SizeSSZ()
		}
	}
	for _, o := range c.Attestations {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("Attestations: %w", err)
		}
	}

	// Field 5: VoluntaryExits
	for _, o := range c.VoluntaryExits {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("VoluntaryExits: %w", err)
		}
	}

	// Field 7: BlsToExecutionChanges
	for _, o := range c.BlsToExecutionChanges {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("BlsToExecutionChanges: %w", err)
		}
	}

	// Field 8: SignedExecutionPayloadBid
	if dst, err = c.SignedExecutionPayloadBid.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("SignedExecutionPayloadBid: %w", err)
	}

	// Field 9: PayloadAttestations
	for _, o := range c.PayloadAttestations {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("PayloadAttestations: %w", err)
		}
	}

	// Field 10: ParentExecutionRequests
	if dst, err = c.ParentExecutionRequests.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("ParentExecutionRequests: %w", err)
	}

	// Field 11: HeadVotes
	for _, o := range c.HeadVotes {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("HeadVotes: %w", err)
		}
	}

	// Field 12: HeadVoteSupportBits
	dst = append(dst, c.HeadVoteSupportBits...)

	// Field 13: LegacyAttesterSlashings
	{
		offset = 4 * len(c.LegacyAttesterSlashings)
		for _, o := range c.LegacyAttesterSlashings {
			dst = ssz.WriteOffset(dst, offset)
			offset += o.SizeSSZ()
		}
	}
	for _, o := range c.LegacyAttesterSlashings {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, fmt.Errorf("LegacyAttesterSlashings: %w", err)
		}
	}
	return dst, err
}

func (c *BeaconBlockBodyEIP8453) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 332 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:96]    // c.RandaoReveal
	sszSlice1 := buf[96:128]  // c.Graffiti
	sszSlice6 := buf[144:304] // c.SyncAggregate

	sszVarOffset2 := ssz.ReadOffset(buf[128:132]) // c.ProposerSlashings
	if sszVarOffset2 != 332 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset2 > size {
		return ssz.ErrOffset
	}
	sszVarOffset3 := ssz.ReadOffset(buf[132:136]) // c.AttesterSlashings
	if sszVarOffset3 > size || sszVarOffset3 < sszVarOffset2 {
		return ssz.ErrOffset
	}
	sszVarOffset4 := ssz.ReadOffset(buf[136:140]) // c.Attestations
	if sszVarOffset4 > size || sszVarOffset4 < sszVarOffset3 {
		return ssz.ErrOffset
	}
	sszVarOffset5 := ssz.ReadOffset(buf[140:144]) // c.VoluntaryExits
	if sszVarOffset5 > size || sszVarOffset5 < sszVarOffset4 {
		return ssz.ErrOffset
	}
	sszVarOffset7 := ssz.ReadOffset(buf[304:308]) // c.BlsToExecutionChanges
	if sszVarOffset7 > size || sszVarOffset7 < sszVarOffset5 {
		return ssz.ErrOffset
	}
	sszVarOffset8 := ssz.ReadOffset(buf[308:312]) // c.SignedExecutionPayloadBid
	if sszVarOffset8 > size || sszVarOffset8 < sszVarOffset7 {
		return ssz.ErrOffset
	}
	sszVarOffset9 := ssz.ReadOffset(buf[312:316]) // c.PayloadAttestations
	if sszVarOffset9 > size || sszVarOffset9 < sszVarOffset8 {
		return ssz.ErrOffset
	}
	sszVarOffset10 := ssz.ReadOffset(buf[316:320]) // c.ParentExecutionRequests
	if sszVarOffset10 > size || sszVarOffset10 < sszVarOffset9 {
		return ssz.ErrOffset
	}
	sszVarOffset11 := ssz.ReadOffset(buf[320:324]) // c.HeadVotes
	if sszVarOffset11 > size || sszVarOffset11 < sszVarOffset10 {
		return ssz.ErrOffset
	}
	sszVarOffset12 := ssz.ReadOffset(buf[324:328]) // c.HeadVoteSupportBits
	if sszVarOffset12 > size || sszVarOffset12 < sszVarOffset11 {
		return ssz.ErrOffset
	}
	sszVarOffset13 := ssz.ReadOffset(buf[328:332]) // c.LegacyAttesterSlashings
	if sszVarOffset13 > size || sszVarOffset13 < sszVarOffset12 {
		return ssz.ErrOffset
	}
	sszSlice2 := buf[sszVarOffset2:sszVarOffset3]    // c.ProposerSlashings
	sszSlice3 := buf[sszVarOffset3:sszVarOffset4]    // c.AttesterSlashings
	sszSlice4 := buf[sszVarOffset4:sszVarOffset5]    // c.Attestations
	sszSlice5 := buf[sszVarOffset5:sszVarOffset7]    // c.VoluntaryExits
	sszSlice7 := buf[sszVarOffset7:sszVarOffset8]    // c.BlsToExecutionChanges
	sszSlice8 := buf[sszVarOffset8:sszVarOffset9]    // c.SignedExecutionPayloadBid
	sszSlice9 := buf[sszVarOffset9:sszVarOffset10]   // c.PayloadAttestations
	sszSlice10 := buf[sszVarOffset10:sszVarOffset11] // c.ParentExecutionRequests
	sszSlice11 := buf[sszVarOffset11:sszVarOffset12] // c.HeadVotes
	sszSlice12 := buf[sszVarOffset12:sszVarOffset13] // c.HeadVoteSupportBits
	sszSlice13 := buf[sszVarOffset13:]               // c.LegacyAttesterSlashings

	// Field 0: RandaoReveal
	c.RandaoReveal = make([]byte, 0, 96)
	c.RandaoReveal = append(c.RandaoReveal, sszSlice0...)

	// Field 1: Graffiti
	c.Graffiti = make([]byte, 0, 32)
	c.Graffiti = append(c.Graffiti, sszSlice1...)

	// Field 2: ProposerSlashings
	{
		if len(sszSlice2)%416 != 0 {
			return fmt.Errorf("misaligned bytes: c.ProposerSlashings length is %d, which is not a multiple of 416: %w", len(sszSlice2), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice2) / 416
		c.ProposerSlashings = make([]*ProposerSlashing, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *ProposerSlashing
			tmp = new(ProposerSlashing)
			tmpSlice := sszSlice2[i*416 : (1+i)*416]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("ProposerSlashings: %w", err)
			}
			c.ProposerSlashings[i] = tmp
		}
	}

	// Field 3: AttesterSlashings
	{
		// empty lists are zero length, so make sure there is room for an offset
		// before attempting to unmarshal it
		if len(sszSlice3) > 3 {
			startOffset := ssz.ReadOffset(sszSlice3[0:4])
			if startOffset == 0 {
				return fmt.Errorf("encountered invalid offset of 0 when decoding c.AttesterSlashings")
			}
			if startOffset%4 != 0 {
				return fmt.Errorf("misaligned list bytes: when decoding c.AttesterSlashings, end-of-list offset is %d, which is not a multiple of 4 (offset size)", startOffset)
			}
			listLen := startOffset / 4
			totalVarBytes := uint64(len(sszSlice3))
			if totalVarBytes < startOffset {
				return fmt.Errorf("list bytes too short to contain an offset when decoding c.AttesterSlashings")
			}
			c.AttesterSlashings = make([]*AttesterSlashingEIP8453, listLen)
			var tmpSlice []byte
			for i := uint64(0); i < listLen; i++ {
				var tmp *AttesterSlashingEIP8453
				tmp = new(AttesterSlashingEIP8453)
				endOffset := totalVarBytes
				if i+1 != listLen {
					endOffset = ssz.ReadOffset(sszSlice3[(i+1)*4 : (i+2)*4])
					if totalVarBytes < endOffset {
						return fmt.Errorf("offset %d points past the end of buffer when decoding c.AttesterSlashings", endOffset)
					}
				}
				if endOffset < startOffset {
					return fmt.Errorf("offset %d is not greater than start offset %d when decoding c.AttesterSlashings", endOffset, startOffset)
				}
				tmpSlice = sszSlice3[startOffset:endOffset]
				if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
					return fmt.Errorf("AttesterSlashings: %w", err)
				}
				c.AttesterSlashings[i] = tmp
				startOffset = endOffset
			}
		} else {
			if len(sszSlice3) > 0 {
				return fmt.Errorf("list bytes too short to contain an offset when decoding c.AttesterSlashings")
			}
			c.AttesterSlashings = make([]*AttesterSlashingEIP8453, 0)
		}
	}

	// Field 4: Attestations
	{
		// empty lists are zero length, so make sure there is room for an offset
		// before attempting to unmarshal it
		if len(sszSlice4) > 3 {
			startOffset := ssz.ReadOffset(sszSlice4[0:4])
			if startOffset == 0 {
				return fmt.Errorf("encountered invalid offset of 0 when decoding c.Attestations")
			}
			if startOffset%4 != 0 {
				return fmt.Errorf("misaligned list bytes: when decoding c.Attestations, end-of-list offset is %d, which is not a multiple of 4 (offset size)", startOffset)
			}
			listLen := startOffset / 4
			totalVarBytes := uint64(len(sszSlice4))
			if totalVarBytes < startOffset {
				return fmt.Errorf("list bytes too short to contain an offset when decoding c.Attestations")
			}
			c.Attestations = make([]*AttestationEIP8453, listLen)
			var tmpSlice []byte
			for i := uint64(0); i < listLen; i++ {
				var tmp *AttestationEIP8453
				tmp = new(AttestationEIP8453)
				endOffset := totalVarBytes
				if i+1 != listLen {
					endOffset = ssz.ReadOffset(sszSlice4[(i+1)*4 : (i+2)*4])
					if totalVarBytes < endOffset {
						return fmt.Errorf("offset %d points past the end of buffer when decoding c.Attestations", endOffset)
					}
				}
				if endOffset < startOffset {
					return fmt.Errorf("offset %d is not greater than start offset %d when decoding c.Attestations", endOffset, startOffset)
				}
				tmpSlice = sszSlice4[startOffset:endOffset]
				if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
					return fmt.Errorf("Attestations: %w", err)
				}
				c.Attestations[i] = tmp
				startOffset = endOffset
			}
		} else {
			if len(sszSlice4) > 0 {
				return fmt.Errorf("list bytes too short to contain an offset when decoding c.Attestations")
			}
			c.Attestations = make([]*AttestationEIP8453, 0)
		}
	}

	// Field 5: VoluntaryExits
	{
		if len(sszSlice5)%112 != 0 {
			return fmt.Errorf("misaligned bytes: c.VoluntaryExits length is %d, which is not a multiple of 112: %w", len(sszSlice5), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice5) / 112
		c.VoluntaryExits = make([]*SignedVoluntaryExit, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *SignedVoluntaryExit
			tmp = new(SignedVoluntaryExit)
			tmpSlice := sszSlice5[i*112 : (1+i)*112]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("VoluntaryExits: %w", err)
			}
			c.VoluntaryExits[i] = tmp
		}
	}

	// Field 6: SyncAggregate
	c.SyncAggregate = new(SyncAggregate)
	if err = c.SyncAggregate.UnmarshalSSZ(sszSlice6); err != nil {
		return fmt.Errorf("SyncAggregate: %w", err)
	}

	// Field 7: BlsToExecutionChanges
	{
		if len(sszSlice7)%172 != 0 {
			return fmt.Errorf("misaligned bytes: c.BlsToExecutionChanges length is %d, which is not a multiple of 172: %w", len(sszSlice7), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice7) / 172
		c.BlsToExecutionChanges = make([]*SignedBLSToExecutionChange, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *SignedBLSToExecutionChange
			tmp = new(SignedBLSToExecutionChange)
			tmpSlice := sszSlice7[i*172 : (1+i)*172]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("BlsToExecutionChanges: %w", err)
			}
			c.BlsToExecutionChanges[i] = tmp
		}
	}

	// Field 8: SignedExecutionPayloadBid
	c.SignedExecutionPayloadBid = new(SignedExecutionPayloadBidHeze)
	if err = c.SignedExecutionPayloadBid.UnmarshalSSZ(sszSlice8); err != nil {
		return fmt.Errorf("SignedExecutionPayloadBid: %w", err)
	}

	// Field 9: PayloadAttestations
	{
		if len(sszSlice9)%202 != 0 {
			return fmt.Errorf("misaligned bytes: c.PayloadAttestations length is %d, which is not a multiple of 202: %w", len(sszSlice9), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice9) / 202
		c.PayloadAttestations = make([]*PayloadAttestation, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *PayloadAttestation
			tmp = new(PayloadAttestation)
			tmpSlice := sszSlice9[i*202 : (1+i)*202]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("PayloadAttestations: %w", err)
			}
			c.PayloadAttestations[i] = tmp
		}
	}

	// Field 10: ParentExecutionRequests
	c.ParentExecutionRequests = new(v1.ExecutionRequestsGloas)
	if err = c.ParentExecutionRequests.UnmarshalSSZ(sszSlice10); err != nil {
		return fmt.Errorf("ParentExecutionRequests: %w", err)
	}

	// Field 11: HeadVotes
	{
		if len(sszSlice11)%264 != 0 {
			return fmt.Errorf("misaligned bytes: c.HeadVotes length is %d, which is not a multiple of 264: %w", len(sszSlice11), ssz.ErrIncorrectListSize)
		}
		numElem := len(sszSlice11) / 264
		c.HeadVotes = make([]*HeadVote, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *HeadVote
			tmp = new(HeadVote)
			tmpSlice := sszSlice11[i*264 : (1+i)*264]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return fmt.Errorf("HeadVotes: %w", err)
			}
			c.HeadVotes[i] = tmp
		}
	}

	// Field 12: HeadVoteSupportBits
	if err = ssz.ValidateProgressiveBitlist(sszSlice12); err != nil {
		return fmt.Errorf("HeadVoteSupportBits: %w", err)
	}
	c.HeadVoteSupportBits = append([]byte{}, go_bitfield.Bitlist(sszSlice12)...)

	// Field 13: LegacyAttesterSlashings
	{
		// empty lists are zero length, so make sure there is room for an offset
		// before attempting to unmarshal it
		if len(sszSlice13) > 3 {
			startOffset := ssz.ReadOffset(sszSlice13[0:4])
			if startOffset == 0 {
				return fmt.Errorf("encountered invalid offset of 0 when decoding c.LegacyAttesterSlashings")
			}
			if startOffset%4 != 0 {
				return fmt.Errorf("misaligned list bytes: when decoding c.LegacyAttesterSlashings, end-of-list offset is %d, which is not a multiple of 4 (offset size)", startOffset)
			}
			listLen := startOffset / 4
			totalVarBytes := uint64(len(sszSlice13))
			if totalVarBytes < startOffset {
				return fmt.Errorf("list bytes too short to contain an offset when decoding c.LegacyAttesterSlashings")
			}
			c.LegacyAttesterSlashings = make([]*AttesterSlashingGloas, listLen)
			var tmpSlice []byte
			for i := uint64(0); i < listLen; i++ {
				var tmp *AttesterSlashingGloas
				tmp = new(AttesterSlashingGloas)
				endOffset := totalVarBytes
				if i+1 != listLen {
					endOffset = ssz.ReadOffset(sszSlice13[(i+1)*4 : (i+2)*4])
					if totalVarBytes < endOffset {
						return fmt.Errorf("offset %d points past the end of buffer when decoding c.LegacyAttesterSlashings", endOffset)
					}
				}
				if endOffset < startOffset {
					return fmt.Errorf("offset %d is not greater than start offset %d when decoding c.LegacyAttesterSlashings", endOffset, startOffset)
				}
				tmpSlice = sszSlice13[startOffset:endOffset]
				if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
					return fmt.Errorf("LegacyAttesterSlashings: %w", err)
				}
				c.LegacyAttesterSlashings[i] = tmp
				startOffset = endOffset
			}
		} else {
			if len(sszSlice13) > 0 {
				return fmt.Errorf("list bytes too short to contain an offset when decoding c.LegacyAttesterSlashings")
			}
			c.LegacyAttesterSlashings = make([]*AttesterSlashingGloas, 0)
		}
	}
	return err
}

func (c *BeaconBlockBodyEIP8453) HashTreeRoot() ([32]byte, error) {
	return c.ProgressiveHashTreeRoot()
}

func (c *BeaconBlockBodyEIP8453) HashTreeRootWith(hh *ssz.Hasher) error {
	return c.ProgressiveHashTreeRootWith(hh)
}

var activeFieldsBeaconBlockBodyEIP8453 = []byte{0b10111101, 0b11111111}

func (c *BeaconBlockBodyEIP8453) ProgressiveHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.ProgressiveHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BeaconBlockBodyEIP8453) ProgressiveHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Position 0: RandaoReveal
	if len(c.RandaoReveal) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.RandaoReveal)
	// Position 1: inactive (zero chunk)
	hh.PutBool(false)
	// Position 2: Graffiti
	if len(c.Graffiti) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Graffiti)
	// Position 3: ProposerSlashings
	{
		subIndx := hh.Index()
		for _, o := range c.ProposerSlashings {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("ProposerSlashings: %w", err)
			}
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.ProposerSlashings)))
	}
	// Position 4: AttesterSlashings
	{
		subIndx := hh.Index()
		for _, o := range c.AttesterSlashings {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("AttesterSlashings: %w", err)
			}
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.AttesterSlashings)))
	}
	// Position 5: Attestations
	{
		subIndx := hh.Index()
		for _, o := range c.Attestations {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("Attestations: %w", err)
			}
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.Attestations)))
	}
	// Position 6: inactive (zero chunk)
	hh.PutBool(false)
	// Position 7: VoluntaryExits
	{
		subIndx := hh.Index()
		for _, o := range c.VoluntaryExits {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("VoluntaryExits: %w", err)
			}
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.VoluntaryExits)))
	}
	// Position 8: SyncAggregate
	if err := c.SyncAggregate.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("SyncAggregate: %w", err)
	}
	// Position 9: BlsToExecutionChanges
	{
		subIndx := hh.Index()
		for _, o := range c.BlsToExecutionChanges {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("BlsToExecutionChanges: %w", err)
			}
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.BlsToExecutionChanges)))
	}
	// Position 10: SignedExecutionPayloadBid
	if err := c.SignedExecutionPayloadBid.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("SignedExecutionPayloadBid: %w", err)
	}
	// Position 11: PayloadAttestations
	{
		subIndx := hh.Index()
		for _, o := range c.PayloadAttestations {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("PayloadAttestations: %w", err)
			}
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.PayloadAttestations)))
	}
	// Position 12: ParentExecutionRequests
	if err := c.ParentExecutionRequests.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("ParentExecutionRequests: %w", err)
	}
	// Position 13: HeadVotes
	{
		subIndx := hh.Index()
		for _, o := range c.HeadVotes {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("HeadVotes: %w", err)
			}
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.HeadVotes)))
	}
	// Position 14: HeadVoteSupportBits
	if len(c.HeadVoteSupportBits) == 0 {
		return ssz.ErrEmptyBitlist
	}
	hh.PutProgressiveBitlist(c.HeadVoteSupportBits)
	// Position 15: LegacyAttesterSlashings
	{
		subIndx := hh.Index()
		for _, o := range c.LegacyAttesterSlashings {
			if err := o.HashTreeRootWith(hh); err != nil {
				return fmt.Errorf("LegacyAttesterSlashings: %w", err)
			}
		}
		hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len(c.LegacyAttesterSlashings)))
	}
	hh.MerkleizeProgressiveWithActiveFields(indx, activeFieldsBeaconBlockBodyEIP8453)
	return nil
}

func (c *BeaconBlockEIP8453) SizeSSZ() int {
	size := 84
	if c.Body == nil {
		c.Body = new(BeaconBlockBodyEIP8453)
	}
	size += c.Body.SizeSSZ()
	return size
}

func (c *BeaconBlockEIP8453) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *BeaconBlockEIP8453) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 84

	// Field 0: Slot
	if dst, err = c.Slot.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Slot: %w", err)
	}

	// Field 1: ProposerIndex
	if dst, err = c.ProposerIndex.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("ProposerIndex: %w", err)
	}

	// Field 2: ParentRoot
	if len(c.ParentRoot) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.ParentRoot...)

	// Field 3: StateRoot
	if len(c.StateRoot) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.StateRoot...)

	// Field 4: Body
	if c.Body == nil {
		c.Body = new(BeaconBlockBodyEIP8453)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Body.SizeSSZ()

	// Field 4: Body
	if dst, err = c.Body.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Body: %w", err)
	}
	return dst, err
}

func (c *BeaconBlockEIP8453) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 84 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:8]   // c.Slot
	sszSlice1 := buf[8:16]  // c.ProposerIndex
	sszSlice2 := buf[16:48] // c.ParentRoot
	sszSlice3 := buf[48:80] // c.StateRoot

	sszVarOffset4 := ssz.ReadOffset(buf[80:84]) // c.Body
	if sszVarOffset4 != 84 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset4 > size {
		return ssz.ErrOffset
	}
	sszSlice4 := buf[sszVarOffset4:] // c.Body

	// Field 0: Slot
	if err = c.Slot.UnmarshalSSZ(sszSlice0); err != nil {
		return fmt.Errorf("Slot: %w", err)
	}

	// Field 1: ProposerIndex
	if err = c.ProposerIndex.UnmarshalSSZ(sszSlice1); err != nil {
		return fmt.Errorf("ProposerIndex: %w", err)
	}

	// Field 2: ParentRoot
	c.ParentRoot = make([]byte, 0, 32)
	c.ParentRoot = append(c.ParentRoot, sszSlice2...)

	// Field 3: StateRoot
	c.StateRoot = make([]byte, 0, 32)
	c.StateRoot = append(c.StateRoot, sszSlice3...)

	// Field 4: Body
	c.Body = new(BeaconBlockBodyEIP8453)
	if err = c.Body.UnmarshalSSZ(sszSlice4); err != nil {
		return fmt.Errorf("Body: %w", err)
	}
	return err
}

func (c *BeaconBlockEIP8453) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BeaconBlockEIP8453) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Slot
	if err := c.Slot.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Slot: %w", err)
	}
	// Field 1: ProposerIndex
	if err := c.ProposerIndex.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("ProposerIndex: %w", err)
	}
	// Field 2: ParentRoot
	if len(c.ParentRoot) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.ParentRoot)
	// Field 3: StateRoot
	if len(c.StateRoot) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.StateRoot)
	// Field 4: Body
	if err := c.Body.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Body: %w", err)
	}
	hh.Merkleize(indx)
	return nil
}

func (c *SignedBeaconBlockEIP8453) SizeSSZ() int {
	size := 100
	if c.Block == nil {
		c.Block = new(BeaconBlockEIP8453)
	}
	size += c.Block.SizeSSZ()
	return size
}

func (c *SignedBeaconBlockEIP8453) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *SignedBeaconBlockEIP8453) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 100

	// Field 0: Block
	if c.Block == nil {
		c.Block = new(BeaconBlockEIP8453)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Block.SizeSSZ()

	// Field 1: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	// Field 0: Block
	if dst, err = c.Block.MarshalSSZTo(dst); err != nil {
		return nil, fmt.Errorf("Block: %w", err)
	}
	return dst, err
}

func (c *SignedBeaconBlockEIP8453) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 100 {
		return ssz.ErrSize
	}

	sszSlice1 := buf[4:100] // c.Signature

	sszVarOffset0 := ssz.ReadOffset(buf[0:4]) // c.Block
	if sszVarOffset0 != 100 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset0 > size {
		return ssz.ErrOffset
	}
	sszSlice0 := buf[sszVarOffset0:] // c.Block

	// Field 0: Block
	c.Block = new(BeaconBlockEIP8453)
	if err = c.Block.UnmarshalSSZ(sszSlice0); err != nil {
		return fmt.Errorf("Block: %w", err)
	}

	// Field 1: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, sszSlice1...)
	return err
}

func (c *SignedBeaconBlockEIP8453) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *SignedBeaconBlockEIP8453) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Block
	if err := c.Block.HashTreeRootWith(hh); err != nil {
		return fmt.Errorf("Block: %w", err)
	}
	// Field 1: Signature
	if len(c.Signature) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Signature)
	hh.Merkleize(indx)
	return nil
}
