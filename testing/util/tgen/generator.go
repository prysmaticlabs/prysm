package tgen

import (
	"math/rand"

	"github.com/prysmaticlabs/go-bitfield"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

var b20 = make([]byte, 20)
var b32 = make([]byte, 32)
var b48 = make([]byte, 48)
var b64 = make([]byte, 64)
var b96 = make([]byte, 96)
var b256 = make([]byte, 256)

func init() {
	_, err := rand.Read(b20)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 20; i++ {
		if b20[i] == 0x00 {
			b20[i] = uint8(rand.Int())
		}
	}
	_, err = rand.Read(b32)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 32; i++ {
		if b32[i] == 0x00 {
			b32[i] = uint8(rand.Int())
		}
	}
	_, err = rand.Read(b48)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 48; i++ {
		if b48[i] == 0x00 {
			b48[i] = uint8(rand.Int())
		}
	}
	_, err = rand.Read(b64)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 64; i++ {
		if b64[i] == 0x00 {
			b64[i] = uint8(rand.Int())
		}
	}
	_, err = rand.Read(b96)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 96; i++ {
		if b96[i] == 0x00 {
			b96[i] = uint8(rand.Int())
		}
	}
	_, err = rand.Read(b256)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 256; i++ {
		if b256[i] == 0x00 {
			b256[i] = uint8(rand.Int())
		}
	}
}

type byteSlices struct {
	B20  []byte
	B32  []byte
	B48  []byte
	B64  []byte
	B96  []byte
	B256 []byte
}

type BlockFields struct {
	byteSlices
	Deposits          []*eth.Deposit
	Atts              []*eth.Attestation
	ProposerSlashings []*eth.ProposerSlashing
	AttesterSlashings []*eth.AttesterSlashing
	VoluntaryExits    []*eth.SignedVoluntaryExit
	SyncAggregate     *eth.SyncAggregate
	ExecPayload       *enginev1.ExecutionPayload
	ExecPayloadHeader *eth.ExecutionPayloadHeader
}

func GetBlockFields() BlockFields {
	deposits := make([]*eth.Deposit, 16)
	for i := range deposits {
		deposits[i] = &eth.Deposit{}
		deposits[i].Proof = make([][]byte, 33)
		for j := range deposits[i].Proof {
			deposits[i].Proof[j] = b32
		}
		deposits[i].Data = &eth.Deposit_Data{
			PublicKey:             b48,
			WithdrawalCredentials: b32,
			Amount:                128,
			Signature:             b96,
		}
	}
	atts := make([]*eth.Attestation, 128)
	for i := range atts {
		atts[i] = &eth.Attestation{}
		atts[i].Signature = b96
		atts[i].AggregationBits = bitfield.NewBitlist(1)
		atts[i].Data = &eth.AttestationData{
			Slot:            128,
			CommitteeIndex:  128,
			BeaconBlockRoot: b32,
			Source: &eth.Checkpoint{
				Epoch: 128,
				Root:  b32,
			},
			Target: &eth.Checkpoint{
				Epoch: 128,
				Root:  b32,
			},
		}
	}
	proposerSlashing := &eth.ProposerSlashing{
		Header_1: &eth.SignedBeaconBlockHeader{
			Header: &eth.BeaconBlockHeader{
				Slot:          128,
				ProposerIndex: 128,
				ParentRoot:    b32,
				StateRoot:     b32,
				BodyRoot:      b32,
			},
			Signature: b96,
		},
		Header_2: &eth.SignedBeaconBlockHeader{
			Header: &eth.BeaconBlockHeader{
				Slot:          128,
				ProposerIndex: 128,
				ParentRoot:    b32,
				StateRoot:     b32,
				BodyRoot:      b32,
			},
			Signature: b96,
		},
	}
	attesterSlashing := &eth.AttesterSlashing{
		Attestation_1: &eth.IndexedAttestation{
			AttestingIndices: []uint64{1, 2, 8},
			Data: &eth.AttestationData{
				Slot:            128,
				CommitteeIndex:  128,
				BeaconBlockRoot: b32,
				Source: &eth.Checkpoint{
					Epoch: 128,
					Root:  b32,
				},
				Target: &eth.Checkpoint{
					Epoch: 128,
					Root:  b32,
				},
			},
			Signature: b96,
		},
		Attestation_2: &eth.IndexedAttestation{
			AttestingIndices: []uint64{1, 2, 8},
			Data: &eth.AttestationData{
				Slot:            128,
				CommitteeIndex:  128,
				BeaconBlockRoot: b32,
				Source: &eth.Checkpoint{
					Epoch: 128,
					Root:  b32,
				},
				Target: &eth.Checkpoint{
					Epoch: 128,
					Root:  b32,
				},
			},
			Signature: b96,
		},
	}
	voluntaryExit := &eth.SignedVoluntaryExit{
		Exit: &eth.VoluntaryExit{
			Epoch:          128,
			ValidatorIndex: 128,
		},
		Signature: b96,
	}
	syncAggregate := &eth.SyncAggregate{
		SyncCommitteeBits:      b64,
		SyncCommitteeSignature: b96,
	}
	execPayload := &enginev1.ExecutionPayload{
		ParentHash:    b32,
		FeeRecipient:  b20,
		StateRoot:     b32,
		ReceiptsRoot:  b32,
		LogsBloom:     b256,
		PrevRandao:    b32,
		BlockNumber:   128,
		GasLimit:      128,
		GasUsed:       128,
		Timestamp:     128,
		ExtraData:     b32,
		BaseFeePerGas: b32,
		BlockHash:     b32,
		Transactions: [][]byte{
			[]byte("transaction1"),
			[]byte("transaction2"),
			[]byte("transaction8"),
		},
	}
	execPayloadHeader := &eth.ExecutionPayloadHeader{
		ParentHash:       b32,
		FeeRecipient:     b20,
		StateRoot:        b32,
		ReceiptsRoot:     b32,
		LogsBloom:        b256,
		PrevRandao:       b32,
		BlockNumber:      128,
		GasLimit:         128,
		GasUsed:          128,
		Timestamp:        128,
		ExtraData:        b32,
		BaseFeePerGas:    b32,
		BlockHash:        b32,
		TransactionsRoot: b32,
	}

	return BlockFields{
		byteSlices: byteSlices{
			B20:  b20,
			B32:  b32,
			B48:  b48,
			B96:  b96,
			B256: b256,
		},
		Deposits:          deposits,
		Atts:              atts,
		ProposerSlashings: []*eth.ProposerSlashing{proposerSlashing},
		AttesterSlashings: []*eth.AttesterSlashing{attesterSlashing},
		VoluntaryExits:    []*eth.SignedVoluntaryExit{voluntaryExit},
		SyncAggregate:     syncAggregate,
		ExecPayload:       execPayload,
		ExecPayloadHeader: execPayloadHeader,
	}
}

func PbBlockBodyPhase0() *eth.BeaconBlockBody {
	f := GetBlockFields()
	return &eth.BeaconBlockBody{
		RandaoReveal: f.B96,
		Eth1Data: &eth.Eth1Data{
			DepositRoot:  f.B32,
			DepositCount: 128,
			BlockHash:    f.B32,
		},
		Graffiti:          f.B32,
		ProposerSlashings: f.ProposerSlashings,
		AttesterSlashings: f.AttesterSlashings,
		Attestations:      f.Atts,
		Deposits:          f.Deposits,
		VoluntaryExits:    f.VoluntaryExits,
	}
}

func PbBlockBodyAltair() *eth.BeaconBlockBodyAltair {
	f := GetBlockFields()
	return &eth.BeaconBlockBodyAltair{
		RandaoReveal: f.B96,
		Eth1Data: &eth.Eth1Data{
			DepositRoot:  f.B32,
			DepositCount: 128,
			BlockHash:    f.B32,
		},
		Graffiti:          f.B32,
		ProposerSlashings: f.ProposerSlashings,
		AttesterSlashings: f.AttesterSlashings,
		Attestations:      f.Atts,
		Deposits:          f.Deposits,
		VoluntaryExits:    f.VoluntaryExits,
		SyncAggregate:     f.SyncAggregate,
	}
}
