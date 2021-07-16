package copyutil

import (
	"math/rand"
	"reflect"
	"testing"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestCopyAttestations(t *testing.T) {
	atts := genAttestations(10)

	got := CopyAttestations(atts)
	if !reflect.DeepEqual(got, atts) {
		t.Errorf("CopyAttestations() = %v, want %v", got, atts)
	}
	assert.NotEmpty(t, got, "Copied attestations have empty fields")
}

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

func TestCopyProposerSlashings(t *testing.T) {
	ps := genProposerSlashings(10)

	got := CopyProposerSlashings(ps)
	if !reflect.DeepEqual(got, ps) {
		t.Errorf("CopyProposerSlashings() = %v, want %v", got, ps)
	}
	assert.NotEmpty(t, got, "Copied proposer slashings have empty fields")
}

// TODO: the rest of the copy methods.

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

func genAttestation() *ethpb.Attestation {
	return &ethpb.Attestation{
		AggregationBits: bytes(),
		Data:            genAttData(),
		Signature:       bytes(),
	}
}

func genAttestations(num int) []*ethpb.Attestation {
	atts := make([]*ethpb.Attestation, num)
	for i := 0; i < num; i++ {
		atts[i] = genAttestation()
	}
	return atts
}

func genAttData() *ethpb.AttestationData {
	return &ethpb.AttestationData{
		Slot:            1,
		CommitteeIndex:  2,
		BeaconBlockRoot: bytes(),
		Source:          genCheckpoint(),
		Target:          genCheckpoint(),
	}
}

func genCheckpoint() *ethpb.Checkpoint {
	return &ethpb.Checkpoint{
		Epoch: 1,
		Root:  bytes(),
	}
}

func genEth1Data() *ethpb.Eth1Data {
	return &ethpb.Eth1Data{
		DepositRoot:  bytes(),
		DepositCount: 4,
		BlockHash:    bytes(),
	}
}

func genPendingAttestation() *pbp2p.PendingAttestation {
	return &pbp2p.PendingAttestation{
		AggregationBits: bytes(),
		Data:            genAttData(),
		InclusionDelay:  3,
		ProposerIndex:   5,
	}
}

func genSignedBeaconBlock() *ethpb.SignedBeaconBlock {
	return &ethpb.SignedBeaconBlock{
		Block:     genBeaconBlock(),
		Signature: bytes(),
	}
}

func genBeaconBlock() *ethpb.BeaconBlock {
	return &ethpb.BeaconBlock{
		Slot:          4,
		ProposerIndex: 5,
		ParentRoot:    bytes(),
		StateRoot:     bytes(),
		Body:          genBeaconBlockBody(),
	}
}

func genBeaconBlockBody() *ethpb.BeaconBlockBody {
	return &ethpb.BeaconBlockBody{
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

func genProposerSlashing() *ethpb.ProposerSlashing {
	return &ethpb.ProposerSlashing{
		Header_1: genSignedBeaconBlockHeader(),
		Header_2: genSignedBeaconBlockHeader(),
	}
}

func genProposerSlashings(num int) []*ethpb.ProposerSlashing {
	ps := make([]*ethpb.ProposerSlashing, num)
	for i := 0; i < num; i++ {
		ps[i] = genProposerSlashing()
	}
	return ps
}

func genAttesterSlashing() *ethpb.AttesterSlashing {
	return &ethpb.AttesterSlashing{
		Attestation_1: genIndexedAttestation(),
		Attestation_2: genIndexedAttestation(),
	}
}

func genIndexedAttestation() *ethpb.IndexedAttestation {
	return &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{1, 2, 3},
		Data:             genAttData(),
		Signature:        bytes(),
	}
}

func genAttesterSlashings(num int) []*ethpb.AttesterSlashing {
	as := make([]*ethpb.AttesterSlashing, num)
	for i := 0; i < num; i++ {
		as[i] = genAttesterSlashing()
	}
	return as
}

func genBeaconBlockHeader() *ethpb.BeaconBlockHeader {
	return &ethpb.BeaconBlockHeader{
		Slot:          10,
		ProposerIndex: 15,
		ParentRoot:    bytes(),
		StateRoot:     bytes(),
		BodyRoot:      bytes(),
	}
}

func genSignedBeaconBlockHeader() *ethpb.SignedBeaconBlockHeader {
	return &ethpb.SignedBeaconBlockHeader{
		Header:    genBeaconBlockHeader(),
		Signature: bytes(),
	}
}

func genDepositData() *ethpb.Deposit_Data {
	return &ethpb.Deposit_Data{
		PublicKey:             bytes(),
		WithdrawalCredentials: bytes(),
		Amount:                20000,
		Signature:             bytes(),
	}
}

func genDeposit() *ethpb.Deposit {
	return &ethpb.Deposit{
		Data:  genDepositData(),
		Proof: [][]byte{bytes(), bytes(), bytes(), bytes()},
	}
}

func genDeposits(num int) []*ethpb.Deposit {
	d := make([]*ethpb.Deposit, num)
	for i := 0; i < num; i++ {
		d[i] = genDeposit()
	}
	return d
}

func genVoluntaryExit() *ethpb.VoluntaryExit {
	return &ethpb.VoluntaryExit{
		Epoch:          5432,
		ValidatorIndex: 888888,
	}
}

func genSignedVoluntaryExit() *ethpb.SignedVoluntaryExit {
	return &ethpb.SignedVoluntaryExit{
		Exit:      genVoluntaryExit(),
		Signature: bytes(),
	}
}

func genSignedVoluntaryExits(num int) []*ethpb.SignedVoluntaryExit {
	sv := make([]*ethpb.SignedVoluntaryExit, num)
	for i := 0; i < num; i++ {
		sv[i] = genSignedVoluntaryExit()
	}
	return sv
}
