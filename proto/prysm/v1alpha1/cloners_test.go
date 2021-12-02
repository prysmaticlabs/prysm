package eth_test

import (
	"math/rand"
	"reflect"
	"testing"

	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
)

func TestCopyETH1Data(t *testing.T) {
	data := genEth1Data()

	got := v1alpha1.CopyETH1Data(data)
	if !reflect.DeepEqual(got, data) {
		t.Errorf("CopyETH1Data() = %v, want %v", got, data)
	}
	assert.NotEmpty(t, got, "Copied eth1data has empty fields")
}

func TestCopyPendingAttestation(t *testing.T) {
	pa := genPendingAttestation()

	got := v1alpha1.CopyPendingAttestation(pa)
	if !reflect.DeepEqual(got, pa) {
		t.Errorf("CopyPendingAttestation() = %v, want %v", got, pa)
	}
	assert.NotEmpty(t, got, "Copied pending attestation has empty fields")
}

func TestCopyAttestation(t *testing.T) {
	att := genAttestation()

	got := v1alpha1.CopyAttestation(att)
	if !reflect.DeepEqual(got, att) {
		t.Errorf("CopyAttestation() = %v, want %v", got, att)
	}
	assert.NotEmpty(t, got, "Copied attestation has empty fields")
}
func TestCopyAttestationData(t *testing.T) {
	att := genAttData()

	got := v1alpha1.CopyAttestationData(att)
	if !reflect.DeepEqual(got, att) {
		t.Errorf("CopyAttestationData() = %v, want %v", got, att)
	}
	assert.NotEmpty(t, got, "Copied attestation data has empty fields")
}

func TestCopyCheckpoint(t *testing.T) {
	cp := genCheckpoint()

	got := v1alpha1.CopyCheckpoint(cp)
	if !reflect.DeepEqual(got, cp) {
		t.Errorf("CopyCheckpoint() = %v, want %v", got, cp)
	}
	assert.NotEmpty(t, got, "Copied checkpoint has empty fields")
}

func TestCopySignedBeaconBlock(t *testing.T) {
	blk := genSignedBeaconBlock()

	got := v1alpha1.CopySignedBeaconBlock(blk)
	if !reflect.DeepEqual(got, blk) {
		t.Errorf("CopySignedBeaconBlock() = %v, want %v", got, blk)
	}
	assert.NotEmpty(t, got, "Copied signed beacon block has empty fields")
}

func TestCopyBeaconBlock(t *testing.T) {
	blk := genBeaconBlock()

	got := v1alpha1.CopyBeaconBlock(blk)
	if !reflect.DeepEqual(got, blk) {
		t.Errorf("CopyBeaconBlock() = %v, want %v", got, blk)
	}
	assert.NotEmpty(t, got, "Copied beacon block has empty fields")
}

func TestCopyBeaconBlockBody(t *testing.T) {
	body := genBeaconBlockBody()

	got := v1alpha1.CopyBeaconBlockBody(body)
	if !reflect.DeepEqual(got, body) {
		t.Errorf("CopyBeaconBlockBody() = %v, want %v", got, body)
	}
	assert.NotEmpty(t, got, "Copied beacon block body has empty fields")
}

func TestCopySignedBeaconBlockAltair(t *testing.T) {
	sbb := genSignedBeaconBlockAltair()

	got := v1alpha1.CopySignedBeaconBlockAltair(sbb)
	if !reflect.DeepEqual(got, sbb) {
		t.Errorf("CopySignedBeaconBlockAltair() = %v, want %v", got, sbb)
	}
	assert.NotEmpty(t, sbb, "Copied signed beacon block altair has empty fields")
}

func TestCopyBeaconBlockAltair(t *testing.T) {
	b := genBeaconBlockAltair()

	got := v1alpha1.CopyBeaconBlockAltair(b)
	if !reflect.DeepEqual(got, b) {
		t.Errorf("CopyBeaconBlockAltair() = %v, want %v", got, b)
	}
	assert.NotEmpty(t, b, "Copied beacon block altair has empty fields")
}

func TestCopyBeaconBlockBodyAltair(t *testing.T) {
	bb := genBeaconBlockBodyAltair()

	got := v1alpha1.CopyBeaconBlockBodyAltair(bb)
	if !reflect.DeepEqual(got, bb) {
		t.Errorf("CopyBeaconBlockBodyAltair() = %v, want %v", got, bb)
	}
	assert.NotEmpty(t, bb, "Copied beacon block body altair has empty fields")
}

func TestCopyProposerSlashings(t *testing.T) {
	ps := genProposerSlashings(10)

	got := v1alpha1.CopyProposerSlashings(ps)
	if !reflect.DeepEqual(got, ps) {
		t.Errorf("CopyProposerSlashings() = %v, want %v", got, ps)
	}
	assert.NotEmpty(t, got, "Copied proposer slashings have empty fields")
}

func TestCopyProposerSlashing(t *testing.T) {
	ps := genProposerSlashing()

	got := v1alpha1.CopyProposerSlashing(ps)
	if !reflect.DeepEqual(got, ps) {
		t.Errorf("CopyProposerSlashing() = %v, want %v", got, ps)
	}
	assert.NotEmpty(t, got, "Copied proposer slashing has empty fields")
}

func TestCopySignedBeaconBlockHeader(t *testing.T) {
	sbh := genSignedBeaconBlockHeader()

	got := v1alpha1.CopySignedBeaconBlockHeader(sbh)
	if !reflect.DeepEqual(got, sbh) {
		t.Errorf("CopySignedBeaconBlockHeader() = %v, want %v", got, sbh)
	}
	assert.NotEmpty(t, got, "Copied signed beacon block header has empty fields")
}

func TestCopyBeaconBlockHeader(t *testing.T) {
	bh := genBeaconBlockHeader()

	got := v1alpha1.CopyBeaconBlockHeader(bh)
	if !reflect.DeepEqual(got, bh) {
		t.Errorf("CopyBeaconBlockHeader() = %v, want %v", got, bh)
	}
	assert.NotEmpty(t, got, "Copied beacon block header has empty fields")
}

func TestCopyAttesterSlashings(t *testing.T) {
	as := genAttesterSlashings(10)

	got := v1alpha1.CopyAttesterSlashings(as)
	if !reflect.DeepEqual(got, as) {
		t.Errorf("CopyAttesterSlashings() = %v, want %v", got, as)
	}
	assert.NotEmpty(t, got, "Copied attester slashings have empty fields")
}

func TestCopyIndexedAttestation(t *testing.T) {
	ia := genIndexedAttestation()

	got := v1alpha1.CopyIndexedAttestation(ia)
	if !reflect.DeepEqual(got, ia) {
		t.Errorf("CopyIndexedAttestation() = %v, want %v", got, ia)
	}
	assert.NotEmpty(t, got, "Copied indexed attestation has empty fields")
}

func TestCopyAttestations(t *testing.T) {
	atts := genAttestations(10)

	got := v1alpha1.CopyAttestations(atts)
	if !reflect.DeepEqual(got, atts) {
		t.Errorf("CopyAttestations() = %v, want %v", got, atts)
	}
	assert.NotEmpty(t, got, "Copied attestations have empty fields")
}

func TestCopyDeposits(t *testing.T) {
	d := genDeposits(10)

	got := v1alpha1.CopyDeposits(d)
	if !reflect.DeepEqual(got, d) {
		t.Errorf("CopyDeposits() = %v, want %v", got, d)
	}
	assert.NotEmpty(t, got, "Copied deposits have empty fields")
}

func TestCopyDeposit(t *testing.T) {
	d := genDeposit()

	got := v1alpha1.CopyDeposit(d)
	if !reflect.DeepEqual(got, d) {
		t.Errorf("CopyDeposit() = %v, want %v", got, d)
	}
	assert.NotEmpty(t, got, "Copied deposit has empty fields")
}

func TestCopyDepositData(t *testing.T) {
	dd := genDepositData()

	got := v1alpha1.CopyDepositData(dd)
	if !reflect.DeepEqual(got, dd) {
		t.Errorf("CopyDepositData() = %v, want %v", got, dd)
	}
	assert.NotEmpty(t, got, "Copied deposit data has empty fields")
}

func TestCopySignedVoluntaryExits(t *testing.T) {
	sv := genSignedVoluntaryExits(10)

	got := v1alpha1.CopySignedVoluntaryExits(sv)
	if !reflect.DeepEqual(got, sv) {
		t.Errorf("CopySignedVoluntaryExits() = %v, want %v", got, sv)
	}
	assert.NotEmpty(t, got, "Copied signed voluntary exits have empty fields")
}

func TestCopySignedVoluntaryExit(t *testing.T) {
	sv := genSignedVoluntaryExit()

	got := v1alpha1.CopySignedVoluntaryExit(sv)
	if !reflect.DeepEqual(got, sv) {
		t.Errorf("CopySignedVoluntaryExit() = %v, want %v", got, sv)
	}
	assert.NotEmpty(t, got, "Copied signed voluntary exit has empty fields")
}

func TestCopyValidator(t *testing.T) {
	v := genValidator()

	got := v1alpha1.CopyValidator(v)
	if !reflect.DeepEqual(got, v) {
		t.Errorf("CopyValidator() = %v, want %v", got, v)
	}
	assert.NotEmpty(t, got, "Copied validator has empty fields")
}

func TestCopySyncCommitteeMessage(t *testing.T) {
	scm := genSyncCommitteeMessage()

	got := v1alpha1.CopySyncCommitteeMessage(scm)
	if !reflect.DeepEqual(got, scm) {
		t.Errorf("CopySyncCommitteeMessage() = %v, want %v", got, scm)
	}
	assert.NotEmpty(t, got, "Copied sync committee message has empty fields")
}

func TestCopySyncCommitteeContribution(t *testing.T) {
	scc := genSyncCommitteeContribution()

	got := v1alpha1.CopySyncCommitteeContribution(scc)
	if !reflect.DeepEqual(got, scc) {
		t.Errorf("CopySyncCommitteeContribution() = %v, want %v", got, scc)
	}
	assert.NotEmpty(t, got, "Copied sync committee contribution has empty fields")
}

func TestCopySyncAggregate(t *testing.T) {
	sa := genSyncAggregate()

	got := v1alpha1.CopySyncAggregate(sa)
	if !reflect.DeepEqual(got, sa) {
		t.Errorf("CopySyncAggregate() = %v, want %v", got, sa)
	}
	assert.NotEmpty(t, got, "Copied sync aggregate has empty fields")
}

func TestCopyPendingAttestationSlice(t *testing.T) {
	tests := []struct {
		name  string
		input []*v1alpha1.PendingAttestation
	}{
		{
			name:  "nil",
			input: nil,
		},
		{
			name:  "empty",
			input: []*v1alpha1.PendingAttestation{},
		},
		{
			name: "correct copy",
			input: []*v1alpha1.PendingAttestation{
				genPendingAttestation(),
				genPendingAttestation(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v1alpha1.CopyPendingAttestationSlice(tt.input); !reflect.DeepEqual(got, tt.input) {
				t.Errorf("CopyPendingAttestationSlice() = %v, want %v", got, tt.input)
			}
		})
	}
}

func TestCopyPayloadHeader(t *testing.T) {
	p := genPayloadHeader()

	got := v1alpha1.CopyExecutionPayloadHeader(p)
	if !reflect.DeepEqual(got, p) {
		t.Errorf("CopyExecutionPayloadHeader() = %v, want %v", got, p)
	}
	assert.NotEmpty(t, got, "Copied execution payload header has empty fields")
}

func TestCopySignedBeaconBlockMerge(t *testing.T) {
	sbb := genSignedBeaconBlockMerge()

	got := v1alpha1.CopySignedBeaconBlockMerge(sbb)
	if !reflect.DeepEqual(got, sbb) {
		t.Errorf("CopySignedBeaconBlockMerge() = %v, want %v", got, sbb)
	}
	assert.NotEmpty(t, sbb, "Copied signed beacon block Merge has empty fields")
}

func TestCopyBeaconBlockMerge(t *testing.T) {
	b := genBeaconBlockMerge()

	got := v1alpha1.CopyBeaconBlockMerge(b)
	if !reflect.DeepEqual(got, b) {
		t.Errorf("CopyBeaconBlockMerge() = %v, want %v", got, b)
	}
	assert.NotEmpty(t, b, "Copied beacon block Merge has empty fields")
}

func TestCopyBeaconBlockBodyMerge(t *testing.T) {
	bb := genBeaconBlockBodyMerge()

	got := v1alpha1.CopyBeaconBlockBodyMerge(bb)
	if !reflect.DeepEqual(got, bb) {
		t.Errorf("CopyBeaconBlockBodyMerge() = %v, want %v", got, bb)
	}
	assert.NotEmpty(t, bb, "Copied beacon block body Merge has empty fields")
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

func genAttestation() *v1alpha1.Attestation {
	return &v1alpha1.Attestation{
		AggregationBits: bytes(),
		Data:            genAttData(),
		Signature:       bytes(),
	}
}

func genAttestations(num int) []*v1alpha1.Attestation {
	atts := make([]*v1alpha1.Attestation, num)
	for i := 0; i < num; i++ {
		atts[i] = genAttestation()
	}
	return atts
}

func genAttData() *v1alpha1.AttestationData {
	return &v1alpha1.AttestationData{
		Slot:            1,
		CommitteeIndex:  2,
		BeaconBlockRoot: bytes(),
		Source:          genCheckpoint(),
		Target:          genCheckpoint(),
	}
}

func genCheckpoint() *v1alpha1.Checkpoint {
	return &v1alpha1.Checkpoint{
		Epoch: 1,
		Root:  bytes(),
	}
}

func genEth1Data() *v1alpha1.Eth1Data {
	return &v1alpha1.Eth1Data{
		DepositRoot:  bytes(),
		DepositCount: 4,
		BlockHash:    bytes(),
	}
}

func genPendingAttestation() *v1alpha1.PendingAttestation {
	return &v1alpha1.PendingAttestation{
		AggregationBits: bytes(),
		Data:            genAttData(),
		InclusionDelay:  3,
		ProposerIndex:   5,
	}
}

func genSignedBeaconBlock() *v1alpha1.SignedBeaconBlock {
	return &v1alpha1.SignedBeaconBlock{
		Block:     genBeaconBlock(),
		Signature: bytes(),
	}
}

func genBeaconBlock() *v1alpha1.BeaconBlock {
	return &v1alpha1.BeaconBlock{
		Slot:          4,
		ProposerIndex: 5,
		ParentRoot:    bytes(),
		StateRoot:     bytes(),
		Body:          genBeaconBlockBody(),
	}
}

func genBeaconBlockBody() *v1alpha1.BeaconBlockBody {
	return &v1alpha1.BeaconBlockBody{
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

func genProposerSlashing() *v1alpha1.ProposerSlashing {
	return &v1alpha1.ProposerSlashing{
		Header_1: genSignedBeaconBlockHeader(),
		Header_2: genSignedBeaconBlockHeader(),
	}
}

func genProposerSlashings(num int) []*v1alpha1.ProposerSlashing {
	ps := make([]*v1alpha1.ProposerSlashing, num)
	for i := 0; i < num; i++ {
		ps[i] = genProposerSlashing()
	}
	return ps
}

func genAttesterSlashing() *v1alpha1.AttesterSlashing {
	return &v1alpha1.AttesterSlashing{
		Attestation_1: genIndexedAttestation(),
		Attestation_2: genIndexedAttestation(),
	}
}

func genIndexedAttestation() *v1alpha1.IndexedAttestation {
	return &v1alpha1.IndexedAttestation{
		AttestingIndices: []uint64{1, 2, 3},
		Data:             genAttData(),
		Signature:        bytes(),
	}
}

func genAttesterSlashings(num int) []*v1alpha1.AttesterSlashing {
	as := make([]*v1alpha1.AttesterSlashing, num)
	for i := 0; i < num; i++ {
		as[i] = genAttesterSlashing()
	}
	return as
}

func genBeaconBlockHeader() *v1alpha1.BeaconBlockHeader {
	return &v1alpha1.BeaconBlockHeader{
		Slot:          10,
		ProposerIndex: 15,
		ParentRoot:    bytes(),
		StateRoot:     bytes(),
		BodyRoot:      bytes(),
	}
}

func genSignedBeaconBlockHeader() *v1alpha1.SignedBeaconBlockHeader {
	return &v1alpha1.SignedBeaconBlockHeader{
		Header:    genBeaconBlockHeader(),
		Signature: bytes(),
	}
}

func genDepositData() *v1alpha1.Deposit_Data {
	return &v1alpha1.Deposit_Data{
		PublicKey:             bytes(),
		WithdrawalCredentials: bytes(),
		Amount:                20000,
		Signature:             bytes(),
	}
}

func genDeposit() *v1alpha1.Deposit {
	return &v1alpha1.Deposit{
		Data:  genDepositData(),
		Proof: [][]byte{bytes(), bytes(), bytes(), bytes()},
	}
}

func genDeposits(num int) []*v1alpha1.Deposit {
	d := make([]*v1alpha1.Deposit, num)
	for i := 0; i < num; i++ {
		d[i] = genDeposit()
	}
	return d
}

func genVoluntaryExit() *v1alpha1.VoluntaryExit {
	return &v1alpha1.VoluntaryExit{
		Epoch:          5432,
		ValidatorIndex: 888888,
	}
}

func genSignedVoluntaryExit() *v1alpha1.SignedVoluntaryExit {
	return &v1alpha1.SignedVoluntaryExit{
		Exit:      genVoluntaryExit(),
		Signature: bytes(),
	}
}

func genSignedVoluntaryExits(num int) []*v1alpha1.SignedVoluntaryExit {
	sv := make([]*v1alpha1.SignedVoluntaryExit, num)
	for i := 0; i < num; i++ {
		sv[i] = genSignedVoluntaryExit()
	}
	return sv
}

func genValidator() *v1alpha1.Validator {
	return &v1alpha1.Validator{
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

func genSyncCommitteeContribution() *v1alpha1.SyncCommitteeContribution {
	return &v1alpha1.SyncCommitteeContribution{
		Slot:              12333,
		BlockRoot:         bytes(),
		SubcommitteeIndex: 4444,
		AggregationBits:   bytes(),
		Signature:         bytes(),
	}
}

func genSyncAggregate() *v1alpha1.SyncAggregate {
	return &v1alpha1.SyncAggregate{
		SyncCommitteeBits:      bytes(),
		SyncCommitteeSignature: bytes(),
	}
}

func genBeaconBlockBodyAltair() *v1alpha1.BeaconBlockBodyAltair {
	return &v1alpha1.BeaconBlockBodyAltair{
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

func genBeaconBlockAltair() *v1alpha1.BeaconBlockAltair {
	return &v1alpha1.BeaconBlockAltair{
		Slot:          123455,
		ProposerIndex: 55433,
		ParentRoot:    bytes(),
		StateRoot:     bytes(),
		Body:          genBeaconBlockBodyAltair(),
	}
}

func genSignedBeaconBlockAltair() *v1alpha1.SignedBeaconBlockAltair {
	return &v1alpha1.SignedBeaconBlockAltair{
		Block:     genBeaconBlockAltair(),
		Signature: bytes(),
	}
}

func genBeaconBlockBodyMerge() *v1alpha1.BeaconBlockBodyMerge {
	return &v1alpha1.BeaconBlockBodyMerge{
		RandaoReveal:      bytes(),
		Eth1Data:          genEth1Data(),
		Graffiti:          bytes(),
		ProposerSlashings: genProposerSlashings(5),
		AttesterSlashings: genAttesterSlashings(5),
		Attestations:      genAttestations(10),
		Deposits:          genDeposits(5),
		VoluntaryExits:    genSignedVoluntaryExits(12),
		SyncAggregate:     genSyncAggregate(),
		ExecutionPayload:  genPayload(),
	}
}

func genBeaconBlockMerge() *v1alpha1.BeaconBlockMerge {
	return &v1alpha1.BeaconBlockMerge{
		Slot:          123455,
		ProposerIndex: 55433,
		ParentRoot:    bytes(),
		StateRoot:     bytes(),
		Body:          genBeaconBlockBodyMerge(),
	}
}

func genSignedBeaconBlockMerge() *v1alpha1.SignedBeaconBlockMerge {
	return &v1alpha1.SignedBeaconBlockMerge{
		Block:     genBeaconBlockMerge(),
		Signature: bytes(),
	}
}

func genSyncCommitteeMessage() *v1alpha1.SyncCommitteeMessage {
	return &v1alpha1.SyncCommitteeMessage{
		Slot:           424555,
		BlockRoot:      bytes(),
		ValidatorIndex: 5443,
		Signature:      bytes(),
	}
}

func genPayload() *v1alpha1.ExecutionPayload {
	return &v1alpha1.ExecutionPayload{
		ParentHash:    bytes(),
		FeeRecipient:  bytes(),
		StateRoot:     bytes(),
		ReceiptRoot:   bytes(),
		LogsBloom:     bytes(),
		Random:        bytes(),
		BlockNumber:   1,
		GasLimit:      2,
		GasUsed:       3,
		Timestamp:     4,
		ExtraData:     bytes(),
		BaseFeePerGas: bytes(),
		BlockHash:     bytes(),
		Transactions:  [][]byte{{'a'}, {'b'}, {'c'}},
	}
}

func genPayloadHeader() *v1alpha1.ExecutionPayloadHeader {
	return &v1alpha1.ExecutionPayloadHeader{
		ParentHash:       bytes(),
		FeeRecipient:     bytes(),
		StateRoot:        bytes(),
		ReceiptRoot:      bytes(),
		LogsBloom:        bytes(),
		Random:           bytes(),
		BlockNumber:      1,
		GasLimit:         2,
		GasUsed:          3,
		Timestamp:        4,
		ExtraData:        bytes(),
		BaseFeePerGas:    bytes(),
		BlockHash:        bytes(),
		TransactionsRoot: bytes(),
	}
}
