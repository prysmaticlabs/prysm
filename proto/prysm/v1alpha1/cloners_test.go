package eth

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/testing/assert"
)

func TestCopyETH1Data(t *testing.T) {
	data := genEth1Data()

	got := CopyETH1Data(data)
	if !reflect.DeepEqual(got, data) {
		t.Errorf("CopyETH1Data() = %v, want %v", got, data)
	}
	assert.NotEmpty(t, got, "Copied eth1data has empty fields")
}

func TestCopyPendingAttestation(t *testing.T) {
	pa := genPendingAttestation()

	got := CopyPendingAttestation(pa)
	if !reflect.DeepEqual(got, pa) {
		t.Errorf("CopyPendingAttestation() = %v, want %v", got, pa)
	}
	assert.NotEmpty(t, got, "Copied pending attestation has empty fields")
}

func TestCopyAttestation(t *testing.T) {
	att := genAttestation()

	got := CopyAttestation(att)
	if !reflect.DeepEqual(got, att) {
		t.Errorf("CopyAttestation() = %v, want %v", got, att)
	}
	assert.NotEmpty(t, got, "Copied attestation has empty fields")
}
func TestCopyAttestationData(t *testing.T) {
	att := genAttData()

	got := CopyAttestationData(att)
	if !reflect.DeepEqual(got, att) {
		t.Errorf("CopyAttestationData() = %v, want %v", got, att)
	}
	assert.NotEmpty(t, got, "Copied attestation data has empty fields")
}

func TestCopyCheckpoint(t *testing.T) {
	cp := genCheckpoint()

	got := CopyCheckpoint(cp)
	if !reflect.DeepEqual(got, cp) {
		t.Errorf("CopyCheckpoint() = %v, want %v", got, cp)
	}
	assert.NotEmpty(t, got, "Copied checkpoint has empty fields")
}

func TestCopySignedBeaconBlock(t *testing.T) {
	blk := genSignedBeaconBlock()

	got := CopySignedBeaconBlock(blk)
	if !reflect.DeepEqual(got, blk) {
		t.Errorf("CopySignedBeaconBlock() = %v, want %v", got, blk)
	}
	assert.NotEmpty(t, got, "Copied signed beacon block has empty fields")
}

func TestCopyBeaconBlock(t *testing.T) {
	blk := genBeaconBlock()

	got := CopyBeaconBlock(blk)
	if !reflect.DeepEqual(got, blk) {
		t.Errorf("CopyBeaconBlock() = %v, want %v", got, blk)
	}
	assert.NotEmpty(t, got, "Copied beacon block has empty fields")
}

func TestCopyBeaconBlockBody(t *testing.T) {
	body := genBeaconBlockBody()

	got := CopyBeaconBlockBody(body)
	if !reflect.DeepEqual(got, body) {
		t.Errorf("CopyBeaconBlockBody() = %v, want %v", got, body)
	}
	assert.NotEmpty(t, got, "Copied beacon block body has empty fields")
}

func TestCopySignedBeaconBlockAltair(t *testing.T) {
	sbb := genSignedBeaconBlockAltair()

	got := CopySignedBeaconBlockAltair(sbb)
	if !reflect.DeepEqual(got, sbb) {
		t.Errorf("CopySignedBeaconBlockAltair() = %v, want %v", got, sbb)
	}
	assert.NotEmpty(t, sbb, "Copied signed beacon block altair has empty fields")
}

func TestCopyBeaconBlockAltair(t *testing.T) {
	b := genBeaconBlockAltair()

	got := CopyBeaconBlockAltair(b)
	if !reflect.DeepEqual(got, b) {
		t.Errorf("CopyBeaconBlockAltair() = %v, want %v", got, b)
	}
	assert.NotEmpty(t, b, "Copied beacon block altair has empty fields")
}

func TestCopyBeaconBlockBodyAltair(t *testing.T) {
	bb := genBeaconBlockBodyAltair()

	got := CopyBeaconBlockBodyAltair(bb)
	if !reflect.DeepEqual(got, bb) {
		t.Errorf("CopyBeaconBlockBodyAltair() = %v, want %v", got, bb)
	}
	assert.NotEmpty(t, bb, "Copied beacon block body altair has empty fields")
}

func TestCopyProposerSlashings(t *testing.T) {
	ps := genProposerSlashings(10)

	got := CopyProposerSlashings(ps)
	if !reflect.DeepEqual(got, ps) {
		t.Errorf("CopyProposerSlashings() = %v, want %v", got, ps)
	}
	assert.NotEmpty(t, got, "Copied proposer slashings have empty fields")
}

func TestCopyProposerSlashing(t *testing.T) {
	ps := genProposerSlashing()

	got := CopyProposerSlashing(ps)
	if !reflect.DeepEqual(got, ps) {
		t.Errorf("CopyProposerSlashing() = %v, want %v", got, ps)
	}
	assert.NotEmpty(t, got, "Copied proposer slashing has empty fields")
}

func TestCopySignedBeaconBlockHeader(t *testing.T) {
	sbh := genSignedBeaconBlockHeader()

	got := CopySignedBeaconBlockHeader(sbh)
	if !reflect.DeepEqual(got, sbh) {
		t.Errorf("CopySignedBeaconBlockHeader() = %v, want %v", got, sbh)
	}
	assert.NotEmpty(t, got, "Copied signed beacon block header has empty fields")
}

func TestCopyBeaconBlockHeader(t *testing.T) {
	bh := genBeaconBlockHeader()

	got := CopyBeaconBlockHeader(bh)
	if !reflect.DeepEqual(got, bh) {
		t.Errorf("CopyBeaconBlockHeader() = %v, want %v", got, bh)
	}
	assert.NotEmpty(t, got, "Copied beacon block header has empty fields")
}

func TestCopyAttesterSlashings(t *testing.T) {
	as := genAttesterSlashings(10)

	got := CopyAttesterSlashings(as)
	if !reflect.DeepEqual(got, as) {
		t.Errorf("CopyAttesterSlashings() = %v, want %v", got, as)
	}
	assert.NotEmpty(t, got, "Copied attester slashings have empty fields")
}

func TestCopyIndexedAttestation(t *testing.T) {
	ia := genIndexedAttestation()

	got := CopyIndexedAttestation(ia)
	if !reflect.DeepEqual(got, ia) {
		t.Errorf("CopyIndexedAttestation() = %v, want %v", got, ia)
	}
	assert.NotEmpty(t, got, "Copied indexed attestation has empty fields")
}

func TestCopyAttestations(t *testing.T) {
	atts := genAttestations(10)

	got := CopyAttestations(atts)
	if !reflect.DeepEqual(got, atts) {
		t.Errorf("CopyAttestations() = %v, want %v", got, atts)
	}
	assert.NotEmpty(t, got, "Copied attestations have empty fields")
}

func TestCopyDeposits(t *testing.T) {
	d := genDeposits(10)

	got := CopyDeposits(d)
	if !reflect.DeepEqual(got, d) {
		t.Errorf("CopyDeposits() = %v, want %v", got, d)
	}
	assert.NotEmpty(t, got, "Copied deposits have empty fields")
}

func TestCopyDeposit(t *testing.T) {
	d := genDeposit()

	got := CopyDeposit(d)
	if !reflect.DeepEqual(got, d) {
		t.Errorf("CopyDeposit() = %v, want %v", got, d)
	}
	assert.NotEmpty(t, got, "Copied deposit has empty fields")
}

func TestCopyDepositData(t *testing.T) {
	dd := genDepositData()

	got := CopyDepositData(dd)
	if !reflect.DeepEqual(got, dd) {
		t.Errorf("CopyDepositData() = %v, want %v", got, dd)
	}
	assert.NotEmpty(t, got, "Copied deposit data has empty fields")
}

func TestCopySignedVoluntaryExits(t *testing.T) {
	sv := genSignedVoluntaryExits(10)

	got := CopySignedVoluntaryExits(sv)
	if !reflect.DeepEqual(got, sv) {
		t.Errorf("CopySignedVoluntaryExits() = %v, want %v", got, sv)
	}
	assert.NotEmpty(t, got, "Copied signed voluntary exits have empty fields")
}

func TestCopySignedVoluntaryExit(t *testing.T) {
	sv := genSignedVoluntaryExit()

	got := CopySignedVoluntaryExit(sv)
	if !reflect.DeepEqual(got, sv) {
		t.Errorf("CopySignedVoluntaryExit() = %v, want %v", got, sv)
	}
	assert.NotEmpty(t, got, "Copied signed voluntary exit has empty fields")
}

func TestCopyValidator(t *testing.T) {
	v := genValidator()

	got := CopyValidator(v)
	if !reflect.DeepEqual(got, v) {
		t.Errorf("CopyValidator() = %v, want %v", got, v)
	}
	assert.NotEmpty(t, got, "Copied validator has empty fields")
}

func TestCopySyncCommitteeMessage(t *testing.T) {
	scm := genSyncCommitteeMessage()

	got := CopySyncCommitteeMessage(scm)
	if !reflect.DeepEqual(got, scm) {
		t.Errorf("CopySyncCommitteeMessage() = %v, want %v", got, scm)
	}
	assert.NotEmpty(t, got, "Copied sync committee message has empty fields")
}

func TestCopySyncCommitteeContribution(t *testing.T) {
	scc := genSyncCommitteeContribution()

	got := CopySyncCommitteeContribution(scc)
	if !reflect.DeepEqual(got, scc) {
		t.Errorf("CopySyncCommitteeContribution() = %v, want %v", got, scc)
	}
	assert.NotEmpty(t, got, "Copied sync committee contribution has empty fields")
}

func TestCopySyncAggregate(t *testing.T) {
	sa := genSyncAggregate()

	got := CopySyncAggregate(sa)
	if !reflect.DeepEqual(got, sa) {
		t.Errorf("CopySyncAggregate() = %v, want %v", got, sa)
	}
	assert.NotEmpty(t, got, "Copied sync aggregate has empty fields")
}

func TestCopyPendingAttestationSlice(t *testing.T) {
	tests := []struct {
		name  string
		input []*PendingAttestation
	}{
		{
			name:  "nil",
			input: nil,
		},
		{
			name:  "empty",
			input: []*PendingAttestation{},
		},
		{
			name: "correct copy",
			input: []*PendingAttestation{
				genPendingAttestation(),
				genPendingAttestation(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyPendingAttestationSlice(tt.input); !reflect.DeepEqual(got, tt.input) {
				t.Errorf("CopyPendingAttestationSlice() = %v, want %v", got, tt.input)
			}
		})
	}
}

func bytes() []byte {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 32; i++ {
		if b[i] == 0x00 {
			b[i] = uint8(rand.Int())
		}
	}
	return b
}

func genAttestation() *Attestation {
	return &Attestation{
		AggregationBits: bytes(),
		Data:            genAttData(),
		Signature:       bytes(),
	}
}

func genAttestations(num int) []*Attestation {
	atts := make([]*Attestation, num)
	for i := 0; i < num; i++ {
		atts[i] = genAttestation()
	}
	return atts
}

func genAttData() *AttestationData {
	return &AttestationData{
		Slot:            1,
		CommitteeIndex:  2,
		BeaconBlockRoot: bytes(),
		Source:          genCheckpoint(),
		Target:          genCheckpoint(),
	}
}

func genCheckpoint() *Checkpoint {
	return &Checkpoint{
		Epoch: 1,
		Root:  bytes(),
	}
}

func genEth1Data() *Eth1Data {
	return &Eth1Data{
		DepositRoot:  bytes(),
		DepositCount: 4,
		BlockHash:    bytes(),
	}
}

func genPendingAttestation() *PendingAttestation {
	return &PendingAttestation{
		AggregationBits: bytes(),
		Data:            genAttData(),
		InclusionDelay:  3,
		ProposerIndex:   5,
	}
}

func genSignedBeaconBlock() *SignedBeaconBlock {
	return &SignedBeaconBlock{
		Block:     genBeaconBlock(),
		Signature: bytes(),
	}
}

func genBeaconBlock() *BeaconBlock {
	return &BeaconBlock{
		Slot:          4,
		ProposerIndex: 5,
		ParentRoot:    bytes(),
		StateRoot:     bytes(),
		Body:          genBeaconBlockBody(),
	}
}

func genBeaconBlockBody() *BeaconBlockBody {
	return &BeaconBlockBody{
		RandaoReveal:      bytes(),
		Eth1Data:          genEth1Data(),
		Graffiti:          bytes(),
		ProposerSlashings: genProposerSlashings(5),
		AttesterSlashings: genAttesterSlashings(5),
		Attestations:      genAttestations(5),
		Deposits:          genDeposits(5),
		VoluntaryExits:    genSignedVoluntaryExits(5),
	}
}

func genProposerSlashing() *ProposerSlashing {
	return &ProposerSlashing{
		Header_1: genSignedBeaconBlockHeader(),
		Header_2: genSignedBeaconBlockHeader(),
	}
}

func genProposerSlashings(num int) []*ProposerSlashing {
	ps := make([]*ProposerSlashing, num)
	for i := 0; i < num; i++ {
		ps[i] = genProposerSlashing()
	}
	return ps
}

func genAttesterSlashing() *AttesterSlashing {
	return &AttesterSlashing{
		Attestation_1: genIndexedAttestation(),
		Attestation_2: genIndexedAttestation(),
	}
}

func genIndexedAttestation() *IndexedAttestation {
	return &IndexedAttestation{
		AttestingIndices: []uint64{1, 2, 3},
		Data:             genAttData(),
		Signature:        bytes(),
	}
}

func genAttesterSlashings(num int) []*AttesterSlashing {
	as := make([]*AttesterSlashing, num)
	for i := 0; i < num; i++ {
		as[i] = genAttesterSlashing()
	}
	return as
}

func genBeaconBlockHeader() *BeaconBlockHeader {
	return &BeaconBlockHeader{
		Slot:          10,
		ProposerIndex: 15,
		ParentRoot:    bytes(),
		StateRoot:     bytes(),
		BodyRoot:      bytes(),
	}
}

func genSignedBeaconBlockHeader() *SignedBeaconBlockHeader {
	return &SignedBeaconBlockHeader{
		Header:    genBeaconBlockHeader(),
		Signature: bytes(),
	}
}

func genDepositData() *Deposit_Data {
	return &Deposit_Data{
		PublicKey:             bytes(),
		WithdrawalCredentials: bytes(),
		Amount:                20000,
		Signature:             bytes(),
	}
}

func genDeposit() *Deposit {
	return &Deposit{
		Data:  genDepositData(),
		Proof: [][]byte{bytes(), bytes(), bytes(), bytes()},
	}
}

func genDeposits(num int) []*Deposit {
	d := make([]*Deposit, num)
	for i := 0; i < num; i++ {
		d[i] = genDeposit()
	}
	return d
}

func genVoluntaryExit() *VoluntaryExit {
	return &VoluntaryExit{
		Epoch:          5432,
		ValidatorIndex: 888888,
	}
}

func genSignedVoluntaryExit() *SignedVoluntaryExit {
	return &SignedVoluntaryExit{
		Exit:      genVoluntaryExit(),
		Signature: bytes(),
	}
}

func genSignedVoluntaryExits(num int) []*SignedVoluntaryExit {
	sv := make([]*SignedVoluntaryExit, num)
	for i := 0; i < num; i++ {
		sv[i] = genSignedVoluntaryExit()
	}
	return sv
}

func genValidator() *Validator {
	return &Validator{
		PublicKey:                  bytes(),
		WithdrawalCredentials:      bytes(),
		EffectiveBalance:           12345,
		Slashed:                    true,
		ActivationEligibilityEpoch: 14322,
		ActivationEpoch:            14325,
		ExitEpoch:                  23425,
		WithdrawableEpoch:          30000,
	}
}

func genSyncCommitteeContribution() *SyncCommitteeContribution {
	return &SyncCommitteeContribution{
		Slot:              12333,
		BlockRoot:         bytes(),
		SubcommitteeIndex: 4444,
		AggregationBits:   bytes(),
		Signature:         bytes(),
	}
}

func genSyncAggregate() *SyncAggregate {
	return &SyncAggregate{
		SyncCommitteeBits:      bytes(),
		SyncCommitteeSignature: bytes(),
	}
}

func genBeaconBlockBodyAltair() *BeaconBlockBodyAltair {
	return &BeaconBlockBodyAltair{
		RandaoReveal:      bytes(),
		Eth1Data:          genEth1Data(),
		Graffiti:          bytes(),
		ProposerSlashings: genProposerSlashings(5),
		AttesterSlashings: genAttesterSlashings(5),
		Attestations:      genAttestations(10),
		Deposits:          genDeposits(5),
		VoluntaryExits:    genSignedVoluntaryExits(12),
		SyncAggregate:     genSyncAggregate(),
	}
}

func genBeaconBlockAltair() *BeaconBlockAltair {
	return &BeaconBlockAltair{
		Slot:          123455,
		ProposerIndex: 55433,
		ParentRoot:    bytes(),
		StateRoot:     bytes(),
		Body:          genBeaconBlockBodyAltair(),
	}
}

func genSignedBeaconBlockAltair() *SignedBeaconBlockAltair {
	return &SignedBeaconBlockAltair{
		Block:     genBeaconBlockAltair(),
		Signature: bytes(),
	}
}

func genSyncCommitteeMessage() *SyncCommitteeMessage {
	return &SyncCommitteeMessage{
		Slot:           424555,
		BlockRoot:      bytes(),
		ValidatorIndex: 5443,
		Signature:      bytes(),
	}
}
