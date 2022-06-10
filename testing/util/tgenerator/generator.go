package tgenerator

import (
	"math/rand"

	"github.com/prysmaticlabs/go-bitfield"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

type fields struct {
	b20               []byte
	b32               []byte
	b48               []byte
	b96               []byte
	b256              []byte
	deposits          []*eth.Deposit
	atts              []*eth.Attestation
	proposerSlashings []*eth.ProposerSlashing
	attesterSlashings []*eth.AttesterSlashing
	voluntaryExits    []*eth.SignedVoluntaryExit
	syncAggregate     *eth.SyncAggregate
	execPayload       *enginev1.ExecutionPayload
	execPayloadHeader *eth.ExecutionPayloadHeader
}

func getFields() fields {
	b20 := bytes20()
	b32 := bytes32()
	b48 := bytes48()
	b96 := bytes96()
	b256 := bytes256()
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
	syncCommitteeBits := bitfield.NewBitvector512()
	syncCommitteeBits.SetBitAt(1, true)
	syncCommitteeBits.SetBitAt(2, true)
	syncCommitteeBits.SetBitAt(8, true)
	syncAggregate := &eth.SyncAggregate{
		SyncCommitteeBits:      syncCommitteeBits,
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

	return fields{
		b20:               b20,
		b32:               b32,
		b48:               b48,
		b96:               b96,
		b256:              b256,
		deposits:          deposits,
		atts:              atts,
		proposerSlashings: []*eth.ProposerSlashing{proposerSlashing},
		attesterSlashings: []*eth.AttesterSlashing{attesterSlashing},
		voluntaryExits:    []*eth.SignedVoluntaryExit{voluntaryExit},
		syncAggregate:     syncAggregate,
		execPayload:       execPayload,
		execPayloadHeader: execPayloadHeader,
	}
}

func PbBlockBodyPhase0() *eth.BeaconBlockBody {
	f := getFields()
	return &eth.BeaconBlockBody{
		RandaoReveal: f.b96,
		Eth1Data: &eth.Eth1Data{
			DepositRoot:  f.b32,
			DepositCount: 128,
			BlockHash:    f.b32,
		},
		Graffiti:          f.b32,
		ProposerSlashings: f.proposerSlashings,
		AttesterSlashings: f.attesterSlashings,
		Attestations:      f.atts,
		Deposits:          f.deposits,
		VoluntaryExits:    f.voluntaryExits,
	}
}

func bytes20() []byte {
	b := make([]byte, 20)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 20; i++ {
		if b[i] == 0x00 {
			b[i] = uint8(rand.Int())
		}
	}
	return b
}

func bytes32() []byte {
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

func bytes48() []byte {
	b := make([]byte, 48)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 48; i++ {
		if b[i] == 0x00 {
			b[i] = uint8(rand.Int())
		}
	}
	return b
}

func bytes96() []byte {
	b := make([]byte, 96)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 96; i++ {
		if b[i] == 0x00 {
			b[i] = uint8(rand.Int())
		}
	}
	return b
}

func bytes256() []byte {
	b := make([]byte, 256)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 256; i++ {
		if b[i] == 0x00 {
			b[i] = uint8(rand.Int())
		}
	}
	return b
}
