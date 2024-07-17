package eth

import (
	"fmt"
	ssz "github.com/prysmaticlabs/fastssz"
	go_bitfield "github.com/prysmaticlabs/go-bitfield"
	v1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
)

func (c *BLSToExecutionChange) SizeSSZ() int {
	size := 76

	return size
}

func (c *BLSToExecutionChange) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *BLSToExecutionChange) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: ValidatorIndex
	if dst, err = c.ValidatorIndex.MarshalSSZTo(dst); err != nil {
		return nil, err
	}

	// Field 1: FromBlsPubkey
	if len(c.FromBlsPubkey) != 48 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.FromBlsPubkey...)

	// Field 2: ToExecutionAddress
	if len(c.ToExecutionAddress) != 20 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.ToExecutionAddress...)

	return dst, err
}

func (c *BLSToExecutionChange) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 76 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:8]   // c.ValidatorIndex
	sszSlice1 := buf[8:56]  // c.FromBlsPubkey
	sszSlice2 := buf[56:76] // c.ToExecutionAddress

	// Field 0: ValidatorIndex
	if err = c.ValidatorIndex.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: FromBlsPubkey
	c.FromBlsPubkey = make([]byte, 0, 48)
	c.FromBlsPubkey = append(c.FromBlsPubkey, sszSlice1...)

	// Field 2: ToExecutionAddress
	c.ToExecutionAddress = make([]byte, 0, 20)
	c.ToExecutionAddress = append(c.ToExecutionAddress, sszSlice2...)
	return err
}

func (c *BLSToExecutionChange) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BLSToExecutionChange) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: ValidatorIndex
	if hash, err := c.ValidatorIndex.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 1: FromBlsPubkey
	if len(c.FromBlsPubkey) != 48 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.FromBlsPubkey)
	// Field 2: ToExecutionAddress
	if len(c.ToExecutionAddress) != 20 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.ToExecutionAddress)
	hh.Merkleize(indx)
	return nil
}

func (c *BeaconBlockBodyCapella) SizeSSZ() int {
	size := 388
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
		c.ExecutionPayload = new(v1.ExecutionPayloadCapella)
	}
	size += c.ExecutionPayload.SizeSSZ()
	size += len(c.BlsToExecutionChanges) * 172
	return size
}

func (c *BeaconBlockBodyCapella) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *BeaconBlockBodyCapella) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 388

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
		c.ExecutionPayload = new(v1.ExecutionPayloadCapella)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.ExecutionPayload.SizeSSZ()

	// Field 10: BlsToExecutionChanges
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.BlsToExecutionChanges) * 172

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
		if dst, err = o.MarshalSSZTo(dst); err != nil {
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
	return dst, err
}

func (c *BeaconBlockBodyCapella) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 388 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:96]    // c.RandaoReveal
	sszSlice1 := buf[96:168]  // c.Eth1Data
	sszSlice2 := buf[168:200] // c.Graffiti
	sszSlice8 := buf[220:380] // c.SyncAggregate

	sszVarOffset3 := ssz.ReadOffset(buf[200:204]) // c.ProposerSlashings
	if sszVarOffset3 < 388 {
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
	sszSlice3 := buf[sszVarOffset3:sszVarOffset4]  // c.ProposerSlashings
	sszSlice4 := buf[sszVarOffset4:sszVarOffset5]  // c.AttesterSlashings
	sszSlice5 := buf[sszVarOffset5:sszVarOffset6]  // c.Attestations
	sszSlice6 := buf[sszVarOffset6:sszVarOffset7]  // c.Deposits
	sszSlice7 := buf[sszVarOffset7:sszVarOffset9]  // c.VoluntaryExits
	sszSlice9 := buf[sszVarOffset9:sszVarOffset10] // c.ExecutionPayload
	sszSlice10 := buf[sszVarOffset10:]             // c.BlsToExecutionChanges

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
			if listLen > 2 {
				return fmt.Errorf("ssz-max exceeded: c.AttesterSlashings has %d elements, ssz-max is 2", listLen)
			}
			listOffsets := make([]uint64, listLen)
			for i := 0; uint64(i) < listLen; i++ {
				listOffsets[i] = ssz.ReadOffset(sszSlice4[i*4 : (i+1)*4])
			}
			c.AttesterSlashings = make([]*AttesterSlashing, len(listOffsets))
			for i := 0; i < len(listOffsets); i++ {
				var tmp *AttesterSlashing
				tmp = new(AttesterSlashing)
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
			if listLen > 128 {
				return fmt.Errorf("ssz-max exceeded: c.Attestations has %d elements, ssz-max is 128", listLen)
			}
			listOffsets := make([]uint64, listLen)
			for i := 0; uint64(i) < listLen; i++ {
				listOffsets[i] = ssz.ReadOffset(sszSlice5[i*4 : (i+1)*4])
			}
			c.Attestations = make([]*Attestation, len(listOffsets))
			for i := 0; i < len(listOffsets); i++ {
				var tmp *Attestation
				tmp = new(Attestation)
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
	c.ExecutionPayload = new(v1.ExecutionPayloadCapella)
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
	return err
}

func (c *BeaconBlockBodyCapella) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BeaconBlockBodyCapella) HashTreeRootWith(hh *ssz.Hasher) (err error) {
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
		if len(c.AttesterSlashings) > 2 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.AttesterSlashings {
			if err := o.HashTreeRootWith(hh); err != nil {
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
			if err := o.HashTreeRootWith(hh); err != nil {
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
	hh.Merkleize(indx)
	return nil
}

func (c *BeaconBlockCapella) SizeSSZ() int {
	size := 84
	if c.Body == nil {
		c.Body = new(BeaconBlockBodyCapella)
	}
	size += c.Body.SizeSSZ()
	return size
}

func (c *BeaconBlockCapella) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *BeaconBlockCapella) MarshalSSZTo(dst []byte) ([]byte, error) {
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
		c.Body = new(BeaconBlockBodyCapella)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Body.SizeSSZ()

	// Field 4: Body
	if dst, err = c.Body.MarshalSSZTo(dst); err != nil {
		return nil, err
	}
	return dst, err
}

func (c *BeaconBlockCapella) UnmarshalSSZ(buf []byte) error {
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
	c.Body = new(BeaconBlockBodyCapella)
	if err = c.Body.UnmarshalSSZ(sszSlice4); err != nil {
		return err
	}
	return err
}

func (c *BeaconBlockCapella) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BeaconBlockCapella) HashTreeRootWith(hh *ssz.Hasher) (err error) {
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

func (c *BeaconStateCapella) SizeSSZ() int {
	size := 2736653
	size += len(c.HistoricalRoots) * 32
	size += len(c.Eth1DataVotes) * 72
	size += len(c.Validators) * 121
	size += len(c.Balances) * 8
	size += len(c.PreviousEpochParticipation)
	size += len(c.CurrentEpochParticipation)
	size += len(c.InactivityScores) * 8
	if c.LatestExecutionPayloadHeader == nil {
		c.LatestExecutionPayloadHeader = new(v1.ExecutionPayloadHeaderCapella)
	}
	size += c.LatestExecutionPayloadHeader.SizeSSZ()
	size += len(c.HistoricalSummaries) * 64
	return size
}

func (c *BeaconStateCapella) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *BeaconStateCapella) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 2736653

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
		c.LatestExecutionPayloadHeader = new(v1.ExecutionPayloadHeaderCapella)
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
	return dst, err
}

func (c *BeaconStateCapella) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 2736653 {
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

	sszVarOffset7 := ssz.ReadOffset(buf[524464:524468]) // c.HistoricalRoots
	if sszVarOffset7 < 2736653 {
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
	sszSlice7 := buf[sszVarOffset7:sszVarOffset9]    // c.HistoricalRoots
	sszSlice9 := buf[sszVarOffset9:sszVarOffset11]   // c.Eth1DataVotes
	sszSlice11 := buf[sszVarOffset11:sszVarOffset12] // c.Validators
	sszSlice12 := buf[sszVarOffset12:sszVarOffset15] // c.Balances
	sszSlice15 := buf[sszVarOffset15:sszVarOffset16] // c.PreviousEpochParticipation
	sszSlice16 := buf[sszVarOffset16:sszVarOffset21] // c.CurrentEpochParticipation
	sszSlice21 := buf[sszVarOffset21:sszVarOffset24] // c.InactivityScores
	sszSlice24 := buf[sszVarOffset24:sszVarOffset27] // c.LatestExecutionPayloadHeader
	sszSlice27 := buf[sszVarOffset27:]               // c.HistoricalSummaries

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
	c.LatestExecutionPayloadHeader = new(v1.ExecutionPayloadHeaderCapella)
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
	return err
}

func (c *BeaconStateCapella) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BeaconStateCapella) HashTreeRootWith(hh *ssz.Hasher) (err error) {
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
	hh.Merkleize(indx)
	return nil
}

func (c *BlindedBeaconBlockBodyCapella) SizeSSZ() int {
	size := 388
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
		c.ExecutionPayloadHeader = new(v1.ExecutionPayloadHeaderCapella)
	}
	size += c.ExecutionPayloadHeader.SizeSSZ()
	size += len(c.BlsToExecutionChanges) * 172
	return size
}

func (c *BlindedBeaconBlockBodyCapella) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *BlindedBeaconBlockBodyCapella) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 388

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
		c.ExecutionPayloadHeader = new(v1.ExecutionPayloadHeaderCapella)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.ExecutionPayloadHeader.SizeSSZ()

	// Field 10: BlsToExecutionChanges
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.BlsToExecutionChanges) * 172

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
		if dst, err = o.MarshalSSZTo(dst); err != nil {
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
	return dst, err
}

func (c *BlindedBeaconBlockBodyCapella) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 388 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:96]    // c.RandaoReveal
	sszSlice1 := buf[96:168]  // c.Eth1Data
	sszSlice2 := buf[168:200] // c.Graffiti
	sszSlice8 := buf[220:380] // c.SyncAggregate

	sszVarOffset3 := ssz.ReadOffset(buf[200:204]) // c.ProposerSlashings
	if sszVarOffset3 < 388 {
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
	sszSlice3 := buf[sszVarOffset3:sszVarOffset4]  // c.ProposerSlashings
	sszSlice4 := buf[sszVarOffset4:sszVarOffset5]  // c.AttesterSlashings
	sszSlice5 := buf[sszVarOffset5:sszVarOffset6]  // c.Attestations
	sszSlice6 := buf[sszVarOffset6:sszVarOffset7]  // c.Deposits
	sszSlice7 := buf[sszVarOffset7:sszVarOffset9]  // c.VoluntaryExits
	sszSlice9 := buf[sszVarOffset9:sszVarOffset10] // c.ExecutionPayloadHeader
	sszSlice10 := buf[sszVarOffset10:]             // c.BlsToExecutionChanges

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
			if listLen > 2 {
				return fmt.Errorf("ssz-max exceeded: c.AttesterSlashings has %d elements, ssz-max is 2", listLen)
			}
			listOffsets := make([]uint64, listLen)
			for i := 0; uint64(i) < listLen; i++ {
				listOffsets[i] = ssz.ReadOffset(sszSlice4[i*4 : (i+1)*4])
			}
			c.AttesterSlashings = make([]*AttesterSlashing, len(listOffsets))
			for i := 0; i < len(listOffsets); i++ {
				var tmp *AttesterSlashing
				tmp = new(AttesterSlashing)
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
			if listLen > 128 {
				return fmt.Errorf("ssz-max exceeded: c.Attestations has %d elements, ssz-max is 128", listLen)
			}
			listOffsets := make([]uint64, listLen)
			for i := 0; uint64(i) < listLen; i++ {
				listOffsets[i] = ssz.ReadOffset(sszSlice5[i*4 : (i+1)*4])
			}
			c.Attestations = make([]*Attestation, len(listOffsets))
			for i := 0; i < len(listOffsets); i++ {
				var tmp *Attestation
				tmp = new(Attestation)
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
	c.ExecutionPayloadHeader = new(v1.ExecutionPayloadHeaderCapella)
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
	return err
}

func (c *BlindedBeaconBlockBodyCapella) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BlindedBeaconBlockBodyCapella) HashTreeRootWith(hh *ssz.Hasher) (err error) {
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
		if len(c.AttesterSlashings) > 2 {
			return ssz.ErrListTooBig
		}
		subIndx := hh.Index()
		for _, o := range c.AttesterSlashings {
			if err := o.HashTreeRootWith(hh); err != nil {
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
			if err := o.HashTreeRootWith(hh); err != nil {
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
	hh.Merkleize(indx)
	return nil
}

func (c *BlindedBeaconBlockCapella) SizeSSZ() int {
	size := 84
	if c.Body == nil {
		c.Body = new(BlindedBeaconBlockBodyCapella)
	}
	size += c.Body.SizeSSZ()
	return size
}

func (c *BlindedBeaconBlockCapella) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *BlindedBeaconBlockCapella) MarshalSSZTo(dst []byte) ([]byte, error) {
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
		c.Body = new(BlindedBeaconBlockBodyCapella)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Body.SizeSSZ()

	// Field 4: Body
	if dst, err = c.Body.MarshalSSZTo(dst); err != nil {
		return nil, err
	}
	return dst, err
}

func (c *BlindedBeaconBlockCapella) UnmarshalSSZ(buf []byte) error {
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
	c.Body = new(BlindedBeaconBlockBodyCapella)
	if err = c.Body.UnmarshalSSZ(sszSlice4); err != nil {
		return err
	}
	return err
}

func (c *BlindedBeaconBlockCapella) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BlindedBeaconBlockCapella) HashTreeRootWith(hh *ssz.Hasher) (err error) {
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

func (c *BuilderBidCapella) SizeSSZ() int {
	size := 84
	if c.Header == nil {
		c.Header = new(v1.ExecutionPayloadHeaderCapella)
	}
	size += c.Header.SizeSSZ()
	return size
}

func (c *BuilderBidCapella) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *BuilderBidCapella) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 84

	// Field 0: Header
	if c.Header == nil {
		c.Header = new(v1.ExecutionPayloadHeaderCapella)
	}
	dst = ssz.WriteOffset(dst, offset)
	offset += c.Header.SizeSSZ()

	// Field 1: Value
	if len(c.Value) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Value...)

	// Field 2: Pubkey
	if len(c.Pubkey) != 48 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.Pubkey...)

	// Field 0: Header
	if dst, err = c.Header.MarshalSSZTo(dst); err != nil {
		return nil, err
	}
	return dst, err
}

func (c *BuilderBidCapella) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 84 {
		return ssz.ErrSize
	}

	sszSlice1 := buf[4:36]  // c.Value
	sszSlice2 := buf[36:84] // c.Pubkey

	sszVarOffset0 := ssz.ReadOffset(buf[0:4]) // c.Header
	if sszVarOffset0 < 84 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset0 > size {
		return ssz.ErrOffset
	}
	sszSlice0 := buf[sszVarOffset0:] // c.Header

	// Field 0: Header
	c.Header = new(v1.ExecutionPayloadHeaderCapella)
	if err = c.Header.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: Value
	c.Value = make([]byte, 0, 32)
	c.Value = append(c.Value, sszSlice1...)

	// Field 2: Pubkey
	c.Pubkey = make([]byte, 0, 48)
	c.Pubkey = append(c.Pubkey, sszSlice2...)
	return err
}

func (c *BuilderBidCapella) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *BuilderBidCapella) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: Header
	if hash, err := c.Header.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}
	// Field 1: Value
	if len(c.Value) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Value)
	// Field 2: Pubkey
	if len(c.Pubkey) != 48 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.Pubkey)
	hh.Merkleize(indx)
	return nil
}

func (c *HistoricalSummary) SizeSSZ() int {
	size := 64

	return size
}

func (c *HistoricalSummary) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *HistoricalSummary) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: BlockSummaryRoot
	if len(c.BlockSummaryRoot) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.BlockSummaryRoot...)

	// Field 1: StateSummaryRoot
	if len(c.StateSummaryRoot) != 32 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, c.StateSummaryRoot...)

	return dst, err
}

func (c *HistoricalSummary) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 64 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:32]  // c.BlockSummaryRoot
	sszSlice1 := buf[32:64] // c.StateSummaryRoot

	// Field 0: BlockSummaryRoot
	c.BlockSummaryRoot = make([]byte, 0, 32)
	c.BlockSummaryRoot = append(c.BlockSummaryRoot, sszSlice0...)

	// Field 1: StateSummaryRoot
	c.StateSummaryRoot = make([]byte, 0, 32)
	c.StateSummaryRoot = append(c.StateSummaryRoot, sszSlice1...)
	return err
}

func (c *HistoricalSummary) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *HistoricalSummary) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: BlockSummaryRoot
	if len(c.BlockSummaryRoot) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.BlockSummaryRoot)
	// Field 1: StateSummaryRoot
	if len(c.StateSummaryRoot) != 32 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes(c.StateSummaryRoot)
	hh.Merkleize(indx)
	return nil
}

func (c *SignedBLSToExecutionChange) SizeSSZ() int {
	size := 172

	return size
}

func (c *SignedBLSToExecutionChange) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *SignedBLSToExecutionChange) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: Message
	if c.Message == nil {
		c.Message = new(BLSToExecutionChange)
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

func (c *SignedBLSToExecutionChange) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 172 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:76]   // c.Message
	sszSlice1 := buf[76:172] // c.Signature

	// Field 0: Message
	c.Message = new(BLSToExecutionChange)
	if err = c.Message.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, sszSlice1...)
	return err
}

func (c *SignedBLSToExecutionChange) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *SignedBLSToExecutionChange) HashTreeRootWith(hh *ssz.Hasher) (err error) {
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

func (c *SignedBeaconBlockCapella) SizeSSZ() int {
	size := 100
	if c.Block == nil {
		c.Block = new(BeaconBlockCapella)
	}
	size += c.Block.SizeSSZ()
	return size
}

func (c *SignedBeaconBlockCapella) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *SignedBeaconBlockCapella) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 100

	// Field 0: Block
	if c.Block == nil {
		c.Block = new(BeaconBlockCapella)
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

func (c *SignedBeaconBlockCapella) UnmarshalSSZ(buf []byte) error {
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
	c.Block = new(BeaconBlockCapella)
	if err = c.Block.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, sszSlice1...)
	return err
}

func (c *SignedBeaconBlockCapella) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *SignedBeaconBlockCapella) HashTreeRootWith(hh *ssz.Hasher) (err error) {
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

func (c *SignedBlindedBeaconBlockCapella) SizeSSZ() int {
	size := 100
	if c.Block == nil {
		c.Block = new(BlindedBeaconBlockCapella)
	}
	size += c.Block.SizeSSZ()
	return size
}

func (c *SignedBlindedBeaconBlockCapella) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *SignedBlindedBeaconBlockCapella) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 100

	// Field 0: Block
	if c.Block == nil {
		c.Block = new(BlindedBeaconBlockCapella)
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

func (c *SignedBlindedBeaconBlockCapella) UnmarshalSSZ(buf []byte) error {
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
	c.Block = new(BlindedBeaconBlockCapella)
	if err = c.Block.UnmarshalSSZ(sszSlice0); err != nil {
		return err
	}

	// Field 1: Signature
	c.Signature = make([]byte, 0, 96)
	c.Signature = append(c.Signature, sszSlice1...)
	return err
}

func (c *SignedBlindedBeaconBlockCapella) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *SignedBlindedBeaconBlockCapella) HashTreeRootWith(hh *ssz.Hasher) (err error) {
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
