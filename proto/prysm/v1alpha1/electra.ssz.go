package eth

import (
	"fmt"
	ssz "github.com/prysmaticlabs/fastssz"
	go_bitfield "github.com/prysmaticlabs/go-bitfield"
	v1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
)

func (c *AggregateAttestationAndProofElectra) SizeSSZ() int {
	size := 108
	if c.Aggregate == nil {
		c.Aggregate = new(AttestationElectra)
	}
	size += c.Aggregate.SizeSSZ()
	return size
}

func (c *AggregateAttestationAndProofElectra) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *AggregateAttestationAndProofElectra) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 108

	// Field 0: AggregatorIndex
	if dst, err = c.AggregatorIndex.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 1: Aggregate
	if c.Aggregate == nil {
		c.Aggregate = new(AttestationElectra)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Aggregate.SizeSSZ()

	// Field 2: SelectionProof
	if len(c.SelectionProof) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.SelectionProof...)

	// Field 1: Aggregate
	if dst, err = c.Aggregate.MarshalSSZTo(dst); err != nil {
		return nil, err
	}
	return dst, err
}

func (c *AggregateAttestationAndProofElectra) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 108 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:8]    // c.AggregatorIndex
	sszSlice2 := buf[12:108] // c.SelectionProof

	sszVarOffset1 := ssz.ReadOffset(buf[8:12]) // c.Aggregate
	if sszVarOffset1 < 108 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset1 > size {
		return ssz.ErrOffset
	}
	sszSlice1 := buf[sszVarOffset1:] // c.Aggregate

	// Field 0: AggregatorIndex
	if err = c.AggregatorIndex.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: Aggregate
	c.Aggregate = new(AttestationElectra)
	if err = c.Aggregate.UnmarshalSSZ(sszSlice1); err != nil {
		return err
	}

	// Field 2: SelectionProof
	c.SelectionProof = make([]byte, 0, 96)
	c.SelectionProof = append(c.SelectionProof, sszSlice2...)
	return err
}

func (c *AggregateAttestationAndProofElectra) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *AggregateAttestationAndProofElectra) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: AggregatorIndex
	if hash, err := c.AggregatorIndex.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 1: Aggregate
	if err := c.Aggregate.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 2: SelectionProof
	if len(c.SelectionProof) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.SelectionProof)
	hh.Merkleize(indx)
	return nil
}

func (c *AttestationElectra) SizeSSZ() int {
	size := 236
	size += len(c.AggregationBits)
	return size
}

func (c *AttestationElectra) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *AttestationElectra) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 236

	// Field 0: AggregationBits
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.AggregationBits)

	// Field 1: Data
	if c.Data == nil {
		c.Data = new(AttestationData)
	}
	if dst, err = c.Data.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 2: CommitteeBits
	if len([]byte(c.CommitteeBits)) != 8 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, []byte(c.CommitteeBits)...)

	// Field 3: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	// Field 0: AggregationBits
	if len(c.AggregationBits) > 131072 {
		return nil, ssz.ErrListTooBig
	}
	dst = append(dst, c.AggregationBits...)
	return dst, err
}

func (c *AttestationElectra) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 236 {
		return ssz.ErrSize
	}

	sszSlice1 := buf[4:132]   // c.Data
	sszSlice2 := buf[132:140] // c.CommitteeBits
	sszSlice3 := buf[140:236] // c.Signature

	sszVarOffset0 := ssz.ReadOffset(buf[0:4]) // c.AggregationBits
	if sszVarOffset0 < 236 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset0 > size {
		return ssz.ErrOffset
	}
	sszSlice0 := buf[sszVarOffset0:] // c.AggregationBits

	// Field 0: AggregationBits
	if err = ssz.ValidateBitlist(sszSlice0, 131072); err != nil {
		return err
	}
	c.AggregationBits = append([]byte{}, go_bitfield.Bitlist(sszSlice0)...)

	// Field 1: Data
	c.Data = new(AttestationData)
	if err = c.Data.UnmarshalSSZ(sszSlice1); err != nil {
		return err
	}

	// Field 2: CommitteeBits
	c.CommitteeBits = make([]byte, 0, 8)
	c.CommitteeBits = append(c.CommitteeBits, go_bitfield.Bitvector64(sszSlice2)...)

	// Field 3: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, sszSlice3...)
	return err
}

func (c *AttestationElectra) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *AttestationElectra) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: AggregationBits
	if len(c.AggregationBits) == 0 {
		return ssz.ErrEmptyBitlist
	}
	hh.PutBitlist(c.AggregationBits, 131072)
	// Field 1: Data
	if err := c.Data.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 2: CommitteeBits
	if len([]byte(c.CommitteeBits)) != 8 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes([]byte(c.CommitteeBits))
	// Field 3: Signature
	if len(c.Signature) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Signature)
	hh.Merkleize(indx)
	return nil
}

func (c *AttesterSlashingElectra) SizeSSZ() int {
	size := 8
	if c.Attestation_1 == nil {
		c.Attestation_1 = new(IndexedAttestationElectra)
	}
	size += c.Attestation_1.SizeSSZ()
	if c.Attestation_2 == nil {
		c.Attestation_2 = new(IndexedAttestationElectra)
	}
	size += c.Attestation_2.SizeSSZ()
	return size
}

func (c *AttesterSlashingElectra) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *AttesterSlashingElectra) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 8

	// Field 0: Attestation_1
	if c.Attestation_1 == nil {
		c.Attestation_1 = new(IndexedAttestationElectra)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Attestation_1.SizeSSZ()

	// Field 1: Attestation_2
	if c.Attestation_2 == nil {
		c.Attestation_2 = new(IndexedAttestationElectra)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Attestation_2.SizeSSZ()

	// Field 0: Attestation_1
	if dst, err = c.Attestation_1.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 1: Attestation_2
	if dst, err = c.Attestation_2.MarshalSSZTo(dst); err != nil {
		return nil, err
	}
	return dst, err
}

func (c *AttesterSlashingElectra) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 8 {
		return ssz.ErrSize
	}

	sszVarOffset0 := ssz.ReadOffset(buf[0:4]) // c.Attestation_1
	if sszVarOffset0 < 8 {
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
	c.Attestation_1 = new(IndexedAttestationElectra)
	if err = c.Attestation_1.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: Attestation_2
	c.Attestation_2 = new(IndexedAttestationElectra)
	if err = c.Attestation_2.UnmarshalSSZ(sszSlice1); err != nil {
		return err
	}
	return err
}

func (c *AttesterSlashingElectra) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *AttesterSlashingElectra) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Attestation_1
	if err := c.Attestation_1.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 1: Attestation_2
	if err := c.Attestation_2.HashTreeRootWith(hh); err != nil {
		return err
	}
	hh.Merkleize(indx)
	return nil
}

func (c *BeaconBlockElectra) SizeSSZ() int {
	size := 84
	if c.Body == nil {
		c.Body = new(BeaconBlockBodyElectra)
	}
	size += c.Body.SizeSSZ()
	return size
}

func (c *BeaconBlockElectra) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *BeaconBlockElectra) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 84

	// Field 0: Slot
	if dst, err = c.Slot.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 1: ProposerIndex
	if dst, err = c.ProposerIndex.MarshalSSZTo(dst); err != nil {
		return nil, err
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
		c.Body = new(BeaconBlockBodyElectra)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Body.SizeSSZ()

	// Field 4: Body
	if dst, err = c.Body.MarshalSSZTo(dst); err != nil {
		return nil, err
	}
	return dst, err
}

func (c *BeaconBlockElectra) UnmarshalSSZ(buf []byte) error {
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
	if sszVarOffset4 < 84 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset4 > size {
		return ssz.ErrOffset
	}
	sszSlice4 := buf[sszVarOffset4:] // c.Body

	// Field 0: Slot
	if err = c.Slot.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: ProposerIndex
	if err = c.ProposerIndex.UnmarshalSSZ(sszSlice1); err != nil {
		return err
	}

	// Field 2: ParentRoot
	c.ParentRoot = make([]byte, 0, 32)
	c.ParentRoot = append(c.ParentRoot, sszSlice2...)

	// Field 3: StateRoot
	c.StateRoot = make([]byte, 0, 32)
	c.StateRoot = append(c.StateRoot, sszSlice3...)

	// Field 4: Body
	c.Body = new(BeaconBlockBodyElectra)
	if err = c.Body.UnmarshalSSZ(sszSlice4); err != nil {
		return err
	}
	return err
}

func (c *BeaconBlockElectra) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BeaconBlockElectra) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Slot
	if hash, err := c.Slot.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 1: ProposerIndex
	if hash, err := c.ProposerIndex.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
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
		return err
	}
	hh.Merkleize(indx)
	return nil
}

func (c *BeaconBlockBodyElectra) SizeSSZ() int {
	size := 396
	size += len(c.ProposerSlashings) * 416
	size += func() int {
		s := 0
		for _, o := range c.AttesterSlashings {
			s += 4
			s += o.SizeSSZ()
		}
		return s
	}()
	size += func() int {
		s := 0
		for _, o := range c.Attestations {
			s += 4
			s += o.SizeSSZ()
		}
		return s
	}()
	size += len(c.Deposits) * 1240
	size += len(c.VoluntaryExits) * 112
	if c.ExecutionPayload == nil {
		c.ExecutionPayload = new(v1.ExecutionPayloadElectra)
	}
	size += c.ExecutionPayload.SizeSSZ()
	size += len(c.BlsToExecutionChanges) * 172
	size += len(c.BlobKzgCommitments) * 48
	size += len(c.Consolidations) * 120
	return size
}

func (c *BeaconBlockBodyElectra) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *BeaconBlockBodyElectra) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 396

	// Field 0: RandaoReveal
	if len(c.RandaoReveal) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.RandaoReveal...)

	// Field 1: Eth1Data
	if c.Eth1Data == nil {
		c.Eth1Data = new(Eth1Data)
	}
	if dst, err = c.Eth1Data.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 2: Graffiti
	if len(c.Graffiti) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Graffiti...)

	// Field 3: ProposerSlashings
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.ProposerSlashings) * 416

	// Field 4: AttesterSlashings
	dst = ssz.WriteOffset(dst, offset)
	offset += func() int {
		s := 0
		for _, o := range c.AttesterSlashings {
			s += 4
			s += o.SizeSSZ()
		}
		return s
	}()

	// Field 5: Attestations
	dst = ssz.WriteOffset(dst, offset)
	offset += func() int {
		s := 0
		for _, o := range c.Attestations {
			s += 4
			s += o.SizeSSZ()
		}
		return s
	}()

	// Field 6: Deposits
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.Deposits) * 1240

	// Field 7: VoluntaryExits
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.VoluntaryExits) * 112

	// Field 8: SyncAggregate
	if c.SyncAggregate == nil {
		c.SyncAggregate = new(SyncAggregate)
	}
	if dst, err = c.SyncAggregate.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 9: ExecutionPayload
	if c.ExecutionPayload == nil {
		c.ExecutionPayload = new(v1.ExecutionPayloadElectra)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.ExecutionPayload.SizeSSZ()

	// Field 10: BlsToExecutionChanges
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.BlsToExecutionChanges) * 172

	// Field 11: BlobKzgCommitments
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.BlobKzgCommitments) * 48

	// Field 12: Consolidations
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.Consolidations) * 120

	// Field 3: ProposerSlashings
	if len(c.ProposerSlashings) > 16 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.ProposerSlashings {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 4: AttesterSlashings
	if len(c.AttesterSlashings) > 1 {
		return nil, ssz.ErrListTooBig
	}
	{
		offset = 4 * len(c.AttesterSlashings)
		for _, o := range c.AttesterSlashings {
			dst = ssz.WriteOffset(dst, offset)
			offset += o.SizeSSZ()
		}
	}
	for _, o := range c.AttesterSlashings {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 5: Attestations
	if len(c.Attestations) > 8 {
		return nil, ssz.ErrListTooBig
	}
	{
		offset = 4 * len(c.Attestations)
		for _, o := range c.Attestations {
			dst = ssz.WriteOffset(dst, offset)
			offset += o.SizeSSZ()
		}
	}
	for _, o := range c.Attestations {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 6: Deposits
	if len(c.Deposits) > 16 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.Deposits {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 7: VoluntaryExits
	if len(c.VoluntaryExits) > 16 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.VoluntaryExits {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 9: ExecutionPayload
	if dst, err = c.ExecutionPayload.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 10: BlsToExecutionChanges
	if len(c.BlsToExecutionChanges) > 16 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.BlsToExecutionChanges {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 11: BlobKzgCommitments
	if len(c.BlobKzgCommitments) > 4096 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.BlobKzgCommitments {
		if len(o) != 48 {
			return nil, ssz.ErrBytesLength
		}
		dst = append(dst, o...)
	}

	// Field 12: Consolidations
	if len(c.Consolidations) > 1 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.Consolidations {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}
	return dst, err
}

func (c *BeaconBlockBodyElectra) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 396 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:96]    // c.RandaoReveal
	sszSlice1 := buf[96:168]  // c.Eth1Data
	sszSlice2 := buf[168:200] // c.Graffiti
	sszSlice8 := buf[220:380] // c.SyncAggregate

	sszVarOffset3 := ssz.ReadOffset(buf[200:204]) // c.ProposerSlashings
	if sszVarOffset3 < 396 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset3 > size {
		return ssz.ErrOffset
	}
	sszVarOffset4 := ssz.ReadOffset(buf[204:208]) // c.AttesterSlashings
	if sszVarOffset4 > size || sszVarOffset4 < sszVarOffset3 {
		return ssz.ErrOffset
	}
	sszVarOffset5 := ssz.ReadOffset(buf[208:212]) // c.Attestations
	if sszVarOffset5 > size || sszVarOffset5 < sszVarOffset4 {
		return ssz.ErrOffset
	}
	sszVarOffset6 := ssz.ReadOffset(buf[212:216]) // c.Deposits
	if sszVarOffset6 > size || sszVarOffset6 < sszVarOffset5 {
		return ssz.ErrOffset
	}
	sszVarOffset7 := ssz.ReadOffset(buf[216:220]) // c.VoluntaryExits
	if sszVarOffset7 > size || sszVarOffset7 < sszVarOffset6 {
		return ssz.ErrOffset
	}
	sszVarOffset9 := ssz.ReadOffset(buf[380:384]) // c.ExecutionPayload
	if sszVarOffset9 > size || sszVarOffset9 < sszVarOffset7 {
		return ssz.ErrOffset
	}
	sszVarOffset10 := ssz.ReadOffset(buf[384:388]) // c.BlsToExecutionChanges
	if sszVarOffset10 > size || sszVarOffset10 < sszVarOffset9 {
		return ssz.ErrOffset
	}
	sszVarOffset11 := ssz.ReadOffset(buf[388:392]) // c.BlobKzgCommitments
	if sszVarOffset11 > size || sszVarOffset11 < sszVarOffset10 {
		return ssz.ErrOffset
	}
	sszVarOffset12 := ssz.ReadOffset(buf[392:396]) // c.Consolidations
	if sszVarOffset12 > size || sszVarOffset12 < sszVarOffset11 {
		return ssz.ErrOffset
	}
	sszSlice3 := buf[sszVarOffset3:sszVarOffset4]    // c.ProposerSlashings
	sszSlice4 := buf[sszVarOffset4:sszVarOffset5]    // c.AttesterSlashings
	sszSlice5 := buf[sszVarOffset5:sszVarOffset6]    // c.Attestations
	sszSlice6 := buf[sszVarOffset6:sszVarOffset7]    // c.Deposits
	sszSlice7 := buf[sszVarOffset7:sszVarOffset9]    // c.VoluntaryExits
	sszSlice9 := buf[sszVarOffset9:sszVarOffset10]   // c.ExecutionPayload
	sszSlice10 := buf[sszVarOffset10:sszVarOffset11] // c.BlsToExecutionChanges
	sszSlice11 := buf[sszVarOffset11:sszVarOffset12] // c.BlobKzgCommitments
	sszSlice12 := buf[sszVarOffset12:]               // c.Consolidations

	// Field 0: RandaoReveal
	c.RandaoReveal = make([]byte, 0, 96)
	c.RandaoReveal = append(c.RandaoReveal, sszSlice0...)

	// Field 1: Eth1Data
	c.Eth1Data = new(Eth1Data)
	if err = c.Eth1Data.UnmarshalSSZ(sszSlice1); err != nil {
		return err
	}

	// Field 2: Graffiti
	c.Graffiti = make([]byte, 0, 32)
	c.Graffiti = append(c.Graffiti, sszSlice2...)

	// Field 3: ProposerSlashings
	{
		if len(sszSlice3)%416 != 0 {
			return fmt.Errorf("misaligned bytes: c.ProposerSlashings length is %d, which is not a multiple of 416", len(sszSlice3))
		}
		numElem := len(sszSlice3) / 416
		if numElem > 16 {
			return fmt.Errorf("ssz-max exceeded: c.ProposerSlashings has %d elements, ssz-max is 16", numElem)
		}
		c.ProposerSlashings = make([]*ProposerSlashing, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *ProposerSlashing
			tmp = new(ProposerSlashing)
			tmpSlice := sszSlice3[i*416 : (1+i)*416]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.ProposerSlashings[i] = tmp
		}
	}

	// Field 4: AttesterSlashings
	{
		// empty lists are zero length, so make sure there is room for an offset
		// before attempting to unmarshal it
		if len(sszSlice4) > 3 {
			firstOffset := ssz.ReadOffset(sszSlice4[0:4])
			if firstOffset%4 != 0 {
				return fmt.Errorf("misaligned list bytes: when decoding c.AttesterSlashings, end-of-list offset is %d, which is not a multiple of 4 (offset size)", firstOffset)
			}
			listLen := firstOffset / 4
			if listLen > 1 {
				return fmt.Errorf("ssz-max exceeded: c.AttesterSlashings has %d elements, ssz-max is 1", listLen)
			}
			listOffsets := make([]uint64, listLen)
			for i := 0; uint64(i) < listLen; i++ {
				listOffsets[i] = ssz.ReadOffset(sszSlice4[i*4 : (i+1)*4])
			}
			c.AttesterSlashings = make([]*AttesterSlashingElectra, len(listOffsets))
			for i := 0; i < len(listOffsets); i++ {
				var tmp *AttesterSlashingElectra
				tmp = new(AttesterSlashingElectra)
				var tmpSlice []byte
				if i+1 == len(listOffsets) {
					tmpSlice = sszSlice4[listOffsets[i]:]
				} else {
					tmpSlice = sszSlice4[listOffsets[i]:listOffsets[i+1]]
				}
				if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
					return err
				}
				c.AttesterSlashings[i] = tmp
			}
		}
	}

	// Field 5: Attestations
	{
		// empty lists are zero length, so make sure there is room for an offset
		// before attempting to unmarshal it
		if len(sszSlice5) > 3 {
			firstOffset := ssz.ReadOffset(sszSlice5[0:4])
			if firstOffset%4 != 0 {
				return fmt.Errorf("misaligned list bytes: when decoding c.Attestations, end-of-list offset is %d, which is not a multiple of 4 (offset size)", firstOffset)
			}
			listLen := firstOffset / 4
			if listLen > 8 {
				return fmt.Errorf("ssz-max exceeded: c.Attestations has %d elements, ssz-max is 8", listLen)
			}
			listOffsets := make([]uint64, listLen)
			for i := 0; uint64(i) < listLen; i++ {
				listOffsets[i] = ssz.ReadOffset(sszSlice5[i*4 : (i+1)*4])
			}
			c.Attestations = make([]*AttestationElectra, len(listOffsets))
			for i := 0; i < len(listOffsets); i++ {
				var tmp *AttestationElectra
				tmp = new(AttestationElectra)
				var tmpSlice []byte
				if i+1 == len(listOffsets) {
					tmpSlice = sszSlice5[listOffsets[i]:]
				} else {
					tmpSlice = sszSlice5[listOffsets[i]:listOffsets[i+1]]
				}
				if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
					return err
				}
				c.Attestations[i] = tmp
			}
		}
	}

	// Field 6: Deposits
	{
		if len(sszSlice6)%1240 != 0 {
			return fmt.Errorf("misaligned bytes: c.Deposits length is %d, which is not a multiple of 1240", len(sszSlice6))
		}
		numElem := len(sszSlice6) / 1240
		if numElem > 16 {
			return fmt.Errorf("ssz-max exceeded: c.Deposits has %d elements, ssz-max is 16", numElem)
		}
		c.Deposits = make([]*Deposit, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *Deposit
			tmp = new(Deposit)
			tmpSlice := sszSlice6[i*1240 : (1+i)*1240]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.Deposits[i] = tmp
		}
	}

	// Field 7: VoluntaryExits
	{
		if len(sszSlice7)%112 != 0 {
			return fmt.Errorf("misaligned bytes: c.VoluntaryExits length is %d, which is not a multiple of 112", len(sszSlice7))
		}
		numElem := len(sszSlice7) / 112
		if numElem > 16 {
			return fmt.Errorf("ssz-max exceeded: c.VoluntaryExits has %d elements, ssz-max is 16", numElem)
		}
		c.VoluntaryExits = make([]*SignedVoluntaryExit, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *SignedVoluntaryExit
			tmp = new(SignedVoluntaryExit)
			tmpSlice := sszSlice7[i*112 : (1+i)*112]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.VoluntaryExits[i] = tmp
		}
	}

	// Field 8: SyncAggregate
	c.SyncAggregate = new(SyncAggregate)
	if err = c.SyncAggregate.UnmarshalSSZ(sszSlice8); err != nil {
		return err
	}

	// Field 9: ExecutionPayload
	c.ExecutionPayload = new(v1.ExecutionPayloadElectra)
	if err = c.ExecutionPayload.UnmarshalSSZ(sszSlice9); err != nil {
		return err
	}

	// Field 10: BlsToExecutionChanges
	{
		if len(sszSlice10)%172 != 0 {
			return fmt.Errorf("misaligned bytes: c.BlsToExecutionChanges length is %d, which is not a multiple of 172", len(sszSlice10))
		}
		numElem := len(sszSlice10) / 172
		if numElem > 16 {
			return fmt.Errorf("ssz-max exceeded: c.BlsToExecutionChanges has %d elements, ssz-max is 16", numElem)
		}
		c.BlsToExecutionChanges = make([]*SignedBLSToExecutionChange, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *SignedBLSToExecutionChange
			tmp = new(SignedBLSToExecutionChange)
			tmpSlice := sszSlice10[i*172 : (1+i)*172]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.BlsToExecutionChanges[i] = tmp
		}
	}

	// Field 11: BlobKzgCommitments
	{
		if len(sszSlice11)%48 != 0 {
			return fmt.Errorf("misaligned bytes: c.BlobKzgCommitments length is %d, which is not a multiple of 48", len(sszSlice11))
		}
		numElem := len(sszSlice11) / 48
		if numElem > 4096 {
			return fmt.Errorf("ssz-max exceeded: c.BlobKzgCommitments has %d elements, ssz-max is 4096", numElem)
		}
		c.BlobKzgCommitments = make([][]byte, numElem)
		for i := 0; i < numElem; i++ {
			var tmp []byte

			tmpSlice := sszSlice11[i*48 : (1+i)*48]
			tmp = make([]byte, 0, 48)
			tmp = append(tmp, tmpSlice...)
			c.BlobKzgCommitments[i] = tmp
		}
	}

	// Field 12: Consolidations
	{
		if len(sszSlice12)%120 != 0 {
			return fmt.Errorf("misaligned bytes: c.Consolidations length is %d, which is not a multiple of 120", len(sszSlice12))
		}
		numElem := len(sszSlice12) / 120
		if numElem > 1 {
			return fmt.Errorf("ssz-max exceeded: c.Consolidations has %d elements, ssz-max is 1", numElem)
		}
		c.Consolidations = make([]*SignedConsolidation, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *SignedConsolidation
			tmp = new(SignedConsolidation)
			tmpSlice := sszSlice12[i*120 : (1+i)*120]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.Consolidations[i] = tmp
		}
	}
	return err
}

func (c *BeaconBlockBodyElectra) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BeaconBlockBodyElectra) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: RandaoReveal
	if len(c.RandaoReveal) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.RandaoReveal)
	// Field 1: Eth1Data
	if err := c.Eth1Data.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 2: Graffiti
	if len(c.Graffiti) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Graffiti)
	// Field 3: ProposerSlashings
	{
		if len(c.ProposerSlashings) > 16 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.ProposerSlashings {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.ProposerSlashings)), 16)
	}
	// Field 4: AttesterSlashings
	{
		if len(c.AttesterSlashings) > 1 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.AttesterSlashings {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.AttesterSlashings)), 1)
	}
	// Field 5: Attestations
	{
		if len(c.Attestations) > 8 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.Attestations {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.Attestations)), 8)
	}
	// Field 6: Deposits
	{
		if len(c.Deposits) > 16 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.Deposits {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.Deposits)), 16)
	}
	// Field 7: VoluntaryExits
	{
		if len(c.VoluntaryExits) > 16 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.VoluntaryExits {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.VoluntaryExits)), 16)
	}
	// Field 8: SyncAggregate
	if err := c.SyncAggregate.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 9: ExecutionPayload
	if hash, err := c.ExecutionPayload.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 10: BlsToExecutionChanges
	{
		if len(c.BlsToExecutionChanges) > 16 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.BlsToExecutionChanges {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.BlsToExecutionChanges)), 16)
	}
	// Field 11: BlobKzgCommitments
	{
		if len(c.BlobKzgCommitments) > 4096 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.BlobKzgCommitments {
			if len(o) != 48 {
				return ssz.ErrBytesLength
			}
			hh.PutBytes(o)
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.BlobKzgCommitments)), 4096)
	}
	// Field 12: Consolidations
	{
		if len(c.Consolidations) > 1 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.Consolidations {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.Consolidations)), 1)
	}
	hh.Merkleize(indx)
	return nil
}

func (c *BeaconStateElectra) SizeSSZ() int {
	size := 2736713
	size += len(c.HistoricalRoots) * 32
	size += len(c.Eth1DataVotes) * 72
	size += len(c.Validators) * 121
	size += len(c.Balances) * 8
	size += len(c.PreviousEpochParticipation)
	size += len(c.CurrentEpochParticipation)
	size += len(c.InactivityScores) * 8
	if c.LatestExecutionPayloadHeader == nil {
		c.LatestExecutionPayloadHeader = new(v1.ExecutionPayloadHeaderElectra)
	}
	size += c.LatestExecutionPayloadHeader.SizeSSZ()
	size += len(c.HistoricalSummaries) * 64
	size += len(c.PendingBalanceDeposits) * 16
	size += len(c.PendingPartialWithdrawals) * 24
	size += len(c.PendingConsolidations) * 16
	return size
}

func (c *BeaconStateElectra) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *BeaconStateElectra) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 2736713

	// Field 0: GenesisTime
	dst = ssz.MarshalUint64(dst, c.GenesisTime)

	// Field 1: GenesisValidatorsRoot
	if len(c.GenesisValidatorsRoot) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.GenesisValidatorsRoot...)

	// Field 2: Slot
	if dst, err = c.Slot.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 3: Fork
	if c.Fork == nil {
		c.Fork = new(Fork)
	}
	if dst, err = c.Fork.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 4: LatestBlockHeader
	if c.LatestBlockHeader == nil {
		c.LatestBlockHeader = new(BeaconBlockHeader)
	}
	if dst, err = c.LatestBlockHeader.MarshalSSZTo(dst); err != nil {
		return nil, err
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

	// Field 8: Eth1Data
	if c.Eth1Data == nil {
		c.Eth1Data = new(Eth1Data)
	}
	if dst, err = c.Eth1Data.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 9: Eth1DataVotes
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.Eth1DataVotes) * 72

	// Field 10: Eth1DepositIndex
	dst = ssz.MarshalUint64(dst, c.Eth1DepositIndex)

	// Field 11: Validators
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.Validators) * 121

	// Field 12: Balances
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.Balances) * 8

	// Field 13: RandaoMixes
	if len(c.RandaoMixes) != 65536 {
		return nil, ssz.ErrBytesLength
	}
	for _, o := range c.RandaoMixes {
		if len(o) != 32 {
			return nil, ssz.ErrBytesLength
		}
		dst = append(dst, o...)
	}

	// Field 14: Slashings
	if len(c.Slashings) != 8192 {
		return nil, ssz.ErrBytesLength
	}
	for _, o := range c.Slashings {
		dst = ssz.MarshalUint64(dst, o)
	}

	// Field 15: PreviousEpochParticipation
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.PreviousEpochParticipation)

	// Field 16: CurrentEpochParticipation
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.CurrentEpochParticipation)

	// Field 17: JustificationBits
	if len([]byte(c.JustificationBits)) != 1 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, []byte(c.JustificationBits)...)

	// Field 18: PreviousJustifiedCheckpoint
	if c.PreviousJustifiedCheckpoint == nil {
		c.PreviousJustifiedCheckpoint = new(Checkpoint)
	}
	if dst, err = c.PreviousJustifiedCheckpoint.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 19: CurrentJustifiedCheckpoint
	if c.CurrentJustifiedCheckpoint == nil {
		c.CurrentJustifiedCheckpoint = new(Checkpoint)
	}
	if dst, err = c.CurrentJustifiedCheckpoint.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 20: FinalizedCheckpoint
	if c.FinalizedCheckpoint == nil {
		c.FinalizedCheckpoint = new(Checkpoint)
	}
	if dst, err = c.FinalizedCheckpoint.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 21: InactivityScores
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.InactivityScores) * 8

	// Field 22: CurrentSyncCommittee
	if c.CurrentSyncCommittee == nil {
		c.CurrentSyncCommittee = new(SyncCommittee)
	}
	if dst, err = c.CurrentSyncCommittee.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 23: NextSyncCommittee
	if c.NextSyncCommittee == nil {
		c.NextSyncCommittee = new(SyncCommittee)
	}
	if dst, err = c.NextSyncCommittee.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 24: LatestExecutionPayloadHeader
	if c.LatestExecutionPayloadHeader == nil {
		c.LatestExecutionPayloadHeader = new(v1.ExecutionPayloadHeaderElectra)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.LatestExecutionPayloadHeader.SizeSSZ()

	// Field 25: NextWithdrawalIndex
	dst = ssz.MarshalUint64(dst, c.NextWithdrawalIndex)

	// Field 26: NextWithdrawalValidatorIndex
	if dst, err = c.NextWithdrawalValidatorIndex.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 27: HistoricalSummaries
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.HistoricalSummaries) * 64

	// Field 28: DepositRequestsStartIndex
	dst = ssz.MarshalUint64(dst, c.DepositRequestsStartIndex)

	// Field 29: DepositBalanceToConsume
	if dst, err = c.DepositBalanceToConsume.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 30: ExitBalanceToConsume
	if dst, err = c.ExitBalanceToConsume.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 31: EarliestExitEpoch
	if dst, err = c.EarliestExitEpoch.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 32: ConsolidationBalanceToConsume
	if dst, err = c.ConsolidationBalanceToConsume.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 33: EarliestConsolidationEpoch
	if dst, err = c.EarliestConsolidationEpoch.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 34: PendingBalanceDeposits
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.PendingBalanceDeposits) * 16

	// Field 35: PendingPartialWithdrawals
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.PendingPartialWithdrawals) * 24

	// Field 36: PendingConsolidations
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.PendingConsolidations) * 16

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

	// Field 9: Eth1DataVotes
	if len(c.Eth1DataVotes) > 2048 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.Eth1DataVotes {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 11: Validators
	if len(c.Validators) > 1099511627776 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.Validators {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 12: Balances
	if len(c.Balances) > 1099511627776 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.Balances {
		dst = ssz.MarshalUint64(dst, o)
	}

	// Field 15: PreviousEpochParticipation
	if len(c.PreviousEpochParticipation) > 1099511627776 {
		return nil, ssz.ErrListTooBig
	}
	dst = append(dst, c.PreviousEpochParticipation...)

	// Field 16: CurrentEpochParticipation
	if len(c.CurrentEpochParticipation) > 1099511627776 {
		return nil, ssz.ErrListTooBig
	}
	dst = append(dst, c.CurrentEpochParticipation...)

	// Field 21: InactivityScores
	if len(c.InactivityScores) > 1099511627776 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.InactivityScores {
		dst = ssz.MarshalUint64(dst, o)
	}

	// Field 24: LatestExecutionPayloadHeader
	if dst, err = c.LatestExecutionPayloadHeader.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 27: HistoricalSummaries
	if len(c.HistoricalSummaries) > 16777216 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.HistoricalSummaries {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 34: PendingBalanceDeposits
	if len(c.PendingBalanceDeposits) > 134217728 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.PendingBalanceDeposits {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 35: PendingPartialWithdrawals
	if len(c.PendingPartialWithdrawals) > 134217728 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.PendingPartialWithdrawals {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 36: PendingConsolidations
	if len(c.PendingConsolidations) > 262144 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.PendingConsolidations {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}
	return dst, err
}

func (c *BeaconStateElectra) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 2736713 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:8]              // c.GenesisTime
	sszSlice1 := buf[8:40]             // c.GenesisValidatorsRoot
	sszSlice2 := buf[40:48]            // c.Slot
	sszSlice3 := buf[48:64]            // c.Fork
	sszSlice4 := buf[64:176]           // c.LatestBlockHeader
	sszSlice5 := buf[176:262320]       // c.BlockRoots
	sszSlice6 := buf[262320:524464]    // c.StateRoots
	sszSlice8 := buf[524468:524540]    // c.Eth1Data
	sszSlice10 := buf[524544:524552]   // c.Eth1DepositIndex
	sszSlice13 := buf[524560:2621712]  // c.RandaoMixes
	sszSlice14 := buf[2621712:2687248] // c.Slashings
	sszSlice17 := buf[2687256:2687257] // c.JustificationBits
	sszSlice18 := buf[2687257:2687297] // c.PreviousJustifiedCheckpoint
	sszSlice19 := buf[2687297:2687337] // c.CurrentJustifiedCheckpoint
	sszSlice20 := buf[2687337:2687377] // c.FinalizedCheckpoint
	sszSlice22 := buf[2687381:2712005] // c.CurrentSyncCommittee
	sszSlice23 := buf[2712005:2736629] // c.NextSyncCommittee
	sszSlice25 := buf[2736633:2736641] // c.NextWithdrawalIndex
	sszSlice26 := buf[2736641:2736649] // c.NextWithdrawalValidatorIndex
	sszSlice28 := buf[2736653:2736661] // c.DepositRequestsStartIndex
	sszSlice29 := buf[2736661:2736669] // c.DepositBalanceToConsume
	sszSlice30 := buf[2736669:2736677] // c.ExitBalanceToConsume
	sszSlice31 := buf[2736677:2736685] // c.EarliestExitEpoch
	sszSlice32 := buf[2736685:2736693] // c.ConsolidationBalanceToConsume
	sszSlice33 := buf[2736693:2736701] // c.EarliestConsolidationEpoch

	sszVarOffset7 := ssz.ReadOffset(buf[524464:524468]) // c.HistoricalRoots
	if sszVarOffset7 < 2736713 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset7 > size {
		return ssz.ErrOffset
	}
	sszVarOffset9 := ssz.ReadOffset(buf[524540:524544]) // c.Eth1DataVotes
	if sszVarOffset9 > size || sszVarOffset9 < sszVarOffset7 {
		return ssz.ErrOffset
	}
	sszVarOffset11 := ssz.ReadOffset(buf[524552:524556]) // c.Validators
	if sszVarOffset11 > size || sszVarOffset11 < sszVarOffset9 {
		return ssz.ErrOffset
	}
	sszVarOffset12 := ssz.ReadOffset(buf[524556:524560]) // c.Balances
	if sszVarOffset12 > size || sszVarOffset12 < sszVarOffset11 {
		return ssz.ErrOffset
	}
	sszVarOffset15 := ssz.ReadOffset(buf[2687248:2687252]) // c.PreviousEpochParticipation
	if sszVarOffset15 > size || sszVarOffset15 < sszVarOffset12 {
		return ssz.ErrOffset
	}
	sszVarOffset16 := ssz.ReadOffset(buf[2687252:2687256]) // c.CurrentEpochParticipation
	if sszVarOffset16 > size || sszVarOffset16 < sszVarOffset15 {
		return ssz.ErrOffset
	}
	sszVarOffset21 := ssz.ReadOffset(buf[2687377:2687381]) // c.InactivityScores
	if sszVarOffset21 > size || sszVarOffset21 < sszVarOffset16 {
		return ssz.ErrOffset
	}
	sszVarOffset24 := ssz.ReadOffset(buf[2736629:2736633]) // c.LatestExecutionPayloadHeader
	if sszVarOffset24 > size || sszVarOffset24 < sszVarOffset21 {
		return ssz.ErrOffset
	}
	sszVarOffset27 := ssz.ReadOffset(buf[2736649:2736653]) // c.HistoricalSummaries
	if sszVarOffset27 > size || sszVarOffset27 < sszVarOffset24 {
		return ssz.ErrOffset
	}
	sszVarOffset34 := ssz.ReadOffset(buf[2736701:2736705]) // c.PendingBalanceDeposits
	if sszVarOffset34 > size || sszVarOffset34 < sszVarOffset27 {
		return ssz.ErrOffset
	}
	sszVarOffset35 := ssz.ReadOffset(buf[2736705:2736709]) // c.PendingPartialWithdrawals
	if sszVarOffset35 > size || sszVarOffset35 < sszVarOffset34 {
		return ssz.ErrOffset
	}
	sszVarOffset36 := ssz.ReadOffset(buf[2736709:2736713]) // c.PendingConsolidations
	if sszVarOffset36 > size || sszVarOffset36 < sszVarOffset35 {
		return ssz.ErrOffset
	}
	sszSlice7 := buf[sszVarOffset7:sszVarOffset9]    // c.HistoricalRoots
	sszSlice9 := buf[sszVarOffset9:sszVarOffset11]   // c.Eth1DataVotes
	sszSlice11 := buf[sszVarOffset11:sszVarOffset12] // c.Validators
	sszSlice12 := buf[sszVarOffset12:sszVarOffset15] // c.Balances
	sszSlice15 := buf[sszVarOffset15:sszVarOffset16] // c.PreviousEpochParticipation
	sszSlice16 := buf[sszVarOffset16:sszVarOffset21] // c.CurrentEpochParticipation
	sszSlice21 := buf[sszVarOffset21:sszVarOffset24] // c.InactivityScores
	sszSlice24 := buf[sszVarOffset24:sszVarOffset27] // c.LatestExecutionPayloadHeader
	sszSlice27 := buf[sszVarOffset27:sszVarOffset34] // c.HistoricalSummaries
	sszSlice34 := buf[sszVarOffset34:sszVarOffset35] // c.PendingBalanceDeposits
	sszSlice35 := buf[sszVarOffset35:sszVarOffset36] // c.PendingPartialWithdrawals
	sszSlice36 := buf[sszVarOffset36:]               // c.PendingConsolidations

	// Field 0: GenesisTime
	c.GenesisTime = ssz.UnmarshallUint64(sszSlice0)

	// Field 1: GenesisValidatorsRoot
	c.GenesisValidatorsRoot = make([]byte, 0, 32)
	c.GenesisValidatorsRoot = append(c.GenesisValidatorsRoot, sszSlice1...)

	// Field 2: Slot
	if err = c.Slot.UnmarshalSSZ(sszSlice2); err != nil {
		return err
	}

	// Field 3: Fork
	c.Fork = new(Fork)
	if err = c.Fork.UnmarshalSSZ(sszSlice3); err != nil {
		return err
	}

	// Field 4: LatestBlockHeader
	c.LatestBlockHeader = new(BeaconBlockHeader)
	if err = c.LatestBlockHeader.UnmarshalSSZ(sszSlice4); err != nil {
		return err
	}

	// Field 5: BlockRoots
	{
		var tmp []byte
		c.BlockRoots = make([][]byte, 8192)
		for i := 0; i < 8192; i++ {
			tmpSlice := sszSlice5[i*32 : (1+i)*32]
			tmp = make([]byte, 0, 32)
			tmp = append(tmp, tmpSlice...)
			c.BlockRoots[i] = tmp
		}
	}

	// Field 6: StateRoots
	{
		var tmp []byte
		c.StateRoots = make([][]byte, 8192)
		for i := 0; i < 8192; i++ {
			tmpSlice := sszSlice6[i*32 : (1+i)*32]
			tmp = make([]byte, 0, 32)
			tmp = append(tmp, tmpSlice...)
			c.StateRoots[i] = tmp
		}
	}

	// Field 7: HistoricalRoots
	{
		if len(sszSlice7)%32 != 0 {
			return fmt.Errorf("misaligned bytes: c.HistoricalRoots length is %d, which is not a multiple of 32", len(sszSlice7))
		}
		numElem := len(sszSlice7) / 32
		if numElem > 16777216 {
			return fmt.Errorf("ssz-max exceeded: c.HistoricalRoots has %d elements, ssz-max is 16777216", numElem)
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

	// Field 8: Eth1Data
	c.Eth1Data = new(Eth1Data)
	if err = c.Eth1Data.UnmarshalSSZ(sszSlice8); err != nil {
		return err
	}

	// Field 9: Eth1DataVotes
	{
		if len(sszSlice9)%72 != 0 {
			return fmt.Errorf("misaligned bytes: c.Eth1DataVotes length is %d, which is not a multiple of 72", len(sszSlice9))
		}
		numElem := len(sszSlice9) / 72
		if numElem > 2048 {
			return fmt.Errorf("ssz-max exceeded: c.Eth1DataVotes has %d elements, ssz-max is 2048", numElem)
		}
		c.Eth1DataVotes = make([]*Eth1Data, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *Eth1Data
			tmp = new(Eth1Data)
			tmpSlice := sszSlice9[i*72 : (1+i)*72]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.Eth1DataVotes[i] = tmp
		}
	}

	// Field 10: Eth1DepositIndex
	c.Eth1DepositIndex = ssz.UnmarshallUint64(sszSlice10)

	// Field 11: Validators
	{
		if len(sszSlice11)%121 != 0 {
			return fmt.Errorf("misaligned bytes: c.Validators length is %d, which is not a multiple of 121", len(sszSlice11))
		}
		numElem := len(sszSlice11) / 121
		if numElem > 1099511627776 {
			return fmt.Errorf("ssz-max exceeded: c.Validators has %d elements, ssz-max is 1099511627776", numElem)
		}
		c.Validators = make([]*Validator, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *Validator
			tmp = new(Validator)
			tmpSlice := sszSlice11[i*121 : (1+i)*121]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.Validators[i] = tmp
		}
	}

	// Field 12: Balances
	{
		if len(sszSlice12)%8 != 0 {
			return fmt.Errorf("misaligned bytes: c.Balances length is %d, which is not a multiple of 8", len(sszSlice12))
		}
		numElem := len(sszSlice12) / 8
		if numElem > 1099511627776 {
			return fmt.Errorf("ssz-max exceeded: c.Balances has %d elements, ssz-max is 1099511627776", numElem)
		}
		c.Balances = make([]uint64, numElem)
		for i := 0; i < numElem; i++ {
			var tmp uint64

			tmpSlice := sszSlice12[i*8 : (1+i)*8]
			tmp = ssz.UnmarshallUint64(tmpSlice)
			c.Balances[i] = tmp
		}
	}

	// Field 13: RandaoMixes
	{
		var tmp []byte
		c.RandaoMixes = make([][]byte, 65536)
		for i := 0; i < 65536; i++ {
			tmpSlice := sszSlice13[i*32 : (1+i)*32]
			tmp = make([]byte, 0, 32)
			tmp = append(tmp, tmpSlice...)
			c.RandaoMixes[i] = tmp
		}
	}

	// Field 14: Slashings
	{
		var tmp uint64
		c.Slashings = make([]uint64, 8192)
		for i := 0; i < 8192; i++ {
			tmpSlice := sszSlice14[i*8 : (1+i)*8]
			tmp = ssz.UnmarshallUint64(tmpSlice)
			c.Slashings[i] = tmp
		}
	}

	// Field 15: PreviousEpochParticipation
	c.PreviousEpochParticipation = append([]byte{}, sszSlice15...)

	// Field 16: CurrentEpochParticipation
	c.CurrentEpochParticipation = append([]byte{}, sszSlice16...)

	// Field 17: JustificationBits
	c.JustificationBits = make([]byte, 0, 1)
	c.JustificationBits = append(c.JustificationBits, go_bitfield.Bitvector4(sszSlice17)...)

	// Field 18: PreviousJustifiedCheckpoint
	c.PreviousJustifiedCheckpoint = new(Checkpoint)
	if err = c.PreviousJustifiedCheckpoint.UnmarshalSSZ(sszSlice18); err != nil {
		return err
	}

	// Field 19: CurrentJustifiedCheckpoint
	c.CurrentJustifiedCheckpoint = new(Checkpoint)
	if err = c.CurrentJustifiedCheckpoint.UnmarshalSSZ(sszSlice19); err != nil {
		return err
	}

	// Field 20: FinalizedCheckpoint
	c.FinalizedCheckpoint = new(Checkpoint)
	if err = c.FinalizedCheckpoint.UnmarshalSSZ(sszSlice20); err != nil {
		return err
	}

	// Field 21: InactivityScores
	{
		if len(sszSlice21)%8 != 0 {
			return fmt.Errorf("misaligned bytes: c.InactivityScores length is %d, which is not a multiple of 8", len(sszSlice21))
		}
		numElem := len(sszSlice21) / 8
		if numElem > 1099511627776 {
			return fmt.Errorf("ssz-max exceeded: c.InactivityScores has %d elements, ssz-max is 1099511627776", numElem)
		}
		c.InactivityScores = make([]uint64, numElem)
		for i := 0; i < numElem; i++ {
			var tmp uint64

			tmpSlice := sszSlice21[i*8 : (1+i)*8]
			tmp = ssz.UnmarshallUint64(tmpSlice)
			c.InactivityScores[i] = tmp
		}
	}

	// Field 22: CurrentSyncCommittee
	c.CurrentSyncCommittee = new(SyncCommittee)
	if err = c.CurrentSyncCommittee.UnmarshalSSZ(sszSlice22); err != nil {
		return err
	}

	// Field 23: NextSyncCommittee
	c.NextSyncCommittee = new(SyncCommittee)
	if err = c.NextSyncCommittee.UnmarshalSSZ(sszSlice23); err != nil {
		return err
	}

	// Field 24: LatestExecutionPayloadHeader
	c.LatestExecutionPayloadHeader = new(v1.ExecutionPayloadHeaderElectra)
	if err = c.LatestExecutionPayloadHeader.UnmarshalSSZ(sszSlice24); err != nil {
		return err
	}

	// Field 25: NextWithdrawalIndex
	c.NextWithdrawalIndex = ssz.UnmarshallUint64(sszSlice25)

	// Field 26: NextWithdrawalValidatorIndex
	if err = c.NextWithdrawalValidatorIndex.UnmarshalSSZ(sszSlice26); err != nil {
		return err
	}

	// Field 27: HistoricalSummaries
	{
		if len(sszSlice27)%64 != 0 {
			return fmt.Errorf("misaligned bytes: c.HistoricalSummaries length is %d, which is not a multiple of 64", len(sszSlice27))
		}
		numElem := len(sszSlice27) / 64
		if numElem > 16777216 {
			return fmt.Errorf("ssz-max exceeded: c.HistoricalSummaries has %d elements, ssz-max is 16777216", numElem)
		}
		c.HistoricalSummaries = make([]*HistoricalSummary, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *HistoricalSummary
			tmp = new(HistoricalSummary)
			tmpSlice := sszSlice27[i*64 : (1+i)*64]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.HistoricalSummaries[i] = tmp
		}
	}

	// Field 28: DepositRequestsStartIndex
	c.DepositRequestsStartIndex = ssz.UnmarshallUint64(sszSlice28)

	// Field 29: DepositBalanceToConsume
	if err = c.DepositBalanceToConsume.UnmarshalSSZ(sszSlice29); err != nil {
		return err
	}

	// Field 30: ExitBalanceToConsume
	if err = c.ExitBalanceToConsume.UnmarshalSSZ(sszSlice30); err != nil {
		return err
	}

	// Field 31: EarliestExitEpoch
	if err = c.EarliestExitEpoch.UnmarshalSSZ(sszSlice31); err != nil {
		return err
	}

	// Field 32: ConsolidationBalanceToConsume
	if err = c.ConsolidationBalanceToConsume.UnmarshalSSZ(sszSlice32); err != nil {
		return err
	}

	// Field 33: EarliestConsolidationEpoch
	if err = c.EarliestConsolidationEpoch.UnmarshalSSZ(sszSlice33); err != nil {
		return err
	}

	// Field 34: PendingBalanceDeposits
	{
		if len(sszSlice34)%16 != 0 {
			return fmt.Errorf("misaligned bytes: c.PendingBalanceDeposits length is %d, which is not a multiple of 16", len(sszSlice34))
		}
		numElem := len(sszSlice34) / 16
		if numElem > 134217728 {
			return fmt.Errorf("ssz-max exceeded: c.PendingBalanceDeposits has %d elements, ssz-max is 134217728", numElem)
		}
		c.PendingBalanceDeposits = make([]*PendingBalanceDeposit, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *PendingBalanceDeposit
			tmp = new(PendingBalanceDeposit)
			tmpSlice := sszSlice34[i*16 : (1+i)*16]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.PendingBalanceDeposits[i] = tmp
		}
	}

	// Field 35: PendingPartialWithdrawals
	{
		if len(sszSlice35)%24 != 0 {
			return fmt.Errorf("misaligned bytes: c.PendingPartialWithdrawals length is %d, which is not a multiple of 24", len(sszSlice35))
		}
		numElem := len(sszSlice35) / 24
		if numElem > 134217728 {
			return fmt.Errorf("ssz-max exceeded: c.PendingPartialWithdrawals has %d elements, ssz-max is 134217728", numElem)
		}
		c.PendingPartialWithdrawals = make([]*PendingPartialWithdrawal, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *PendingPartialWithdrawal
			tmp = new(PendingPartialWithdrawal)
			tmpSlice := sszSlice35[i*24 : (1+i)*24]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.PendingPartialWithdrawals[i] = tmp
		}
	}

	// Field 36: PendingConsolidations
	{
		if len(sszSlice36)%16 != 0 {
			return fmt.Errorf("misaligned bytes: c.PendingConsolidations length is %d, which is not a multiple of 16", len(sszSlice36))
		}
		numElem := len(sszSlice36) / 16
		if numElem > 262144 {
			return fmt.Errorf("ssz-max exceeded: c.PendingConsolidations has %d elements, ssz-max is 262144", numElem)
		}
		c.PendingConsolidations = make([]*PendingConsolidation, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *PendingConsolidation
			tmp = new(PendingConsolidation)
			tmpSlice := sszSlice36[i*16 : (1+i)*16]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.PendingConsolidations[i] = tmp
		}
	}
	return err
}

func (c *BeaconStateElectra) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BeaconStateElectra) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: GenesisTime
	hh.PutUint64(c.GenesisTime)
	// Field 1: GenesisValidatorsRoot
	if len(c.GenesisValidatorsRoot) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.GenesisValidatorsRoot)
	// Field 2: Slot
	if hash, err := c.Slot.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 3: Fork
	if err := c.Fork.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 4: LatestBlockHeader
	if err := c.LatestBlockHeader.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 5: BlockRoots
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
	// Field 6: StateRoots
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
	// Field 7: HistoricalRoots
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
	// Field 8: Eth1Data
	if err := c.Eth1Data.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 9: Eth1DataVotes
	{
		if len(c.Eth1DataVotes) > 2048 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.Eth1DataVotes {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.Eth1DataVotes)), 2048)
	}
	// Field 10: Eth1DepositIndex
	hh.PutUint64(c.Eth1DepositIndex)
	// Field 11: Validators
	{
		if len(c.Validators) > 1099511627776 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.Validators {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.Validators)), 1099511627776)
	}
	// Field 12: Balances
	{
		if len(c.Balances) > 1099511627776 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.Balances {
			hh.AppendUint64(o)
		}
		hh.FillUpTo32()
		numItems := uint64(len(c.Balances))
		hh.MerkleizeWithMixin(subIndx, numItems, ssz.CalculateLimit(1099511627776, numItems, 8))
	}
	// Field 13: RandaoMixes
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
	// Field 14: Slashings
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
	// Field 15: PreviousEpochParticipation

	{
		if len(c.PreviousEpochParticipation) > 1099511627776 {
			return ssz.ErrBytesLength
		}
		subIndx := hh.Index()
		hh.PutBytes(c.PreviousEpochParticipation)
		numItems := uint64(len(c.PreviousEpochParticipation))
		hh.MerkleizeWithMixin(subIndx, numItems, (1099511627776*1+31)/32)
	}

	// Field 16: CurrentEpochParticipation

	{
		if len(c.CurrentEpochParticipation) > 1099511627776 {
			return ssz.ErrBytesLength
		}
		subIndx := hh.Index()
		hh.PutBytes(c.CurrentEpochParticipation)
		numItems := uint64(len(c.CurrentEpochParticipation))
		hh.MerkleizeWithMixin(subIndx, numItems, (1099511627776*1+31)/32)
	}

	// Field 17: JustificationBits
	if len([]byte(c.JustificationBits)) != 1 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes([]byte(c.JustificationBits))
	// Field 18: PreviousJustifiedCheckpoint
	if err := c.PreviousJustifiedCheckpoint.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 19: CurrentJustifiedCheckpoint
	if err := c.CurrentJustifiedCheckpoint.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 20: FinalizedCheckpoint
	if err := c.FinalizedCheckpoint.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 21: InactivityScores
	{
		if len(c.InactivityScores) > 1099511627776 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.InactivityScores {
			hh.AppendUint64(o)
		}
		hh.FillUpTo32()
		numItems := uint64(len(c.InactivityScores))
		hh.MerkleizeWithMixin(subIndx, numItems, ssz.CalculateLimit(1099511627776, numItems, 8))
	}
	// Field 22: CurrentSyncCommittee
	if err := c.CurrentSyncCommittee.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 23: NextSyncCommittee
	if err := c.NextSyncCommittee.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 24: LatestExecutionPayloadHeader
	if hash, err := c.LatestExecutionPayloadHeader.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 25: NextWithdrawalIndex
	hh.PutUint64(c.NextWithdrawalIndex)
	// Field 26: NextWithdrawalValidatorIndex
	if hash, err := c.NextWithdrawalValidatorIndex.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 27: HistoricalSummaries
	{
		if len(c.HistoricalSummaries) > 16777216 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.HistoricalSummaries {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.HistoricalSummaries)), 16777216)
	}
	// Field 28: DepositRequestsStartIndex
	hh.PutUint64(c.DepositRequestsStartIndex)
	// Field 29: DepositBalanceToConsume
	if hash, err := c.DepositBalanceToConsume.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 30: ExitBalanceToConsume
	if hash, err := c.ExitBalanceToConsume.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 31: EarliestExitEpoch
	if hash, err := c.EarliestExitEpoch.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 32: ConsolidationBalanceToConsume
	if hash, err := c.ConsolidationBalanceToConsume.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 33: EarliestConsolidationEpoch
	if hash, err := c.EarliestConsolidationEpoch.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 34: PendingBalanceDeposits
	{
		if len(c.PendingBalanceDeposits) > 134217728 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.PendingBalanceDeposits {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.PendingBalanceDeposits)), 134217728)
	}
	// Field 35: PendingPartialWithdrawals
	{
		if len(c.PendingPartialWithdrawals) > 134217728 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.PendingPartialWithdrawals {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.PendingPartialWithdrawals)), 134217728)
	}
	// Field 36: PendingConsolidations
	{
		if len(c.PendingConsolidations) > 262144 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.PendingConsolidations {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.PendingConsolidations)), 262144)
	}
	hh.Merkleize(indx)
	return nil
}

func (c *BlindedBeaconBlockBodyElectra) SizeSSZ() int {
	size := 396
	size += len(c.ProposerSlashings) * 416
	size += func() int {
		s := 0
		for _, o := range c.AttesterSlashings {
			s += 4
			s += o.SizeSSZ()
		}
		return s
	}()
	size += func() int {
		s := 0
		for _, o := range c.Attestations {
			s += 4
			s += o.SizeSSZ()
		}
		return s
	}()
	size += len(c.Deposits) * 1240
	size += len(c.VoluntaryExits) * 112
	if c.ExecutionPayloadHeader == nil {
		c.ExecutionPayloadHeader = new(v1.ExecutionPayloadHeaderElectra)
	}
	size += c.ExecutionPayloadHeader.SizeSSZ()
	size += len(c.BlsToExecutionChanges) * 172
	size += len(c.BlobKzgCommitments) * 48
	size += len(c.Consolidations) * 120
	return size
}

func (c *BlindedBeaconBlockBodyElectra) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *BlindedBeaconBlockBodyElectra) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 396

	// Field 0: RandaoReveal
	if len(c.RandaoReveal) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.RandaoReveal...)

	// Field 1: Eth1Data
	if c.Eth1Data == nil {
		c.Eth1Data = new(Eth1Data)
	}
	if dst, err = c.Eth1Data.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 2: Graffiti
	if len(c.Graffiti) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Graffiti...)

	// Field 3: ProposerSlashings
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.ProposerSlashings) * 416

	// Field 4: AttesterSlashings
	dst = ssz.WriteOffset(dst, offset)
	offset += func() int {
		s := 0
		for _, o := range c.AttesterSlashings {
			s += 4
			s += o.SizeSSZ()
		}
		return s
	}()

	// Field 5: Attestations
	dst = ssz.WriteOffset(dst, offset)
	offset += func() int {
		s := 0
		for _, o := range c.Attestations {
			s += 4
			s += o.SizeSSZ()
		}
		return s
	}()

	// Field 6: Deposits
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.Deposits) * 1240

	// Field 7: VoluntaryExits
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.VoluntaryExits) * 112

	// Field 8: SyncAggregate
	if c.SyncAggregate == nil {
		c.SyncAggregate = new(SyncAggregate)
	}
	if dst, err = c.SyncAggregate.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 9: ExecutionPayloadHeader
	if c.ExecutionPayloadHeader == nil {
		c.ExecutionPayloadHeader = new(v1.ExecutionPayloadHeaderElectra)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.ExecutionPayloadHeader.SizeSSZ()

	// Field 10: BlsToExecutionChanges
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.BlsToExecutionChanges) * 172

	// Field 11: BlobKzgCommitments
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.BlobKzgCommitments) * 48

	// Field 12: Consolidations
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.Consolidations) * 120

	// Field 3: ProposerSlashings
	if len(c.ProposerSlashings) > 16 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.ProposerSlashings {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 4: AttesterSlashings
	if len(c.AttesterSlashings) > 1 {
		return nil, ssz.ErrListTooBig
	}
	{
		offset = 4 * len(c.AttesterSlashings)
		for _, o := range c.AttesterSlashings {
			dst = ssz.WriteOffset(dst, offset)
			offset += o.SizeSSZ()
		}
	}
	for _, o := range c.AttesterSlashings {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 5: Attestations
	if len(c.Attestations) > 8 {
		return nil, ssz.ErrListTooBig
	}
	{
		offset = 4 * len(c.Attestations)
		for _, o := range c.Attestations {
			dst = ssz.WriteOffset(dst, offset)
			offset += o.SizeSSZ()
		}
	}
	for _, o := range c.Attestations {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 6: Deposits
	if len(c.Deposits) > 16 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.Deposits {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 7: VoluntaryExits
	if len(c.VoluntaryExits) > 16 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.VoluntaryExits {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 9: ExecutionPayloadHeader
	if dst, err = c.ExecutionPayloadHeader.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 10: BlsToExecutionChanges
	if len(c.BlsToExecutionChanges) > 16 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.BlsToExecutionChanges {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 11: BlobKzgCommitments
	if len(c.BlobKzgCommitments) > 4096 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.BlobKzgCommitments {
		if len(o) != 48 {
			return nil, ssz.ErrBytesLength
		}
		dst = append(dst, o...)
	}

	// Field 12: Consolidations
	if len(c.Consolidations) > 1 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.Consolidations {
		if dst, err = o.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}
	return dst, err
}

func (c *BlindedBeaconBlockBodyElectra) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 396 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:96]    // c.RandaoReveal
	sszSlice1 := buf[96:168]  // c.Eth1Data
	sszSlice2 := buf[168:200] // c.Graffiti
	sszSlice8 := buf[220:380] // c.SyncAggregate

	sszVarOffset3 := ssz.ReadOffset(buf[200:204]) // c.ProposerSlashings
	if sszVarOffset3 < 396 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset3 > size {
		return ssz.ErrOffset
	}
	sszVarOffset4 := ssz.ReadOffset(buf[204:208]) // c.AttesterSlashings
	if sszVarOffset4 > size || sszVarOffset4 < sszVarOffset3 {
		return ssz.ErrOffset
	}
	sszVarOffset5 := ssz.ReadOffset(buf[208:212]) // c.Attestations
	if sszVarOffset5 > size || sszVarOffset5 < sszVarOffset4 {
		return ssz.ErrOffset
	}
	sszVarOffset6 := ssz.ReadOffset(buf[212:216]) // c.Deposits
	if sszVarOffset6 > size || sszVarOffset6 < sszVarOffset5 {
		return ssz.ErrOffset
	}
	sszVarOffset7 := ssz.ReadOffset(buf[216:220]) // c.VoluntaryExits
	if sszVarOffset7 > size || sszVarOffset7 < sszVarOffset6 {
		return ssz.ErrOffset
	}
	sszVarOffset9 := ssz.ReadOffset(buf[380:384]) // c.ExecutionPayloadHeader
	if sszVarOffset9 > size || sszVarOffset9 < sszVarOffset7 {
		return ssz.ErrOffset
	}
	sszVarOffset10 := ssz.ReadOffset(buf[384:388]) // c.BlsToExecutionChanges
	if sszVarOffset10 > size || sszVarOffset10 < sszVarOffset9 {
		return ssz.ErrOffset
	}
	sszVarOffset11 := ssz.ReadOffset(buf[388:392]) // c.BlobKzgCommitments
	if sszVarOffset11 > size || sszVarOffset11 < sszVarOffset10 {
		return ssz.ErrOffset
	}
	sszVarOffset12 := ssz.ReadOffset(buf[392:396]) // c.Consolidations
	if sszVarOffset12 > size || sszVarOffset12 < sszVarOffset11 {
		return ssz.ErrOffset
	}
	sszSlice3 := buf[sszVarOffset3:sszVarOffset4]    // c.ProposerSlashings
	sszSlice4 := buf[sszVarOffset4:sszVarOffset5]    // c.AttesterSlashings
	sszSlice5 := buf[sszVarOffset5:sszVarOffset6]    // c.Attestations
	sszSlice6 := buf[sszVarOffset6:sszVarOffset7]    // c.Deposits
	sszSlice7 := buf[sszVarOffset7:sszVarOffset9]    // c.VoluntaryExits
	sszSlice9 := buf[sszVarOffset9:sszVarOffset10]   // c.ExecutionPayloadHeader
	sszSlice10 := buf[sszVarOffset10:sszVarOffset11] // c.BlsToExecutionChanges
	sszSlice11 := buf[sszVarOffset11:sszVarOffset12] // c.BlobKzgCommitments
	sszSlice12 := buf[sszVarOffset12:]               // c.Consolidations

	// Field 0: RandaoReveal
	c.RandaoReveal = make([]byte, 0, 96)
	c.RandaoReveal = append(c.RandaoReveal, sszSlice0...)

	// Field 1: Eth1Data
	c.Eth1Data = new(Eth1Data)
	if err = c.Eth1Data.UnmarshalSSZ(sszSlice1); err != nil {
		return err
	}

	// Field 2: Graffiti
	c.Graffiti = make([]byte, 0, 32)
	c.Graffiti = append(c.Graffiti, sszSlice2...)

	// Field 3: ProposerSlashings
	{
		if len(sszSlice3)%416 != 0 {
			return fmt.Errorf("misaligned bytes: c.ProposerSlashings length is %d, which is not a multiple of 416", len(sszSlice3))
		}
		numElem := len(sszSlice3) / 416
		if numElem > 16 {
			return fmt.Errorf("ssz-max exceeded: c.ProposerSlashings has %d elements, ssz-max is 16", numElem)
		}
		c.ProposerSlashings = make([]*ProposerSlashing, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *ProposerSlashing
			tmp = new(ProposerSlashing)
			tmpSlice := sszSlice3[i*416 : (1+i)*416]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.ProposerSlashings[i] = tmp
		}
	}

	// Field 4: AttesterSlashings
	{
		// empty lists are zero length, so make sure there is room for an offset
		// before attempting to unmarshal it
		if len(sszSlice4) > 3 {
			firstOffset := ssz.ReadOffset(sszSlice4[0:4])
			if firstOffset%4 != 0 {
				return fmt.Errorf("misaligned list bytes: when decoding c.AttesterSlashings, end-of-list offset is %d, which is not a multiple of 4 (offset size)", firstOffset)
			}
			listLen := firstOffset / 4
			if listLen > 1 {
				return fmt.Errorf("ssz-max exceeded: c.AttesterSlashings has %d elements, ssz-max is 1", listLen)
			}
			listOffsets := make([]uint64, listLen)
			for i := 0; uint64(i) < listLen; i++ {
				listOffsets[i] = ssz.ReadOffset(sszSlice4[i*4 : (i+1)*4])
			}
			c.AttesterSlashings = make([]*AttesterSlashingElectra, len(listOffsets))
			for i := 0; i < len(listOffsets); i++ {
				var tmp *AttesterSlashingElectra
				tmp = new(AttesterSlashingElectra)
				var tmpSlice []byte
				if i+1 == len(listOffsets) {
					tmpSlice = sszSlice4[listOffsets[i]:]
				} else {
					tmpSlice = sszSlice4[listOffsets[i]:listOffsets[i+1]]
				}
				if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
					return err
				}
				c.AttesterSlashings[i] = tmp
			}
		}
	}

	// Field 5: Attestations
	{
		// empty lists are zero length, so make sure there is room for an offset
		// before attempting to unmarshal it
		if len(sszSlice5) > 3 {
			firstOffset := ssz.ReadOffset(sszSlice5[0:4])
			if firstOffset%4 != 0 {
				return fmt.Errorf("misaligned list bytes: when decoding c.Attestations, end-of-list offset is %d, which is not a multiple of 4 (offset size)", firstOffset)
			}
			listLen := firstOffset / 4
			if listLen > 8 {
				return fmt.Errorf("ssz-max exceeded: c.Attestations has %d elements, ssz-max is 8", listLen)
			}
			listOffsets := make([]uint64, listLen)
			for i := 0; uint64(i) < listLen; i++ {
				listOffsets[i] = ssz.ReadOffset(sszSlice5[i*4 : (i+1)*4])
			}
			c.Attestations = make([]*AttestationElectra, len(listOffsets))
			for i := 0; i < len(listOffsets); i++ {
				var tmp *AttestationElectra
				tmp = new(AttestationElectra)
				var tmpSlice []byte
				if i+1 == len(listOffsets) {
					tmpSlice = sszSlice5[listOffsets[i]:]
				} else {
					tmpSlice = sszSlice5[listOffsets[i]:listOffsets[i+1]]
				}
				if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
					return err
				}
				c.Attestations[i] = tmp
			}
		}
	}

	// Field 6: Deposits
	{
		if len(sszSlice6)%1240 != 0 {
			return fmt.Errorf("misaligned bytes: c.Deposits length is %d, which is not a multiple of 1240", len(sszSlice6))
		}
		numElem := len(sszSlice6) / 1240
		if numElem > 16 {
			return fmt.Errorf("ssz-max exceeded: c.Deposits has %d elements, ssz-max is 16", numElem)
		}
		c.Deposits = make([]*Deposit, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *Deposit
			tmp = new(Deposit)
			tmpSlice := sszSlice6[i*1240 : (1+i)*1240]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.Deposits[i] = tmp
		}
	}

	// Field 7: VoluntaryExits
	{
		if len(sszSlice7)%112 != 0 {
			return fmt.Errorf("misaligned bytes: c.VoluntaryExits length is %d, which is not a multiple of 112", len(sszSlice7))
		}
		numElem := len(sszSlice7) / 112
		if numElem > 16 {
			return fmt.Errorf("ssz-max exceeded: c.VoluntaryExits has %d elements, ssz-max is 16", numElem)
		}
		c.VoluntaryExits = make([]*SignedVoluntaryExit, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *SignedVoluntaryExit
			tmp = new(SignedVoluntaryExit)
			tmpSlice := sszSlice7[i*112 : (1+i)*112]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.VoluntaryExits[i] = tmp
		}
	}

	// Field 8: SyncAggregate
	c.SyncAggregate = new(SyncAggregate)
	if err = c.SyncAggregate.UnmarshalSSZ(sszSlice8); err != nil {
		return err
	}

	// Field 9: ExecutionPayloadHeader
	c.ExecutionPayloadHeader = new(v1.ExecutionPayloadHeaderElectra)
	if err = c.ExecutionPayloadHeader.UnmarshalSSZ(sszSlice9); err != nil {
		return err
	}

	// Field 10: BlsToExecutionChanges
	{
		if len(sszSlice10)%172 != 0 {
			return fmt.Errorf("misaligned bytes: c.BlsToExecutionChanges length is %d, which is not a multiple of 172", len(sszSlice10))
		}
		numElem := len(sszSlice10) / 172
		if numElem > 16 {
			return fmt.Errorf("ssz-max exceeded: c.BlsToExecutionChanges has %d elements, ssz-max is 16", numElem)
		}
		c.BlsToExecutionChanges = make([]*SignedBLSToExecutionChange, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *SignedBLSToExecutionChange
			tmp = new(SignedBLSToExecutionChange)
			tmpSlice := sszSlice10[i*172 : (1+i)*172]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.BlsToExecutionChanges[i] = tmp
		}
	}

	// Field 11: BlobKzgCommitments
	{
		if len(sszSlice11)%48 != 0 {
			return fmt.Errorf("misaligned bytes: c.BlobKzgCommitments length is %d, which is not a multiple of 48", len(sszSlice11))
		}
		numElem := len(sszSlice11) / 48
		if numElem > 4096 {
			return fmt.Errorf("ssz-max exceeded: c.BlobKzgCommitments has %d elements, ssz-max is 4096", numElem)
		}
		c.BlobKzgCommitments = make([][]byte, numElem)
		for i := 0; i < numElem; i++ {
			var tmp []byte

			tmpSlice := sszSlice11[i*48 : (1+i)*48]
			tmp = make([]byte, 0, 48)
			tmp = append(tmp, tmpSlice...)
			c.BlobKzgCommitments[i] = tmp
		}
	}

	// Field 12: Consolidations
	{
		if len(sszSlice12)%120 != 0 {
			return fmt.Errorf("misaligned bytes: c.Consolidations length is %d, which is not a multiple of 120", len(sszSlice12))
		}
		numElem := len(sszSlice12) / 120
		if numElem > 1 {
			return fmt.Errorf("ssz-max exceeded: c.Consolidations has %d elements, ssz-max is 1", numElem)
		}
		c.Consolidations = make([]*SignedConsolidation, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *SignedConsolidation
			tmp = new(SignedConsolidation)
			tmpSlice := sszSlice12[i*120 : (1+i)*120]
			if err = tmp.UnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.Consolidations[i] = tmp
		}
	}
	return err
}

func (c *BlindedBeaconBlockBodyElectra) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BlindedBeaconBlockBodyElectra) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: RandaoReveal
	if len(c.RandaoReveal) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.RandaoReveal)
	// Field 1: Eth1Data
	if err := c.Eth1Data.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 2: Graffiti
	if len(c.Graffiti) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Graffiti)
	// Field 3: ProposerSlashings
	{
		if len(c.ProposerSlashings) > 16 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.ProposerSlashings {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.ProposerSlashings)), 16)
	}
	// Field 4: AttesterSlashings
	{
		if len(c.AttesterSlashings) > 1 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.AttesterSlashings {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.AttesterSlashings)), 1)
	}
	// Field 5: Attestations
	{
		if len(c.Attestations) > 8 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.Attestations {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.Attestations)), 8)
	}
	// Field 6: Deposits
	{
		if len(c.Deposits) > 16 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.Deposits {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.Deposits)), 16)
	}
	// Field 7: VoluntaryExits
	{
		if len(c.VoluntaryExits) > 16 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.VoluntaryExits {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.VoluntaryExits)), 16)
	}
	// Field 8: SyncAggregate
	if err := c.SyncAggregate.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 9: ExecutionPayloadHeader
	if hash, err := c.ExecutionPayloadHeader.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 10: BlsToExecutionChanges
	{
		if len(c.BlsToExecutionChanges) > 16 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.BlsToExecutionChanges {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.BlsToExecutionChanges)), 16)
	}
	// Field 11: BlobKzgCommitments
	{
		if len(c.BlobKzgCommitments) > 4096 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.BlobKzgCommitments {
			if len(o) != 48 {
				return ssz.ErrBytesLength
			}
			hh.PutBytes(o)
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.BlobKzgCommitments)), 4096)
	}
	// Field 12: Consolidations
	{
		if len(c.Consolidations) > 1 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.Consolidations {
			if err := o.HashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.Consolidations)), 1)
	}
	hh.Merkleize(indx)
	return nil
}

func (c *BlindedBeaconBlockElectra) SizeSSZ() int {
	size := 84
	if c.Body == nil {
		c.Body = new(BlindedBeaconBlockBodyElectra)
	}
	size += c.Body.SizeSSZ()
	return size
}

func (c *BlindedBeaconBlockElectra) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *BlindedBeaconBlockElectra) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 84

	// Field 0: Slot
	if dst, err = c.Slot.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 1: ProposerIndex
	if dst, err = c.ProposerIndex.MarshalSSZTo(dst); err != nil {
		return nil, err
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
		c.Body = new(BlindedBeaconBlockBodyElectra)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Body.SizeSSZ()

	// Field 4: Body
	if dst, err = c.Body.MarshalSSZTo(dst); err != nil {
		return nil, err
	}
	return dst, err
}

func (c *BlindedBeaconBlockElectra) UnmarshalSSZ(buf []byte) error {
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
	if sszVarOffset4 < 84 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset4 > size {
		return ssz.ErrOffset
	}
	sszSlice4 := buf[sszVarOffset4:] // c.Body

	// Field 0: Slot
	if err = c.Slot.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: ProposerIndex
	if err = c.ProposerIndex.UnmarshalSSZ(sszSlice1); err != nil {
		return err
	}

	// Field 2: ParentRoot
	c.ParentRoot = make([]byte, 0, 32)
	c.ParentRoot = append(c.ParentRoot, sszSlice2...)

	// Field 3: StateRoot
	c.StateRoot = make([]byte, 0, 32)
	c.StateRoot = append(c.StateRoot, sszSlice3...)

	// Field 4: Body
	c.Body = new(BlindedBeaconBlockBodyElectra)
	if err = c.Body.UnmarshalSSZ(sszSlice4); err != nil {
		return err
	}
	return err
}

func (c *BlindedBeaconBlockElectra) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BlindedBeaconBlockElectra) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Slot
	if hash, err := c.Slot.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 1: ProposerIndex
	if hash, err := c.ProposerIndex.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
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
		return err
	}
	hh.Merkleize(indx)
	return nil
}

func (c *Consolidation) SizeSSZ() int {
	size := 24

	return size
}

func (c *Consolidation) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *Consolidation) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: SourceIndex
	if dst, err = c.SourceIndex.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 1: TargetIndex
	if dst, err = c.TargetIndex.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 2: Epoch
	if dst, err = c.Epoch.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	return dst, err
}

func (c *Consolidation) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 24 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:8]   // c.SourceIndex
	sszSlice1 := buf[8:16]  // c.TargetIndex
	sszSlice2 := buf[16:24] // c.Epoch

	// Field 0: SourceIndex
	if err = c.SourceIndex.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: TargetIndex
	if err = c.TargetIndex.UnmarshalSSZ(sszSlice1); err != nil {
		return err
	}

	// Field 2: Epoch
	if err = c.Epoch.UnmarshalSSZ(sszSlice2); err != nil {
		return err
	}
	return err
}

func (c *Consolidation) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *Consolidation) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: SourceIndex
	if hash, err := c.SourceIndex.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 1: TargetIndex
	if hash, err := c.TargetIndex.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 2: Epoch
	if hash, err := c.Epoch.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	hh.Merkleize(indx)
	return nil
}

func (c *IndexedAttestationElectra) SizeSSZ() int {
	size := 228
	size += len(c.AttestingIndices) * 8
	return size
}

func (c *IndexedAttestationElectra) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *IndexedAttestationElectra) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 228

	// Field 0: AttestingIndices
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.AttestingIndices) * 8

	// Field 1: Data
	if c.Data == nil {
		c.Data = new(AttestationData)
	}
	if dst, err = c.Data.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 2: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	// Field 0: AttestingIndices
	if len(c.AttestingIndices) > 131072 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.AttestingIndices {
		dst = ssz.MarshalUint64(dst, o)
	}
	return dst, err
}

func (c *IndexedAttestationElectra) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 228 {
		return ssz.ErrSize
	}

	sszSlice1 := buf[4:132]   // c.Data
	sszSlice2 := buf[132:228] // c.Signature

	sszVarOffset0 := ssz.ReadOffset(buf[0:4]) // c.AttestingIndices
	if sszVarOffset0 < 228 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset0 > size {
		return ssz.ErrOffset
	}
	sszSlice0 := buf[sszVarOffset0:] // c.AttestingIndices

	// Field 0: AttestingIndices
	{
		if len(sszSlice0)%8 != 0 {
			return fmt.Errorf("misaligned bytes: c.AttestingIndices length is %d, which is not a multiple of 8", len(sszSlice0))
		}
		numElem := len(sszSlice0) / 8
		if numElem > 131072 {
			return fmt.Errorf("ssz-max exceeded: c.AttestingIndices has %d elements, ssz-max is 131072", numElem)
		}
		c.AttestingIndices = make([]uint64, numElem)
		for i := 0; i < numElem; i++ {
			var tmp uint64

			tmpSlice := sszSlice0[i*8 : (1+i)*8]
			tmp = ssz.UnmarshallUint64(tmpSlice)
			c.AttestingIndices[i] = tmp
		}
	}

	// Field 1: Data
	c.Data = new(AttestationData)
	if err = c.Data.UnmarshalSSZ(sszSlice1); err != nil {
		return err
	}

	// Field 2: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, sszSlice2...)
	return err
}

func (c *IndexedAttestationElectra) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *IndexedAttestationElectra) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: AttestingIndices
	{
		if len(c.AttestingIndices) > 131072 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.AttestingIndices {
			hh.AppendUint64(o)
		}
		hh.FillUpTo32()
		numItems := uint64(len(c.AttestingIndices))
		hh.MerkleizeWithMixin(subIndx, numItems, ssz.CalculateLimit(131072, numItems, 8))
	}
	// Field 1: Data
	if err := c.Data.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 2: Signature
	if len(c.Signature) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Signature)
	hh.Merkleize(indx)
	return nil
}

func (c *PendingBalanceDeposit) SizeSSZ() int {
	size := 16

	return size
}

func (c *PendingBalanceDeposit) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *PendingBalanceDeposit) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Index
	if dst, err = c.Index.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 1: Amount
	dst = ssz.MarshalUint64(dst, c.Amount)

	return dst, err
}

func (c *PendingBalanceDeposit) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 16 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:8]  // c.Index
	sszSlice1 := buf[8:16] // c.Amount

	// Field 0: Index
	if err = c.Index.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: Amount
	c.Amount = ssz.UnmarshallUint64(sszSlice1)
	return err
}

func (c *PendingBalanceDeposit) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *PendingBalanceDeposit) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Index
	if hash, err := c.Index.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 1: Amount
	hh.PutUint64(c.Amount)
	hh.Merkleize(indx)
	return nil
}

func (c *PendingConsolidation) SizeSSZ() int {
	size := 16

	return size
}

func (c *PendingConsolidation) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *PendingConsolidation) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: SourceIndex
	if dst, err = c.SourceIndex.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 1: TargetIndex
	if dst, err = c.TargetIndex.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	return dst, err
}

func (c *PendingConsolidation) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 16 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:8]  // c.SourceIndex
	sszSlice1 := buf[8:16] // c.TargetIndex

	// Field 0: SourceIndex
	if err = c.SourceIndex.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: TargetIndex
	if err = c.TargetIndex.UnmarshalSSZ(sszSlice1); err != nil {
		return err
	}
	return err
}

func (c *PendingConsolidation) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *PendingConsolidation) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: SourceIndex
	if hash, err := c.SourceIndex.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 1: TargetIndex
	if hash, err := c.TargetIndex.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	hh.Merkleize(indx)
	return nil
}

func (c *PendingPartialWithdrawal) SizeSSZ() int {
	size := 24

	return size
}

func (c *PendingPartialWithdrawal) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *PendingPartialWithdrawal) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Index
	if dst, err = c.Index.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 1: Amount
	dst = ssz.MarshalUint64(dst, c.Amount)

	// Field 2: WithdrawableEpoch
	if dst, err = c.WithdrawableEpoch.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	return dst, err
}

func (c *PendingPartialWithdrawal) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 24 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:8]   // c.Index
	sszSlice1 := buf[8:16]  // c.Amount
	sszSlice2 := buf[16:24] // c.WithdrawableEpoch

	// Field 0: Index
	if err = c.Index.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: Amount
	c.Amount = ssz.UnmarshallUint64(sszSlice1)

	// Field 2: WithdrawableEpoch
	if err = c.WithdrawableEpoch.UnmarshalSSZ(sszSlice2); err != nil {
		return err
	}
	return err
}

func (c *PendingPartialWithdrawal) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *PendingPartialWithdrawal) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Index
	if hash, err := c.Index.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 1: Amount
	hh.PutUint64(c.Amount)
	// Field 2: WithdrawableEpoch
	if hash, err := c.WithdrawableEpoch.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	hh.Merkleize(indx)
	return nil
}

func (c *SignedAggregateAttestationAndProofElectra) SizeSSZ() int {
	size := 100
	if c.Message == nil {
		c.Message = new(AggregateAttestationAndProofElectra)
	}
	size += c.Message.SizeSSZ()
	return size
}

func (c *SignedAggregateAttestationAndProofElectra) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *SignedAggregateAttestationAndProofElectra) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 100

	// Field 0: Message
	if c.Message == nil {
		c.Message = new(AggregateAttestationAndProofElectra)
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
		return nil, err
	}
	return dst, err
}

func (c *SignedAggregateAttestationAndProofElectra) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 100 {
		return ssz.ErrSize
	}

	sszSlice1 := buf[4:100] // c.Signature

	sszVarOffset0 := ssz.ReadOffset(buf[0:4]) // c.Message
	if sszVarOffset0 < 100 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset0 > size {
		return ssz.ErrOffset
	}
	sszSlice0 := buf[sszVarOffset0:] // c.Message

	// Field 0: Message
	c.Message = new(AggregateAttestationAndProofElectra)
	if err = c.Message.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, sszSlice1...)
	return err
}

func (c *SignedAggregateAttestationAndProofElectra) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *SignedAggregateAttestationAndProofElectra) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Message
	if err := c.Message.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 1: Signature
	if len(c.Signature) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Signature)
	hh.Merkleize(indx)
	return nil
}

func (c *SignedBeaconBlockElectra) SizeSSZ() int {
	size := 100
	if c.Block == nil {
		c.Block = new(BeaconBlockElectra)
	}
	size += c.Block.SizeSSZ()
	return size
}

func (c *SignedBeaconBlockElectra) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *SignedBeaconBlockElectra) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 100

	// Field 0: Block
	if c.Block == nil {
		c.Block = new(BeaconBlockElectra)
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
		return nil, err
	}
	return dst, err
}

func (c *SignedBeaconBlockElectra) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 100 {
		return ssz.ErrSize
	}

	sszSlice1 := buf[4:100] // c.Signature

	sszVarOffset0 := ssz.ReadOffset(buf[0:4]) // c.Block
	if sszVarOffset0 < 100 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset0 > size {
		return ssz.ErrOffset
	}
	sszSlice0 := buf[sszVarOffset0:] // c.Block

	// Field 0: Block
	c.Block = new(BeaconBlockElectra)
	if err = c.Block.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, sszSlice1...)
	return err
}

func (c *SignedBeaconBlockElectra) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *SignedBeaconBlockElectra) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Block
	if err := c.Block.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 1: Signature
	if len(c.Signature) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Signature)
	hh.Merkleize(indx)
	return nil
}

func (c *SignedBlindedBeaconBlockElectra) SizeSSZ() int {
	size := 100
	if c.Message == nil {
		c.Message = new(BlindedBeaconBlockElectra)
	}
	size += c.Message.SizeSSZ()
	return size
}

func (c *SignedBlindedBeaconBlockElectra) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *SignedBlindedBeaconBlockElectra) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 100

	// Field 0: Message
	if c.Message == nil {
		c.Message = new(BlindedBeaconBlockElectra)
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
		return nil, err
	}
	return dst, err
}

func (c *SignedBlindedBeaconBlockElectra) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 100 {
		return ssz.ErrSize
	}

	sszSlice1 := buf[4:100] // c.Signature

	sszVarOffset0 := ssz.ReadOffset(buf[0:4]) // c.Message
	if sszVarOffset0 < 100 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset0 > size {
		return ssz.ErrOffset
	}
	sszSlice0 := buf[sszVarOffset0:] // c.Message

	// Field 0: Message
	c.Message = new(BlindedBeaconBlockElectra)
	if err = c.Message.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, sszSlice1...)
	return err
}

func (c *SignedBlindedBeaconBlockElectra) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *SignedBlindedBeaconBlockElectra) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Message
	if err := c.Message.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 1: Signature
	if len(c.Signature) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Signature)
	hh.Merkleize(indx)
	return nil
}

func (c *SignedConsolidation) SizeSSZ() int {
	size := 120

	return size
}

func (c *SignedConsolidation) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *SignedConsolidation) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Message
	if c.Message == nil {
		c.Message = new(Consolidation)
	}
	if dst, err = c.Message.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 1: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	return dst, err
}

func (c *SignedConsolidation) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 120 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:24]   // c.Message
	sszSlice1 := buf[24:120] // c.Signature

	// Field 0: Message
	c.Message = new(Consolidation)
	if err = c.Message.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, sszSlice1...)
	return err
}

func (c *SignedConsolidation) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *SignedConsolidation) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Message
	if err := c.Message.HashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 1: Signature
	if len(c.Signature) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Signature)
	hh.Merkleize(indx)
	return nil
}
