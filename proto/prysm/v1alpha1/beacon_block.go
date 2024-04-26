package eth

//TODO: consider replacing this entirely... indexattestation doesn't need to be a protobuf

func (a *IndexedAttestation) GetAttestingIndicesVal() []uint64 {
	return a.AttestingIndices
}
func (a *IndexedAttestation) SetAttestingIndicesVal(indices []uint64) {
	a.AttestingIndices = indices
}

func (a *IndexedAttestation) SetData(data *AttestationData) {
	a.Data = data
}

func (a *IndexedAttestation) SetSignature(sig []byte) {
	a.Signature = sig
}

func (a *IndexedAttestationElectra) GetAttestingIndicesVal() []uint64 {
	return a.AttestingIndices
}
func (a *IndexedAttestationElectra) SetAttestingIndicesVal(indices []uint64) {
	a.AttestingIndices = indices
}

func (a *IndexedAttestationElectra) SetData(data *AttestationData) {
	a.Data = data
}

func (a *IndexedAttestationElectra) SetSignature(sig []byte) {
	a.Signature = sig
}
