package v1

import (
	"fmt"
	ssz "github.com/ferranbt/fastssz"
	prysmaticlabs_eth2_types "github.com/prysmaticlabs/eth2-types"
	prysmaticlabs_go_bitfield "github.com/prysmaticlabs/go-bitfield"
)

func (c *AggregateAttestationAndProof) XXSizeSSZ() int {
	size := 108
	if c.Aggregate == nil {
		c.Aggregate = new(Attestation)
	}
	size += c.Aggregate.SizeSSZ()
	return size
}
func (c *AggregateAttestationAndProof) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *AggregateAttestationAndProof) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 108

	// Field 0: AggregatorIndex
	dst = ssz.MarshalUint64(dst, uint64(c.AggregatorIndex))

	// Field 1: Aggregate
	if c.Aggregate == nil {
		c.Aggregate = new(Attestation)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Aggregate.SizeSSZ()

	// Field 2: SelectionProof
	if len(c.SelectionProof) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.SelectionProof...)

	// Field 1: Aggregate
	if dst, err = c.Aggregate.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}
	return dst, err
}
func (c *AggregateAttestationAndProof) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 108 {
		return ssz.ErrSize
	}

	s0 := buf[0:8]    // c.AggregatorIndex
	s2 := buf[12:108] // c.SelectionProof

	v1 := ssz.ReadOffset(buf[8:12]) // c.Aggregate
	if v1 < 108 {
		return ssz.ErrInvalidVariableOffset
	}
	if v1 > size {
		return ssz.ErrOffset
	}
	s1 := buf[v1:] // c.Aggregate

	// Field 0: AggregatorIndex
	c.AggregatorIndex = prysmaticlabs_eth2_types.ValidatorIndex(ssz.UnmarshallUint64(s0))

	// Field 1: Aggregate
	c.Aggregate = new(Attestation)
	if err = c.Aggregate.XXUnmarshalSSZ(s1); err != nil {
		return err
	}

	// Field 2: SelectionProof
	c.SelectionProof = make([]byte, 0, 96)
	c.SelectionProof = append(c.SelectionProof, s2...)
	return err
}
func (c *AggregateAttestationAndProof) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *AggregateAttestationAndProof) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: AggregatorIndex
	hh.PutUint64(uint64(c.AggregatorIndex))
	// Field 1: Aggregate
	if err := c.Aggregate.XXHashTreeRootWith(hh); err != nil {
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
func (c *Attestation) XXSizeSSZ() int {
	size := 228

	return size
}
func (c *Attestation) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *Attestation) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 228

	// Field 0: AggregationBits
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.AggregationBits) * 1

	// Field 1: Data
	if c.Data == nil {
		c.Data = new(AttestationData)
	}
	if dst, err = c.Data.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 2: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	// Field 0: AggregationBits
	if len(c.AggregationBits) > 2048 {
		return nil, ssz.ErrListTooBig
	}
	dst = append(dst, c.AggregationBits...)
	return dst, err
}
func (c *Attestation) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 228 {
		return ssz.ErrSize
	}

	s1 := buf[4:132]   // c.Data
	s2 := buf[132:228] // c.Signature

	v0 := ssz.ReadOffset(buf[0:4]) // c.AggregationBits
	if v0 < 228 {
		return ssz.ErrInvalidVariableOffset
	}
	if v0 > size {
		return ssz.ErrOffset
	}
	s0 := buf[v0:] // c.AggregationBits

	// Field 0: AggregationBits
	if err = ssz.ValidateBitlist(s0, 2048); err != nil {
		return err
	}
	c.AggregationBits = append([]byte{}, prysmaticlabs_go_bitfield.Bitlist(s0)...)

	// Field 1: Data
	c.Data = new(AttestationData)
	if err = c.Data.XXUnmarshalSSZ(s1); err != nil {
		return err
	}

	// Field 2: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, s2...)
	return err
}
func (c *Attestation) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *Attestation) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: AggregationBits
	if len(c.AggregationBits) == 0 {
		return ssz.ErrEmptyBitlist
	}
	hh.PutBitlist(c.AggregationBits, 2048)
	// Field 1: Data
	if err := c.Data.XXHashTreeRootWith(hh); err != nil {
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
func (c *AttestationData) XXSizeSSZ() int {
	size := 128

	return size
}
func (c *AttestationData) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *AttestationData) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Slot
	dst = ssz.MarshalUint64(dst, uint64(c.Slot))

	// Field 1: Index
	dst = ssz.MarshalUint64(dst, uint64(c.Index))

	// Field 2: BeaconBlockRoot
	if len(c.BeaconBlockRoot) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.BeaconBlockRoot...)

	// Field 3: Source
	if c.Source == nil {
		c.Source = new(Checkpoint)
	}
	if dst, err = c.Source.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 4: Target
	if c.Target == nil {
		c.Target = new(Checkpoint)
	}
	if dst, err = c.Target.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}

	return dst, err
}
func (c *AttestationData) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 128 {
		return ssz.ErrSize
	}

	s0 := buf[0:8]    // c.Slot
	s1 := buf[8:16]   // c.Index
	s2 := buf[16:48]  // c.BeaconBlockRoot
	s3 := buf[48:88]  // c.Source
	s4 := buf[88:128] // c.Target

	// Field 0: Slot
	c.Slot = prysmaticlabs_eth2_types.Slot(ssz.UnmarshallUint64(s0))

	// Field 1: Index
	c.Index = prysmaticlabs_eth2_types.CommitteeIndex(ssz.UnmarshallUint64(s1))

	// Field 2: BeaconBlockRoot
	c.BeaconBlockRoot = make([]byte, 0, 32)
	c.BeaconBlockRoot = append(c.BeaconBlockRoot, s2...)

	// Field 3: Source
	c.Source = new(Checkpoint)
	if err = c.Source.XXUnmarshalSSZ(s3); err != nil {
		return err
	}

	// Field 4: Target
	c.Target = new(Checkpoint)
	if err = c.Target.XXUnmarshalSSZ(s4); err != nil {
		return err
	}
	return err
}
func (c *AttestationData) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *AttestationData) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Slot
	hh.PutUint64(uint64(c.Slot))
	// Field 1: Index
	hh.PutUint64(uint64(c.Index))
	// Field 2: BeaconBlockRoot
	if len(c.BeaconBlockRoot) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.BeaconBlockRoot)
	// Field 3: Source
	if err := c.Source.XXHashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 4: Target
	if err := c.Target.XXHashTreeRootWith(hh); err != nil {
		return err
	}
	hh.Merkleize(indx)
	return nil
}
func (c *AttesterSlashing) XXSizeSSZ() int {
	size := 8
	if c.Attestation_1 == nil {
		c.Attestation_1 = new(IndexedAttestation)
	}
	size += c.Attestation_1.SizeSSZ()
	if c.Attestation_2 == nil {
		c.Attestation_2 = new(IndexedAttestation)
	}
	size += c.Attestation_2.SizeSSZ()
	return size
}
func (c *AttesterSlashing) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *AttesterSlashing) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 8

	// Field 0: Attestation_1
	if c.Attestation_1 == nil {
		c.Attestation_1 = new(IndexedAttestation)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Attestation_1.SizeSSZ()

	// Field 1: Attestation_2
	if c.Attestation_2 == nil {
		c.Attestation_2 = new(IndexedAttestation)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Attestation_2.SizeSSZ()

	// Field 0: Attestation_1
	if dst, err = c.Attestation_1.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 1: Attestation_2
	if dst, err = c.Attestation_2.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}
	return dst, err
}
func (c *AttesterSlashing) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 8 {
		return ssz.ErrSize
	}

	v0 := ssz.ReadOffset(buf[0:4]) // c.Attestation_1
	if v0 < 8 {
		return ssz.ErrInvalidVariableOffset
	}
	if v0 > size {
		return ssz.ErrOffset
	}
	v1 := ssz.ReadOffset(buf[4:8]) // c.Attestation_2
	if v1 > size || v1 < v0 {
		return ssz.ErrOffset
	}
	s0 := buf[v0:v1] // c.Attestation_1
	s1 := buf[v1:]   // c.Attestation_2

	// Field 0: Attestation_1
	c.Attestation_1 = new(IndexedAttestation)
	if err = c.Attestation_1.XXUnmarshalSSZ(s0); err != nil {
		return err
	}

	// Field 1: Attestation_2
	c.Attestation_2 = new(IndexedAttestation)
	if err = c.Attestation_2.XXUnmarshalSSZ(s1); err != nil {
		return err
	}
	return err
}
func (c *AttesterSlashing) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *AttesterSlashing) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Attestation_1
	if err := c.Attestation_1.XXHashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 1: Attestation_2
	if err := c.Attestation_2.XXHashTreeRootWith(hh); err != nil {
		return err
	}
	hh.Merkleize(indx)
	return nil
}
func (c *BeaconBlock) XXSizeSSZ() int {
	size := 84
	if c.Body == nil {
		c.Body = new(BeaconBlockBody)
	}
	size += c.Body.SizeSSZ()
	return size
}
func (c *BeaconBlock) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *BeaconBlock) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 84

	// Field 0: Slot
	dst = ssz.MarshalUint64(dst, uint64(c.Slot))

	// Field 1: ProposerIndex
	dst = ssz.MarshalUint64(dst, uint64(c.ProposerIndex))

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
		c.Body = new(BeaconBlockBody)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Body.SizeSSZ()

	// Field 4: Body
	if dst, err = c.Body.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}
	return dst, err
}
func (c *BeaconBlock) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 84 {
		return ssz.ErrSize
	}

	s0 := buf[0:8]   // c.Slot
	s1 := buf[8:16]  // c.ProposerIndex
	s2 := buf[16:48] // c.ParentRoot
	s3 := buf[48:80] // c.StateRoot

	v4 := ssz.ReadOffset(buf[80:84]) // c.Body
	if v4 < 84 {
		return ssz.ErrInvalidVariableOffset
	}
	if v4 > size {
		return ssz.ErrOffset
	}
	s4 := buf[v4:] // c.Body

	// Field 0: Slot
	c.Slot = prysmaticlabs_eth2_types.Slot(ssz.UnmarshallUint64(s0))

	// Field 1: ProposerIndex
	c.ProposerIndex = prysmaticlabs_eth2_types.ValidatorIndex(ssz.UnmarshallUint64(s1))

	// Field 2: ParentRoot
	c.ParentRoot = make([]byte, 0, 32)
	c.ParentRoot = append(c.ParentRoot, s2...)

	// Field 3: StateRoot
	c.StateRoot = make([]byte, 0, 32)
	c.StateRoot = append(c.StateRoot, s3...)

	// Field 4: Body
	c.Body = new(BeaconBlockBody)
	if err = c.Body.XXUnmarshalSSZ(s4); err != nil {
		return err
	}
	return err
}
func (c *BeaconBlock) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BeaconBlock) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Slot
	hh.PutUint64(uint64(c.Slot))
	// Field 1: ProposerIndex
	hh.PutUint64(uint64(c.ProposerIndex))
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
	if err := c.Body.XXHashTreeRootWith(hh); err != nil {
		return err
	}
	hh.Merkleize(indx)
	return nil
}
func (c *BeaconBlockBody) XXSizeSSZ() int {
	size := 220
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
	return size
}
func (c *BeaconBlockBody) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *BeaconBlockBody) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 220

	// Field 0: RandaoReveal
	if len(c.RandaoReveal) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.RandaoReveal...)

	// Field 1: Eth1Data
	if c.Eth1Data == nil {
		c.Eth1Data = new(Eth1Data)
	}
	if dst, err = c.Eth1Data.XXMarshalSSZTo(dst); err != nil {
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

	// Field 3: ProposerSlashings
	if len(c.ProposerSlashings) > 16 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.ProposerSlashings {
		if dst, err = o.XXMarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 4: AttesterSlashings
	if len(c.AttesterSlashings) > 2 {
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
		if dst, err = o.XXMarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 5: Attestations
	if len(c.Attestations) > 128 {
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
		if dst, err = o.XXMarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 6: Deposits
	if len(c.Deposits) > 16 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.Deposits {
		if dst, err = o.XXMarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 7: VoluntaryExits
	if len(c.VoluntaryExits) > 16 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.VoluntaryExits {
		if dst, err = o.XXMarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}
	return dst, err
}
func (c *BeaconBlockBody) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 220 {
		return ssz.ErrSize
	}

	s0 := buf[0:96]    // c.RandaoReveal
	s1 := buf[96:168]  // c.Eth1Data
	s2 := buf[168:200] // c.Graffiti

	v3 := ssz.ReadOffset(buf[200:204]) // c.ProposerSlashings
	if v3 < 220 {
		return ssz.ErrInvalidVariableOffset
	}
	if v3 > size {
		return ssz.ErrOffset
	}
	v4 := ssz.ReadOffset(buf[204:208]) // c.AttesterSlashings
	if v4 > size || v4 < v3 {
		return ssz.ErrOffset
	}
	v5 := ssz.ReadOffset(buf[208:212]) // c.Attestations
	if v5 > size || v5 < v4 {
		return ssz.ErrOffset
	}
	v6 := ssz.ReadOffset(buf[212:216]) // c.Deposits
	if v6 > size || v6 < v5 {
		return ssz.ErrOffset
	}
	v7 := ssz.ReadOffset(buf[216:220]) // c.VoluntaryExits
	if v7 > size || v7 < v6 {
		return ssz.ErrOffset
	}
	s3 := buf[v3:v4] // c.ProposerSlashings
	s4 := buf[v4:v5] // c.AttesterSlashings
	s5 := buf[v5:v6] // c.Attestations
	s6 := buf[v6:v7] // c.Deposits
	s7 := buf[v7:]   // c.VoluntaryExits

	// Field 0: RandaoReveal
	c.RandaoReveal = make([]byte, 0, 96)
	c.RandaoReveal = append(c.RandaoReveal, s0...)

	// Field 1: Eth1Data
	c.Eth1Data = new(Eth1Data)
	if err = c.Eth1Data.XXUnmarshalSSZ(s1); err != nil {
		return err
	}

	// Field 2: Graffiti
	c.Graffiti = make([]byte, 0, 32)
	c.Graffiti = append(c.Graffiti, s2...)

	// Field 3: ProposerSlashings
	{
		if len(s3)%416 != 0 {
			return fmt.Errorf("misaligned bytes: c.ProposerSlashings length is %d, which is not a multiple of 416", len(s3))
		}
		numElem := len(s3) / 416
		if numElem > 16 {
			return fmt.Errorf("ssz-max exceeded: c.ProposerSlashings has %d elements, ssz-max is 16", numElem)
		}
		c.ProposerSlashings = make([]*ProposerSlashing, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *ProposerSlashing
			tmp = new(ProposerSlashing)
			tmpSlice := s3[i*416 : (1+i)*416]
			if err = tmp.XXUnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.ProposerSlashings[i] = tmp
		}
	}

	// Field 4: AttesterSlashings
	{
		// empty lists are zero length, so make sure there is room for an offset
		// before attempting to unmarshal it
		if len(s4) > 3 {
			firstOffset := ssz.ReadOffset(s4[0:4])
			if firstOffset%4 != 0 {
				return fmt.Errorf("misaligned list bytes: when decoding c.AttesterSlashings, end-of-list offset is %d, which is not a multiple of 4 (offset size)", firstOffset)
			}
			listLen := firstOffset / 4
			if listLen > 2 {
				return fmt.Errorf("ssz-max exceeded: c.AttesterSlashings has %d elements, ssz-max is 2", listLen)
			}
			listOffsets := make([]uint64, listLen)
			for i := 0; uint64(i) < listLen; i++ {
				listOffsets[i] = ssz.ReadOffset(s4[i*4 : (i+1)*4])
			}
			c.AttesterSlashings = make([]*AttesterSlashing, len(listOffsets))
			for i := 0; i < len(listOffsets); i++ {
				var tmp *AttesterSlashing
				tmp = new(AttesterSlashing)
				var tmpSlice []byte
				if i+1 == len(listOffsets) {
					tmpSlice = s4[listOffsets[i]:]
				} else {
					tmpSlice = s4[listOffsets[i]:listOffsets[i+1]]
				}
				if err = tmp.XXUnmarshalSSZ(tmpSlice); err != nil {
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
		if len(s5) > 3 {
			firstOffset := ssz.ReadOffset(s5[0:4])
			if firstOffset%4 != 0 {
				return fmt.Errorf("misaligned list bytes: when decoding c.Attestations, end-of-list offset is %d, which is not a multiple of 4 (offset size)", firstOffset)
			}
			listLen := firstOffset / 4
			if listLen > 128 {
				return fmt.Errorf("ssz-max exceeded: c.Attestations has %d elements, ssz-max is 128", listLen)
			}
			listOffsets := make([]uint64, listLen)
			for i := 0; uint64(i) < listLen; i++ {
				listOffsets[i] = ssz.ReadOffset(s5[i*4 : (i+1)*4])
			}
			c.Attestations = make([]*Attestation, len(listOffsets))
			for i := 0; i < len(listOffsets); i++ {
				var tmp *Attestation
				tmp = new(Attestation)
				var tmpSlice []byte
				if i+1 == len(listOffsets) {
					tmpSlice = s5[listOffsets[i]:]
				} else {
					tmpSlice = s5[listOffsets[i]:listOffsets[i+1]]
				}
				if err = tmp.XXUnmarshalSSZ(tmpSlice); err != nil {
					return err
				}
				c.Attestations[i] = tmp
			}
		}
	}

	// Field 6: Deposits
	{
		if len(s6)%1240 != 0 {
			return fmt.Errorf("misaligned bytes: c.Deposits length is %d, which is not a multiple of 1240", len(s6))
		}
		numElem := len(s6) / 1240
		if numElem > 16 {
			return fmt.Errorf("ssz-max exceeded: c.Deposits has %d elements, ssz-max is 16", numElem)
		}
		c.Deposits = make([]*Deposit, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *Deposit
			tmp = new(Deposit)
			tmpSlice := s6[i*1240 : (1+i)*1240]
			if err = tmp.XXUnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.Deposits[i] = tmp
		}
	}

	// Field 7: VoluntaryExits
	{
		if len(s7)%112 != 0 {
			return fmt.Errorf("misaligned bytes: c.VoluntaryExits length is %d, which is not a multiple of 112", len(s7))
		}
		numElem := len(s7) / 112
		if numElem > 16 {
			return fmt.Errorf("ssz-max exceeded: c.VoluntaryExits has %d elements, ssz-max is 16", numElem)
		}
		c.VoluntaryExits = make([]*SignedVoluntaryExit, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *SignedVoluntaryExit
			tmp = new(SignedVoluntaryExit)
			tmpSlice := s7[i*112 : (1+i)*112]
			if err = tmp.XXUnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.VoluntaryExits[i] = tmp
		}
	}
	return err
}
func (c *BeaconBlockBody) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BeaconBlockBody) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: RandaoReveal
	if len(c.RandaoReveal) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.RandaoReveal)
	// Field 1: Eth1Data
	if err := c.Eth1Data.XXHashTreeRootWith(hh); err != nil {
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
			if err := o.XXHashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.ProposerSlashings)), 16)
	}
	// Field 4: AttesterSlashings
	{
		if len(c.AttesterSlashings) > 2 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.AttesterSlashings {
			if err := o.XXHashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.AttesterSlashings)), 2)
	}
	// Field 5: Attestations
	{
		if len(c.Attestations) > 128 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.Attestations {
			if err := o.XXHashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.Attestations)), 128)
	}
	// Field 6: Deposits
	{
		if len(c.Deposits) > 16 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.Deposits {
			if err := o.XXHashTreeRootWith(hh); err != nil {
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
			if err := o.XXHashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.VoluntaryExits)), 16)
	}
	hh.Merkleize(indx)
	return nil
}
func (c *BeaconBlockV1) XXSizeSSZ() int {
	size := 84
	if c.Body == nil {
		c.Body = new(BeaconBlockBodyV1)
	}
	size += c.Body.SizeSSZ()
	return size
}
func (c *BeaconBlockV1) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *BeaconBlockV1) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 84

	// Field 0: Slot
	dst = ssz.MarshalUint64(dst, uint64(c.Slot))

	// Field 1: ProposerIndex
	dst = ssz.MarshalUint64(dst, uint64(c.ProposerIndex))

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
		c.Body = new(BeaconBlockBodyV1)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Body.SizeSSZ()

	// Field 4: Body
	if dst, err = c.Body.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}
	return dst, err
}
func (c *BeaconBlockV1) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 84 {
		return ssz.ErrSize
	}

	s0 := buf[0:8]   // c.Slot
	s1 := buf[8:16]  // c.ProposerIndex
	s2 := buf[16:48] // c.ParentRoot
	s3 := buf[48:80] // c.StateRoot

	v4 := ssz.ReadOffset(buf[80:84]) // c.Body
	if v4 < 84 {
		return ssz.ErrInvalidVariableOffset
	}
	if v4 > size {
		return ssz.ErrOffset
	}
	s4 := buf[v4:] // c.Body

	// Field 0: Slot
	c.Slot = prysmaticlabs_eth2_types.Slot(ssz.UnmarshallUint64(s0))

	// Field 1: ProposerIndex
	c.ProposerIndex = prysmaticlabs_eth2_types.ValidatorIndex(ssz.UnmarshallUint64(s1))

	// Field 2: ParentRoot
	c.ParentRoot = make([]byte, 0, 32)
	c.ParentRoot = append(c.ParentRoot, s2...)

	// Field 3: StateRoot
	c.StateRoot = make([]byte, 0, 32)
	c.StateRoot = append(c.StateRoot, s3...)

	// Field 4: Body
	c.Body = new(BeaconBlockBodyV1)
	if err = c.Body.XXUnmarshalSSZ(s4); err != nil {
		return err
	}
	return err
}
func (c *BeaconBlockV1) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BeaconBlockV1) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Slot
	hh.PutUint64(uint64(c.Slot))
	// Field 1: ProposerIndex
	hh.PutUint64(uint64(c.ProposerIndex))
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
	if err := c.Body.XXHashTreeRootWith(hh); err != nil {
		return err
	}
	hh.Merkleize(indx)
	return nil
}
func (c *BeaconBlockHeader) XXSizeSSZ() int {
	size := 112

	return size
}
func (c *BeaconBlockHeader) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *BeaconBlockHeader) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Slot
	dst = ssz.MarshalUint64(dst, uint64(c.Slot))

	// Field 1: ProposerIndex
	dst = ssz.MarshalUint64(dst, uint64(c.ProposerIndex))

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

	// Field 4: BodyRoot
	if len(c.BodyRoot) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.BodyRoot...)

	return dst, err
}
func (c *BeaconBlockHeader) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 112 {
		return ssz.ErrSize
	}

	s0 := buf[0:8]    // c.Slot
	s1 := buf[8:16]   // c.ProposerIndex
	s2 := buf[16:48]  // c.ParentRoot
	s3 := buf[48:80]  // c.StateRoot
	s4 := buf[80:112] // c.BodyRoot

	// Field 0: Slot
	c.Slot = prysmaticlabs_eth2_types.Slot(ssz.UnmarshallUint64(s0))

	// Field 1: ProposerIndex
	c.ProposerIndex = prysmaticlabs_eth2_types.ValidatorIndex(ssz.UnmarshallUint64(s1))

	// Field 2: ParentRoot
	c.ParentRoot = make([]byte, 0, 32)
	c.ParentRoot = append(c.ParentRoot, s2...)

	// Field 3: StateRoot
	c.StateRoot = make([]byte, 0, 32)
	c.StateRoot = append(c.StateRoot, s3...)

	// Field 4: BodyRoot
	c.BodyRoot = make([]byte, 0, 32)
	c.BodyRoot = append(c.BodyRoot, s4...)
	return err
}
func (c *BeaconBlockHeader) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BeaconBlockHeader) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Slot
	hh.PutUint64(uint64(c.Slot))
	// Field 1: ProposerIndex
	hh.PutUint64(uint64(c.ProposerIndex))
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
	// Field 4: BodyRoot
	if len(c.BodyRoot) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.BodyRoot)
	hh.Merkleize(indx)
	return nil
}
func (c *BeaconBlockBodyV1) XXSizeSSZ() int {
	size := 380
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
	return size
}
func (c *BeaconBlockBodyV1) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *BeaconBlockBodyV1) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 380

	// Field 0: RandaoReveal
	if len(c.RandaoReveal) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.RandaoReveal...)

	// Field 1: Eth1Data
	if c.Eth1Data == nil {
		c.Eth1Data = new(Eth1Data)
	}
	if dst, err = c.Eth1Data.XXMarshalSSZTo(dst); err != nil {
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

	// Field 8: SyncCommitteeBits
	if len([]byte(c.SyncCommitteeBits)) != 64 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, []byte(c.SyncCommitteeBits)...)

	// Field 9: SyncCommitteeSignature
	if len(c.SyncCommitteeSignature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.SyncCommitteeSignature...)

	// Field 3: ProposerSlashings
	if len(c.ProposerSlashings) > 16 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.ProposerSlashings {
		if dst, err = o.XXMarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 4: AttesterSlashings
	if len(c.AttesterSlashings) > 2 {
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
		if dst, err = o.XXMarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 5: Attestations
	if len(c.Attestations) > 128 {
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
		if dst, err = o.XXMarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 6: Deposits
	if len(c.Deposits) > 16 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.Deposits {
		if dst, err = o.XXMarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}

	// Field 7: VoluntaryExits
	if len(c.VoluntaryExits) > 16 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.VoluntaryExits {
		if dst, err = o.XXMarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}
	return dst, err
}
func (c *BeaconBlockBodyV1) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 380 {
		return ssz.ErrSize
	}

	s0 := buf[0:96]    // c.RandaoReveal
	s1 := buf[96:168]  // c.Eth1Data
	s2 := buf[168:200] // c.Graffiti
	s8 := buf[220:284] // c.SyncCommitteeBits
	s9 := buf[284:380] // c.SyncCommitteeSignature

	v3 := ssz.ReadOffset(buf[200:204]) // c.ProposerSlashings
	if v3 < 380 {
		return ssz.ErrInvalidVariableOffset
	}
	if v3 > size {
		return ssz.ErrOffset
	}
	v4 := ssz.ReadOffset(buf[204:208]) // c.AttesterSlashings
	if v4 > size || v4 < v3 {
		return ssz.ErrOffset
	}
	v5 := ssz.ReadOffset(buf[208:212]) // c.Attestations
	if v5 > size || v5 < v4 {
		return ssz.ErrOffset
	}
	v6 := ssz.ReadOffset(buf[212:216]) // c.Deposits
	if v6 > size || v6 < v5 {
		return ssz.ErrOffset
	}
	v7 := ssz.ReadOffset(buf[216:220]) // c.VoluntaryExits
	if v7 > size || v7 < v6 {
		return ssz.ErrOffset
	}
	s3 := buf[v3:v4] // c.ProposerSlashings
	s4 := buf[v4:v5] // c.AttesterSlashings
	s5 := buf[v5:v6] // c.Attestations
	s6 := buf[v6:v7] // c.Deposits
	s7 := buf[v7:]   // c.VoluntaryExits

	// Field 0: RandaoReveal
	c.RandaoReveal = make([]byte, 0, 96)
	c.RandaoReveal = append(c.RandaoReveal, s0...)

	// Field 1: Eth1Data
	c.Eth1Data = new(Eth1Data)
	if err = c.Eth1Data.XXUnmarshalSSZ(s1); err != nil {
		return err
	}

	// Field 2: Graffiti
	c.Graffiti = make([]byte, 0, 32)
	c.Graffiti = append(c.Graffiti, s2...)

	// Field 3: ProposerSlashings
	{
		if len(s3)%416 != 0 {
			return fmt.Errorf("misaligned bytes: c.ProposerSlashings length is %d, which is not a multiple of 416", len(s3))
		}
		numElem := len(s3) / 416
		if numElem > 16 {
			return fmt.Errorf("ssz-max exceeded: c.ProposerSlashings has %d elements, ssz-max is 16", numElem)
		}
		c.ProposerSlashings = make([]*ProposerSlashing, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *ProposerSlashing
			tmp = new(ProposerSlashing)
			tmpSlice := s3[i*416 : (1+i)*416]
			if err = tmp.XXUnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.ProposerSlashings[i] = tmp
		}
	}

	// Field 4: AttesterSlashings
	{
		// empty lists are zero length, so make sure there is room for an offset
		// before attempting to unmarshal it
		if len(s4) > 3 {
			firstOffset := ssz.ReadOffset(s4[0:4])
			if firstOffset%4 != 0 {
				return fmt.Errorf("misaligned list bytes: when decoding c.AttesterSlashings, end-of-list offset is %d, which is not a multiple of 4 (offset size)", firstOffset)
			}
			listLen := firstOffset / 4
			if listLen > 2 {
				return fmt.Errorf("ssz-max exceeded: c.AttesterSlashings has %d elements, ssz-max is 2", listLen)
			}
			listOffsets := make([]uint64, listLen)
			for i := 0; uint64(i) < listLen; i++ {
				listOffsets[i] = ssz.ReadOffset(s4[i*4 : (i+1)*4])
			}
			c.AttesterSlashings = make([]*AttesterSlashing, len(listOffsets))
			for i := 0; i < len(listOffsets); i++ {
				var tmp *AttesterSlashing
				tmp = new(AttesterSlashing)
				var tmpSlice []byte
				if i+1 == len(listOffsets) {
					tmpSlice = s4[listOffsets[i]:]
				} else {
					tmpSlice = s4[listOffsets[i]:listOffsets[i+1]]
				}
				if err = tmp.XXUnmarshalSSZ(tmpSlice); err != nil {
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
		if len(s5) > 3 {
			firstOffset := ssz.ReadOffset(s5[0:4])
			if firstOffset%4 != 0 {
				return fmt.Errorf("misaligned list bytes: when decoding c.Attestations, end-of-list offset is %d, which is not a multiple of 4 (offset size)", firstOffset)
			}
			listLen := firstOffset / 4
			if listLen > 128 {
				return fmt.Errorf("ssz-max exceeded: c.Attestations has %d elements, ssz-max is 128", listLen)
			}
			listOffsets := make([]uint64, listLen)
			for i := 0; uint64(i) < listLen; i++ {
				listOffsets[i] = ssz.ReadOffset(s5[i*4 : (i+1)*4])
			}
			c.Attestations = make([]*Attestation, len(listOffsets))
			for i := 0; i < len(listOffsets); i++ {
				var tmp *Attestation
				tmp = new(Attestation)
				var tmpSlice []byte
				if i+1 == len(listOffsets) {
					tmpSlice = s5[listOffsets[i]:]
				} else {
					tmpSlice = s5[listOffsets[i]:listOffsets[i+1]]
				}
				if err = tmp.XXUnmarshalSSZ(tmpSlice); err != nil {
					return err
				}
				c.Attestations[i] = tmp
			}
		}
	}

	// Field 6: Deposits
	{
		if len(s6)%1240 != 0 {
			return fmt.Errorf("misaligned bytes: c.Deposits length is %d, which is not a multiple of 1240", len(s6))
		}
		numElem := len(s6) / 1240
		if numElem > 16 {
			return fmt.Errorf("ssz-max exceeded: c.Deposits has %d elements, ssz-max is 16", numElem)
		}
		c.Deposits = make([]*Deposit, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *Deposit
			tmp = new(Deposit)
			tmpSlice := s6[i*1240 : (1+i)*1240]
			if err = tmp.XXUnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.Deposits[i] = tmp
		}
	}

	// Field 7: VoluntaryExits
	{
		if len(s7)%112 != 0 {
			return fmt.Errorf("misaligned bytes: c.VoluntaryExits length is %d, which is not a multiple of 112", len(s7))
		}
		numElem := len(s7) / 112
		if numElem > 16 {
			return fmt.Errorf("ssz-max exceeded: c.VoluntaryExits has %d elements, ssz-max is 16", numElem)
		}
		c.VoluntaryExits = make([]*SignedVoluntaryExit, numElem)
		for i := 0; i < numElem; i++ {
			var tmp *SignedVoluntaryExit
			tmp = new(SignedVoluntaryExit)
			tmpSlice := s7[i*112 : (1+i)*112]
			if err = tmp.XXUnmarshalSSZ(tmpSlice); err != nil {
				return err
			}
			c.VoluntaryExits[i] = tmp
		}
	}

	// Field 8: SyncCommitteeBits
	c.SyncCommitteeBits = make([]byte, 0, 64)
	c.SyncCommitteeBits = append(c.SyncCommitteeBits, prysmaticlabs_go_bitfield.Bitvector512(s8)...)

	// Field 9: SyncCommitteeSignature
	c.SyncCommitteeSignature = make([]byte, 0, 96)
	c.SyncCommitteeSignature = append(c.SyncCommitteeSignature, s9...)
	return err
}
func (c *BeaconBlockBodyV1) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BeaconBlockBodyV1) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: RandaoReveal
	if len(c.RandaoReveal) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.RandaoReveal)
	// Field 1: Eth1Data
	if err := c.Eth1Data.XXHashTreeRootWith(hh); err != nil {
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
			if err := o.XXHashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.ProposerSlashings)), 16)
	}
	// Field 4: AttesterSlashings
	{
		if len(c.AttesterSlashings) > 2 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.AttesterSlashings {
			if err := o.XXHashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.AttesterSlashings)), 2)
	}
	// Field 5: Attestations
	{
		if len(c.Attestations) > 128 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.Attestations {
			if err := o.XXHashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.Attestations)), 128)
	}
	// Field 6: Deposits
	{
		if len(c.Deposits) > 16 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.Deposits {
			if err := o.XXHashTreeRootWith(hh); err != nil {
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
			if err := o.XXHashTreeRootWith(hh); err != nil {
				return err
			}
		}
		hh.MerkleizeWithMixin(subIndx, uint64(len(c.VoluntaryExits)), 16)
	}
	// Field 8: SyncCommitteeBits
	if len([]byte(c.SyncCommitteeBits)) != 64 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes([]byte(c.SyncCommitteeBits))
	// Field 9: SyncCommitteeSignature
	if len(c.SyncCommitteeSignature) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.SyncCommitteeSignature)
	hh.Merkleize(indx)
	return nil
}
func (c *Checkpoint) XXSizeSSZ() int {
	size := 40

	return size
}
func (c *Checkpoint) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *Checkpoint) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Epoch
	dst = ssz.MarshalUint64(dst, uint64(c.Epoch))

	// Field 1: Root
	if len(c.Root) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Root...)

	return dst, err
}
func (c *Checkpoint) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 40 {
		return ssz.ErrSize
	}

	s0 := buf[0:8]  // c.Epoch
	s1 := buf[8:40] // c.Root

	// Field 0: Epoch
	c.Epoch = prysmaticlabs_eth2_types.Epoch(ssz.UnmarshallUint64(s0))

	// Field 1: Root
	c.Root = make([]byte, 0, 32)
	c.Root = append(c.Root, s1...)
	return err
}
func (c *Checkpoint) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *Checkpoint) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Epoch
	hh.PutUint64(uint64(c.Epoch))
	// Field 1: Root
	if len(c.Root) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Root)
	hh.Merkleize(indx)
	return nil
}
func (c *Deposit) XXSizeSSZ() int {
	size := 1240

	return size
}
func (c *Deposit) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *Deposit) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Proof
	if len(c.Proof) != 33 {
		return nil, ssz.ErrBytesLength
	}
	for _, o := range c.Proof {
		if len(o) != 32 {
			return nil, ssz.ErrBytesLength
		}
		dst = append(dst, o...)
	}

	// Field 1: Data
	if c.Data == nil {
		c.Data = new(Deposit_Data)
	}
	if dst, err = c.Data.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}

	return dst, err
}
func (c *Deposit) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 1240 {
		return ssz.ErrSize
	}

	s0 := buf[0:1056]    // c.Proof
	s1 := buf[1056:1240] // c.Data

	// Field 0: Proof
	{
		var tmp []byte
		c.Proof = make([][]byte, 33)
		for i := 0; i < 33; i++ {
			tmpSlice := s0[i*32 : (1+i)*32]
			tmp = make([]byte, 0, 32)
			tmp = append(tmp, tmpSlice...)
			c.Proof[i] = tmp
		}
	}

	// Field 1: Data
	c.Data = new(Deposit_Data)
	if err = c.Data.XXUnmarshalSSZ(s1); err != nil {
		return err
	}
	return err
}
func (c *Deposit) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *Deposit) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Proof
	{
		if len(c.Proof) != 33 {
			return ssz.ErrVectorLength
		}
		subIndx := hh.Index()
		for _, o := range c.Proof {
			if len(o) != 32 {
				return ssz.ErrBytesLength
			}
			hh.Append(o)
		}
		hh.Merkleize(subIndx)
	}
	// Field 1: Data
	if err := c.Data.XXHashTreeRootWith(hh); err != nil {
		return err
	}
	hh.Merkleize(indx)
	return nil
}
func (c *Deposit_Data) XXSizeSSZ() int {
	size := 184

	return size
}
func (c *Deposit_Data) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *Deposit_Data) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Pubkey
	if len(c.Pubkey) != 48 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Pubkey...)

	// Field 1: WithdrawalCredentials
	if len(c.WithdrawalCredentials) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.WithdrawalCredentials...)

	// Field 2: Amount
	dst = ssz.MarshalUint64(dst, c.Amount)

	// Field 3: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	return dst, err
}
func (c *Deposit_Data) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 184 {
		return ssz.ErrSize
	}

	s0 := buf[0:48]   // c.Pubkey
	s1 := buf[48:80]  // c.WithdrawalCredentials
	s2 := buf[80:88]  // c.Amount
	s3 := buf[88:184] // c.Signature

	// Field 0: Pubkey
	c.Pubkey = make([]byte, 0, 48)
	c.Pubkey = append(c.Pubkey, s0...)

	// Field 1: WithdrawalCredentials
	c.WithdrawalCredentials = make([]byte, 0, 32)
	c.WithdrawalCredentials = append(c.WithdrawalCredentials, s1...)

	// Field 2: Amount
	c.Amount = ssz.UnmarshallUint64(s2)

	// Field 3: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, s3...)
	return err
}
func (c *Deposit_Data) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *Deposit_Data) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Pubkey
	if len(c.Pubkey) != 48 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Pubkey)
	// Field 1: WithdrawalCredentials
	if len(c.WithdrawalCredentials) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.WithdrawalCredentials)
	// Field 2: Amount
	hh.PutUint64(c.Amount)
	// Field 3: Signature
	if len(c.Signature) != 96 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Signature)
	hh.Merkleize(indx)
	return nil
}
func (c *Eth1Data) XXSizeSSZ() int {
	size := 72

	return size
}
func (c *Eth1Data) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *Eth1Data) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: DepositRoot
	if len(c.DepositRoot) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.DepositRoot...)

	// Field 1: DepositCount
	dst = ssz.MarshalUint64(dst, c.DepositCount)

	// Field 2: BlockHash
	if len(c.BlockHash) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.BlockHash...)

	return dst, err
}
func (c *Eth1Data) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 72 {
		return ssz.ErrSize
	}

	s0 := buf[0:32]  // c.DepositRoot
	s1 := buf[32:40] // c.DepositCount
	s2 := buf[40:72] // c.BlockHash

	// Field 0: DepositRoot
	c.DepositRoot = make([]byte, 0, 32)
	c.DepositRoot = append(c.DepositRoot, s0...)

	// Field 1: DepositCount
	c.DepositCount = ssz.UnmarshallUint64(s1)

	// Field 2: BlockHash
	c.BlockHash = make([]byte, 0, 32)
	c.BlockHash = append(c.BlockHash, s2...)
	return err
}
func (c *Eth1Data) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *Eth1Data) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: DepositRoot
	if len(c.DepositRoot) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.DepositRoot)
	// Field 1: DepositCount
	hh.PutUint64(c.DepositCount)
	// Field 2: BlockHash
	if len(c.BlockHash) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.BlockHash)
	hh.Merkleize(indx)
	return nil
}
func (c *IndexedAttestation) XXSizeSSZ() int {
	size := 228
	size += len(c.AttestingIndices) * 8
	return size
}
func (c *IndexedAttestation) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *IndexedAttestation) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 228

	// Field 0: AttestingIndices
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.AttestingIndices) * 8

	// Field 1: Data
	if c.Data == nil {
		c.Data = new(AttestationData)
	}
	if dst, err = c.Data.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 2: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	// Field 0: AttestingIndices
	if len(c.AttestingIndices) > 2048 {
		return nil, ssz.ErrListTooBig
	}
	for _, o := range c.AttestingIndices {
		dst = ssz.MarshalUint64(dst, o)
	}
	return dst, err
}
func (c *IndexedAttestation) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 228 {
		return ssz.ErrSize
	}

	s1 := buf[4:132]   // c.Data
	s2 := buf[132:228] // c.Signature

	v0 := ssz.ReadOffset(buf[0:4]) // c.AttestingIndices
	if v0 < 228 {
		return ssz.ErrInvalidVariableOffset
	}
	if v0 > size {
		return ssz.ErrOffset
	}
	s0 := buf[v0:] // c.AttestingIndices

	// Field 0: AttestingIndices
	{
		if len(s0)%8 != 0 {
			return fmt.Errorf("misaligned bytes: c.AttestingIndices length is %d, which is not a multiple of 8", len(s0))
		}
		numElem := len(s0) / 8
		if numElem > 2048 {
			return fmt.Errorf("ssz-max exceeded: c.AttestingIndices has %d elements, ssz-max is 2048", numElem)
		}
		c.AttestingIndices = make([]uint64, numElem)
		for i := 0; i < numElem; i++ {
			var tmp uint64

			tmpSlice := s0[i*8 : (1+i)*8]
			tmp = ssz.UnmarshallUint64(tmpSlice)
			c.AttestingIndices[i] = tmp
		}
	}

	// Field 1: Data
	c.Data = new(AttestationData)
	if err = c.Data.XXUnmarshalSSZ(s1); err != nil {
		return err
	}

	// Field 2: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, s2...)
	return err
}
func (c *IndexedAttestation) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *IndexedAttestation) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: AttestingIndices
	{
		if len(c.AttestingIndices) > 2048 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.AttestingIndices {
			hh.AppendUint64(o)
		}
		hh.FillUpTo32()
		numItems := uint64(len(c.AttestingIndices))
		hh.MerkleizeWithMixin(subIndx, numItems, ssz.CalculateLimit(2048, numItems, 8))
	}
	// Field 1: Data
	if err := c.Data.XXHashTreeRootWith(hh); err != nil {
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
func (c *PendingAttestation) XXSizeSSZ() int {
	size := 148

	return size
}
func (c *PendingAttestation) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *PendingAttestation) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 148

	// Field 0: AggregationBits
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.AggregationBits) * 1

	// Field 1: Data
	if c.Data == nil {
		c.Data = new(AttestationData)
	}
	if dst, err = c.Data.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 2: InclusionDelay
	dst = ssz.MarshalUint64(dst, uint64(c.InclusionDelay))

	// Field 3: ProposerIndex
	dst = ssz.MarshalUint64(dst, uint64(c.ProposerIndex))

	// Field 0: AggregationBits
	if len(c.AggregationBits) > 2048 {
		return nil, ssz.ErrListTooBig
	}
	dst = append(dst, c.AggregationBits...)
	return dst, err
}
func (c *PendingAttestation) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 148 {
		return ssz.ErrSize
	}

	s1 := buf[4:132]   // c.Data
	s2 := buf[132:140] // c.InclusionDelay
	s3 := buf[140:148] // c.ProposerIndex

	v0 := ssz.ReadOffset(buf[0:4]) // c.AggregationBits
	if v0 < 148 {
		return ssz.ErrInvalidVariableOffset
	}
	if v0 > size {
		return ssz.ErrOffset
	}
	s0 := buf[v0:] // c.AggregationBits

	// Field 0: AggregationBits
	if err = ssz.ValidateBitlist(s0, 2048); err != nil {
		return err
	}
	c.AggregationBits = append([]byte{}, prysmaticlabs_go_bitfield.Bitlist(s0)...)

	// Field 1: Data
	c.Data = new(AttestationData)
	if err = c.Data.XXUnmarshalSSZ(s1); err != nil {
		return err
	}

	// Field 2: InclusionDelay
	c.InclusionDelay = prysmaticlabs_eth2_types.Slot(ssz.UnmarshallUint64(s2))

	// Field 3: ProposerIndex
	c.ProposerIndex = prysmaticlabs_eth2_types.ValidatorIndex(ssz.UnmarshallUint64(s3))
	return err
}
func (c *PendingAttestation) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *PendingAttestation) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: AggregationBits
	if len(c.AggregationBits) == 0 {
		return ssz.ErrEmptyBitlist
	}
	hh.PutBitlist(c.AggregationBits, 2048)
	// Field 1: Data
	if err := c.Data.XXHashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 2: InclusionDelay
	hh.PutUint64(uint64(c.InclusionDelay))
	// Field 3: ProposerIndex
	hh.PutUint64(uint64(c.ProposerIndex))
	hh.Merkleize(indx)
	return nil
}
func (c *ProposerSlashing) XXSizeSSZ() int {
	size := 416

	return size
}
func (c *ProposerSlashing) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *ProposerSlashing) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: SignedHeader_1
	if c.SignedHeader_1 == nil {
		c.SignedHeader_1 = new(SignedBeaconBlockHeader)
	}
	if dst, err = c.SignedHeader_1.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 1: SignedHeader_2
	if c.SignedHeader_2 == nil {
		c.SignedHeader_2 = new(SignedBeaconBlockHeader)
	}
	if dst, err = c.SignedHeader_2.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}

	return dst, err
}
func (c *ProposerSlashing) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 416 {
		return ssz.ErrSize
	}

	s0 := buf[0:208]   // c.SignedHeader_1
	s1 := buf[208:416] // c.SignedHeader_2

	// Field 0: SignedHeader_1
	c.SignedHeader_1 = new(SignedBeaconBlockHeader)
	if err = c.SignedHeader_1.XXUnmarshalSSZ(s0); err != nil {
		return err
	}

	// Field 1: SignedHeader_2
	c.SignedHeader_2 = new(SignedBeaconBlockHeader)
	if err = c.SignedHeader_2.XXUnmarshalSSZ(s1); err != nil {
		return err
	}
	return err
}
func (c *ProposerSlashing) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *ProposerSlashing) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: SignedHeader_1
	if err := c.SignedHeader_1.XXHashTreeRootWith(hh); err != nil {
		return err
	}
	// Field 1: SignedHeader_2
	if err := c.SignedHeader_2.XXHashTreeRootWith(hh); err != nil {
		return err
	}
	hh.Merkleize(indx)
	return nil
}
func (c *SignedAggregateAttestationAndProof) XXSizeSSZ() int {
	size := 100
	if c.Message == nil {
		c.Message = new(AggregateAttestationAndProof)
	}
	size += c.Message.SizeSSZ()
	return size
}
func (c *SignedAggregateAttestationAndProof) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *SignedAggregateAttestationAndProof) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 100

	// Field 0: Message
	if c.Message == nil {
		c.Message = new(AggregateAttestationAndProof)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Message.SizeSSZ()

	// Field 1: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	// Field 0: Message
	if dst, err = c.Message.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}
	return dst, err
}
func (c *SignedAggregateAttestationAndProof) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 100 {
		return ssz.ErrSize
	}

	s1 := buf[4:100] // c.Signature

	v0 := ssz.ReadOffset(buf[0:4]) // c.Message
	if v0 < 100 {
		return ssz.ErrInvalidVariableOffset
	}
	if v0 > size {
		return ssz.ErrOffset
	}
	s0 := buf[v0:] // c.Message

	// Field 0: Message
	c.Message = new(AggregateAttestationAndProof)
	if err = c.Message.XXUnmarshalSSZ(s0); err != nil {
		return err
	}

	// Field 1: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, s1...)
	return err
}
func (c *SignedAggregateAttestationAndProof) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *SignedAggregateAttestationAndProof) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Message
	if err := c.Message.XXHashTreeRootWith(hh); err != nil {
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
func (c *SignedBeaconBlock) XXSizeSSZ() int {
	size := 100
	if c.Block == nil {
		c.Block = new(BeaconBlock)
	}
	size += c.Block.SizeSSZ()
	return size
}
func (c *SignedBeaconBlock) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *SignedBeaconBlock) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 100

	// Field 0: Block
	if c.Block == nil {
		c.Block = new(BeaconBlock)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Block.SizeSSZ()

	// Field 1: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	// Field 0: Block
	if dst, err = c.Block.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}
	return dst, err
}
func (c *SignedBeaconBlock) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 100 {
		return ssz.ErrSize
	}

	s1 := buf[4:100] // c.Signature

	v0 := ssz.ReadOffset(buf[0:4]) // c.Block
	if v0 < 100 {
		return ssz.ErrInvalidVariableOffset
	}
	if v0 > size {
		return ssz.ErrOffset
	}
	s0 := buf[v0:] // c.Block

	// Field 0: Block
	c.Block = new(BeaconBlock)
	if err = c.Block.XXUnmarshalSSZ(s0); err != nil {
		return err
	}

	// Field 1: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, s1...)
	return err
}
func (c *SignedBeaconBlock) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *SignedBeaconBlock) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Block
	if err := c.Block.XXHashTreeRootWith(hh); err != nil {
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
func (c *SignedBeaconBlockV1) XXSizeSSZ() int {
	size := 100
	if c.Block == nil {
		c.Block = new(BeaconBlockV1)
	}
	size += c.Block.SizeSSZ()
	return size
}
func (c *SignedBeaconBlockV1) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *SignedBeaconBlockV1) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 100

	// Field 0: Block
	if c.Block == nil {
		c.Block = new(BeaconBlockV1)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Block.SizeSSZ()

	// Field 1: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	// Field 0: Block
	if dst, err = c.Block.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}
	return dst, err
}
func (c *SignedBeaconBlockV1) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 100 {
		return ssz.ErrSize
	}

	s1 := buf[4:100] // c.Signature

	v0 := ssz.ReadOffset(buf[0:4]) // c.Block
	if v0 < 100 {
		return ssz.ErrInvalidVariableOffset
	}
	if v0 > size {
		return ssz.ErrOffset
	}
	s0 := buf[v0:] // c.Block

	// Field 0: Block
	c.Block = new(BeaconBlockV1)
	if err = c.Block.XXUnmarshalSSZ(s0); err != nil {
		return err
	}

	// Field 1: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, s1...)
	return err
}
func (c *SignedBeaconBlockV1) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *SignedBeaconBlockV1) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Block
	if err := c.Block.XXHashTreeRootWith(hh); err != nil {
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
func (c *SignedBeaconBlockHeader) XXSizeSSZ() int {
	size := 208

	return size
}
func (c *SignedBeaconBlockHeader) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *SignedBeaconBlockHeader) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Message
	if c.Message == nil {
		c.Message = new(BeaconBlockHeader)
	}
	if dst, err = c.Message.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 1: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	return dst, err
}
func (c *SignedBeaconBlockHeader) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 208 {
		return ssz.ErrSize
	}

	s0 := buf[0:112]   // c.Message
	s1 := buf[112:208] // c.Signature

	// Field 0: Message
	c.Message = new(BeaconBlockHeader)
	if err = c.Message.XXUnmarshalSSZ(s0); err != nil {
		return err
	}

	// Field 1: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, s1...)
	return err
}
func (c *SignedBeaconBlockHeader) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *SignedBeaconBlockHeader) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Message
	if err := c.Message.XXHashTreeRootWith(hh); err != nil {
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
func (c *SignedVoluntaryExit) XXSizeSSZ() int {
	size := 112

	return size
}
func (c *SignedVoluntaryExit) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *SignedVoluntaryExit) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Message
	if c.Message == nil {
		c.Message = new(VoluntaryExit)
	}
	if dst, err = c.Message.XXMarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 1: Signature
	if len(c.Signature) != 96 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Signature...)

	return dst, err
}
func (c *SignedVoluntaryExit) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 112 {
		return ssz.ErrSize
	}

	s0 := buf[0:16]   // c.Message
	s1 := buf[16:112] // c.Signature

	// Field 0: Message
	c.Message = new(VoluntaryExit)
	if err = c.Message.XXUnmarshalSSZ(s0); err != nil {
		return err
	}

	// Field 1: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, s1...)
	return err
}
func (c *SignedVoluntaryExit) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *SignedVoluntaryExit) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Message
	if err := c.Message.XXHashTreeRootWith(hh); err != nil {
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
func (c *Validator) XXSizeSSZ() int {
	size := 121

	return size
}
func (c *Validator) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *Validator) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Pubkey
	if len(c.Pubkey) != 48 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Pubkey...)

	// Field 1: WithdrawalCredentials
	if len(c.WithdrawalCredentials) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.WithdrawalCredentials...)

	// Field 2: EffectiveBalance
	dst = ssz.MarshalUint64(dst, c.EffectiveBalance)

	// Field 3: Slashed
	dst = ssz.MarshalBool(dst, c.Slashed)

	// Field 4: ActivationEligibilityEpoch
	dst = ssz.MarshalUint64(dst, uint64(c.ActivationEligibilityEpoch))

	// Field 5: ActivationEpoch
	dst = ssz.MarshalUint64(dst, uint64(c.ActivationEpoch))

	// Field 6: ExitEpoch
	dst = ssz.MarshalUint64(dst, uint64(c.ExitEpoch))

	// Field 7: WithdrawableEpoch
	dst = ssz.MarshalUint64(dst, uint64(c.WithdrawableEpoch))

	return dst, err
}
func (c *Validator) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 121 {
		return ssz.ErrSize
	}

	s0 := buf[0:48]    // c.Pubkey
	s1 := buf[48:80]   // c.WithdrawalCredentials
	s2 := buf[80:88]   // c.EffectiveBalance
	s3 := buf[88:89]   // c.Slashed
	s4 := buf[89:97]   // c.ActivationEligibilityEpoch
	s5 := buf[97:105]  // c.ActivationEpoch
	s6 := buf[105:113] // c.ExitEpoch
	s7 := buf[113:121] // c.WithdrawableEpoch

	// Field 0: Pubkey
	c.Pubkey = make([]byte, 0, 48)
	c.Pubkey = append(c.Pubkey, s0...)

	// Field 1: WithdrawalCredentials
	c.WithdrawalCredentials = make([]byte, 0, 32)
	c.WithdrawalCredentials = append(c.WithdrawalCredentials, s1...)

	// Field 2: EffectiveBalance
	c.EffectiveBalance = ssz.UnmarshallUint64(s2)

	// Field 3: Slashed
	c.Slashed = ssz.UnmarshalBool(s3)

	// Field 4: ActivationEligibilityEpoch
	c.ActivationEligibilityEpoch = prysmaticlabs_eth2_types.Epoch(ssz.UnmarshallUint64(s4))

	// Field 5: ActivationEpoch
	c.ActivationEpoch = prysmaticlabs_eth2_types.Epoch(ssz.UnmarshallUint64(s5))

	// Field 6: ExitEpoch
	c.ExitEpoch = prysmaticlabs_eth2_types.Epoch(ssz.UnmarshallUint64(s6))

	// Field 7: WithdrawableEpoch
	c.WithdrawableEpoch = prysmaticlabs_eth2_types.Epoch(ssz.UnmarshallUint64(s7))
	return err
}
func (c *Validator) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *Validator) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Pubkey
	if len(c.Pubkey) != 48 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Pubkey)
	// Field 1: WithdrawalCredentials
	if len(c.WithdrawalCredentials) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.WithdrawalCredentials)
	// Field 2: EffectiveBalance
	hh.PutUint64(c.EffectiveBalance)
	// Field 3: Slashed
	hh.PutBool(c.Slashed)
	// Field 4: ActivationEligibilityEpoch
	hh.PutUint64(uint64(c.ActivationEligibilityEpoch))
	// Field 5: ActivationEpoch
	hh.PutUint64(uint64(c.ActivationEpoch))
	// Field 6: ExitEpoch
	hh.PutUint64(uint64(c.ExitEpoch))
	// Field 7: WithdrawableEpoch
	hh.PutUint64(uint64(c.WithdrawableEpoch))
	hh.Merkleize(indx)
	return nil
}
func (c *VoluntaryExit) XXSizeSSZ() int {
	size := 16

	return size
}
func (c *VoluntaryExit) XXMarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.XXSizeSSZ())
	return c.XXMarshalSSZTo(buf[:0])
}

func (c *VoluntaryExit) XXMarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Epoch
	dst = ssz.MarshalUint64(dst, uint64(c.Epoch))

	// Field 1: ValidatorIndex
	dst = ssz.MarshalUint64(dst, uint64(c.ValidatorIndex))

	return dst, err
}
func (c *VoluntaryExit) XXUnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 16 {
		return ssz.ErrSize
	}

	s0 := buf[0:8]  // c.Epoch
	s1 := buf[8:16] // c.ValidatorIndex

	// Field 0: Epoch
	c.Epoch = prysmaticlabs_eth2_types.Epoch(ssz.UnmarshallUint64(s0))

	// Field 1: ValidatorIndex
	c.ValidatorIndex = prysmaticlabs_eth2_types.ValidatorIndex(ssz.UnmarshallUint64(s1))
	return err
}
func (c *VoluntaryExit) XXHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.XXHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *VoluntaryExit) XXHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Epoch
	hh.PutUint64(uint64(c.Epoch))
	// Field 1: ValidatorIndex
	hh.PutUint64(uint64(c.ValidatorIndex))
	hh.Merkleize(indx)
	return nil
}
