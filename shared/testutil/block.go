package testutil

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/prysmaticlabs/prysm/validator/pandora"
	"math/big"

	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	v1 "github.com/prysmaticlabs/ethereumapis/eth/v1"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
)

// BlockGenConfig is used to define the requested conditions
// for block generation.
type BlockGenConfig struct {
	NumProposerSlashings uint64
	NumAttesterSlashings uint64
	NumAttestations      uint64
	NumDeposits          uint64
	NumVoluntaryExits    uint64
}

// DefaultBlockGenConfig returns the block config that utilizes the
// current params in the beacon config.
func DefaultBlockGenConfig() *BlockGenConfig {
	return &BlockGenConfig{
		NumProposerSlashings: 0,
		NumAttesterSlashings: 0,
		NumAttestations:      1,
		NumDeposits:          0,
		NumVoluntaryExits:    0,
	}
}

// NewBeaconBlock creates a beacon block with minimum marshalable fields.
func NewBeaconBlock() *ethpb.SignedBeaconBlock {
	return &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: make([]byte, 32),
			StateRoot:  make([]byte, 32),
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: make([]byte, 96),
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot: make([]byte, 32),
					BlockHash:   make([]byte, 32),
				},
				Graffiti:          make([]byte, 32),
				Attestations:      []*ethpb.Attestation{},
				AttesterSlashings: []*ethpb.AttesterSlashing{},
				Deposits:          []*ethpb.Deposit{},
				ProposerSlashings: []*ethpb.ProposerSlashing{},
				VoluntaryExits:    []*ethpb.SignedVoluntaryExit{},
				PandoraShard:      []*ethpb.PandoraShard{},
			},
		},
		Signature: make([]byte, 96),
	}
}

// GenerateFullBlock generates a fully valid block with the requested parameters.
// Use BlockGenConfig to declare the conditions you would like the block generated under.
func GenerateFullBlock(
	bState *stateTrie.BeaconState,
	privs []bls.SecretKey,
	conf *BlockGenConfig,
	slot types.Slot,
) (*ethpb.SignedBeaconBlock, error) {
	ctx := context.Background()
	currentSlot := bState.Slot()
	if currentSlot > slot {
		return nil, fmt.Errorf("current slot in state is larger than given slot. %d > %d", currentSlot, slot)
	}
	bState = bState.Copy()

	if conf == nil {
		conf = &BlockGenConfig{}
	}

	var err error
	var pSlashings []*ethpb.ProposerSlashing
	numToGen := conf.NumProposerSlashings
	if numToGen > 0 {
		pSlashings, err = generateProposerSlashings(bState, privs, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d proposer slashings:", numToGen)
		}
	}

	numToGen = conf.NumAttesterSlashings
	var aSlashings []*ethpb.AttesterSlashing
	if numToGen > 0 {
		aSlashings, err = generateAttesterSlashings(bState, privs, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attester slashings:", numToGen)
		}
	}

	numToGen = conf.NumAttestations
	var atts []*ethpb.Attestation
	if numToGen > 0 {
		atts, err = GenerateAttestations(bState, privs, numToGen, slot, false)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attestations:", numToGen)
		}
	}

	numToGen = conf.NumDeposits
	var newDeposits []*ethpb.Deposit
	eth1Data := bState.Eth1Data()
	if numToGen > 0 {
		newDeposits, eth1Data, err = generateDepositsAndEth1Data(bState, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d deposits:", numToGen)
		}
	}

	numToGen = conf.NumVoluntaryExits
	var exits []*ethpb.SignedVoluntaryExit
	if numToGen > 0 {
		exits, err = generateVoluntaryExits(bState, privs, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attester slashings:", numToGen)
		}
	}

	newHeader := bState.LatestBlockHeader()
	prevStateRoot, err := bState.HashTreeRoot(ctx)
	if err != nil {
		return nil, err
	}
	newHeader.StateRoot = prevStateRoot[:]
	parentRoot, err := newHeader.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	if slot == currentSlot {
		slot = currentSlot + 1
	}

	// Temporarily incrementing the beacon state slot here since BeaconProposerIndex is a
	// function deterministic on beacon state slot.
	if err := bState.SetSlot(slot); err != nil {
		return nil, err
	}
	reveal, err := RandaoReveal(bState, helpers.CurrentEpoch(bState), privs)
	if err != nil {
		return nil, err
	}

	idx, err := helpers.BeaconProposerIndex(bState)
	if err != nil {
		return nil, err
	}

	block := &ethpb.BeaconBlock{
		Slot:          slot,
		ParentRoot:    parentRoot[:],
		ProposerIndex: idx,
		Body: &ethpb.BeaconBlockBody{
			Eth1Data:          eth1Data,
			RandaoReveal:      reveal,
			ProposerSlashings: pSlashings,
			AttesterSlashings: aSlashings,
			Attestations:      atts,
			VoluntaryExits:    exits,
			Deposits:          newDeposits,
			Graffiti:          make([]byte, 32),
			PandoraShard:      []*ethpb.PandoraShard{},
		},
	}
	if err := bState.SetSlot(currentSlot); err != nil {
		return nil, err
	}

	signature, err := BlockSignature(bState, block, privs)
	if err != nil {
		return nil, err
	}

	return &ethpb.SignedBeaconBlock{Block: block, Signature: signature.Marshal()}, nil
}

// GenerateProposerSlashingForValidator for a specific validator index.
func GenerateProposerSlashingForValidator(
	bState *stateTrie.BeaconState,
	priv bls.SecretKey,
	idx types.ValidatorIndex,
) (*ethpb.ProposerSlashing, error) {
	header1 := HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: idx,
			Slot:          bState.Slot(),
			BodyRoot:      bytesutil.PadTo([]byte{0, 1, 0}, 32),
		},
	})
	currentEpoch := helpers.CurrentEpoch(bState)
	var err error
	header1.Signature, err = helpers.ComputeDomainAndSign(bState, currentEpoch, header1.Header, params.BeaconConfig().DomainBeaconProposer, priv)
	if err != nil {
		return nil, err
	}

	header2 := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: idx,
			Slot:          bState.Slot(),
			BodyRoot:      bytesutil.PadTo([]byte{0, 2, 0}, 32),
			StateRoot:     make([]byte, 32),
			ParentRoot:    make([]byte, 32),
		},
	}
	header2.Signature, err = helpers.ComputeDomainAndSign(bState, currentEpoch, header2.Header, params.BeaconConfig().DomainBeaconProposer, priv)
	if err != nil {
		return nil, err
	}

	return &ethpb.ProposerSlashing{
		Header_1: header1,
		Header_2: header2,
	}, nil
}

func generateProposerSlashings(
	bState *stateTrie.BeaconState,
	privs []bls.SecretKey,
	numSlashings uint64,
) ([]*ethpb.ProposerSlashing, error) {
	proposerSlashings := make([]*ethpb.ProposerSlashing, numSlashings)
	for i := uint64(0); i < numSlashings; i++ {
		proposerIndex, err := randValIndex(bState)
		if err != nil {
			return nil, err
		}
		slashing, err := GenerateProposerSlashingForValidator(bState, privs[proposerIndex], proposerIndex)
		if err != nil {
			return nil, err
		}
		proposerSlashings[i] = slashing
	}
	return proposerSlashings, nil
}

// GenerateAttesterSlashingForValidator for a specific validator index.
func GenerateAttesterSlashingForValidator(
	bState *stateTrie.BeaconState,
	priv bls.SecretKey,
	idx types.ValidatorIndex,
) (*ethpb.AttesterSlashing, error) {
	currentEpoch := helpers.CurrentEpoch(bState)

	att1 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Slot:            bState.Slot(),
			CommitteeIndex:  0,
			BeaconBlockRoot: make([]byte, 32),
			Target: &ethpb.Checkpoint{
				Epoch: currentEpoch,
				Root:  params.BeaconConfig().ZeroHash[:],
			},
			Source: &ethpb.Checkpoint{
				Epoch: currentEpoch + 1,
				Root:  params.BeaconConfig().ZeroHash[:],
			},
		},
		AttestingIndices: []uint64{uint64(idx)},
	}
	var err error
	att1.Signature, err = helpers.ComputeDomainAndSign(bState, currentEpoch, att1.Data, params.BeaconConfig().DomainBeaconAttester, priv)
	if err != nil {
		return nil, err
	}

	att2 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Slot:            bState.Slot(),
			CommitteeIndex:  0,
			BeaconBlockRoot: make([]byte, 32),
			Target: &ethpb.Checkpoint{
				Epoch: currentEpoch,
				Root:  params.BeaconConfig().ZeroHash[:],
			},
			Source: &ethpb.Checkpoint{
				Epoch: currentEpoch,
				Root:  params.BeaconConfig().ZeroHash[:],
			},
		},
		AttestingIndices: []uint64{uint64(idx)},
	}
	att2.Signature, err = helpers.ComputeDomainAndSign(bState, currentEpoch, att2.Data, params.BeaconConfig().DomainBeaconAttester, priv)
	if err != nil {
		return nil, err
	}

	return &ethpb.AttesterSlashing{
		Attestation_1: att1,
		Attestation_2: att2,
	}, nil
}

func generateAttesterSlashings(
	bState *stateTrie.BeaconState,
	privs []bls.SecretKey,
	numSlashings uint64,
) ([]*ethpb.AttesterSlashing, error) {
	attesterSlashings := make([]*ethpb.AttesterSlashing, numSlashings)
	randGen := rand.NewDeterministicGenerator()
	for i := uint64(0); i < numSlashings; i++ {
		committeeIndex := randGen.Uint64() % params.BeaconConfig().MaxCommitteesPerSlot
		committee, err := helpers.BeaconCommitteeFromState(bState, bState.Slot(), types.CommitteeIndex(committeeIndex))
		if err != nil {
			return nil, err
		}
		randIndex := randGen.Uint64() % uint64(len(committee))
		valIndex := committee[randIndex]
		slashing, err := GenerateAttesterSlashingForValidator(bState, privs[valIndex], valIndex)
		if err != nil {
			return nil, err
		}
		attesterSlashings[i] = slashing
	}
	return attesterSlashings, nil
}

func generateDepositsAndEth1Data(
	bState *stateTrie.BeaconState,
	numDeposits uint64,
) (
	[]*ethpb.Deposit,
	*ethpb.Eth1Data,
	error,
) {
	previousDepsLen := bState.Eth1DepositIndex()
	currentDeposits, _, err := DeterministicDepositsAndKeys(previousDepsLen + numDeposits)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get deposits")
	}
	eth1Data, err := DeterministicEth1Data(len(currentDeposits))
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get eth1data")
	}
	return currentDeposits[previousDepsLen:], eth1Data, nil
}

func generateVoluntaryExits(
	bState *stateTrie.BeaconState,
	privs []bls.SecretKey,
	numExits uint64,
) ([]*ethpb.SignedVoluntaryExit, error) {
	currentEpoch := helpers.CurrentEpoch(bState)

	voluntaryExits := make([]*ethpb.SignedVoluntaryExit, numExits)
	for i := 0; i < len(voluntaryExits); i++ {
		valIndex, err := randValIndex(bState)
		if err != nil {
			return nil, err
		}
		exit := &ethpb.SignedVoluntaryExit{
			Exit: &ethpb.VoluntaryExit{
				Epoch:          helpers.PrevEpoch(bState),
				ValidatorIndex: valIndex,
			},
		}
		exit.Signature, err = helpers.ComputeDomainAndSign(bState, currentEpoch, exit.Exit, params.BeaconConfig().DomainVoluntaryExit, privs[valIndex])
		if err != nil {
			return nil, err
		}
		voluntaryExits[i] = exit
	}
	return voluntaryExits, nil
}

func randValIndex(bState *stateTrie.BeaconState) (types.ValidatorIndex, error) {
	activeCount, err := helpers.ActiveValidatorCount(bState, helpers.CurrentEpoch(bState))
	if err != nil {
		return 0, err
	}
	return types.ValidatorIndex(rand.NewGenerator().Uint64() % activeCount), nil
}

// HydrateSignedBeaconHeader hydrates a signed beacon block header with correct field length sizes
// to comply with fssz marshalling and unmarshalling rules.
func HydrateSignedBeaconHeader(h *ethpb.SignedBeaconBlockHeader) *ethpb.SignedBeaconBlockHeader {
	if h.Signature == nil {
		h.Signature = make([]byte, params.BeaconConfig().BLSSignatureLength)
	}
	h.Header = HydrateBeaconHeader(h.Header)
	return h
}

// HydrateBeaconHeader hydrates a beacon block header with correct field length sizes
// to comply with fssz marshalling and unmarshalling rules.
func HydrateBeaconHeader(h *ethpb.BeaconBlockHeader) *ethpb.BeaconBlockHeader {
	if h == nil {
		h = &ethpb.BeaconBlockHeader{}
	}
	if h.BodyRoot == nil {
		h.BodyRoot = make([]byte, 32)
	}
	if h.StateRoot == nil {
		h.StateRoot = make([]byte, 32)
	}
	if h.ParentRoot == nil {
		h.ParentRoot = make([]byte, 32)
	}
	return h
}

// HydrateSignedBeaconBlock hydrates a signed beacon block with correct field length sizes
// to comply with fssz marshalling and unmarshalling rules.
func HydrateSignedBeaconBlock(b *ethpb.SignedBeaconBlock) *ethpb.SignedBeaconBlock {
	if b.Signature == nil {
		b.Signature = make([]byte, params.BeaconConfig().BLSSignatureLength)
	}
	b.Block = HydrateBeaconBlock(b.Block)
	return b
}

// HydrateBeaconBlock hydrates a beacon block with correct field length sizes
// to comply with fssz marshalling and unmarshalling rules.
func HydrateBeaconBlock(b *ethpb.BeaconBlock) *ethpb.BeaconBlock {
	if b == nil {
		b = &ethpb.BeaconBlock{}
	}
	if b.ParentRoot == nil {
		b.ParentRoot = make([]byte, 32)
	}
	if b.StateRoot == nil {
		b.StateRoot = make([]byte, 32)
	}
	b.Body = HydrateBeaconBlockBody(b.Body)
	return b
}

// HydrateBeaconBlockBody hydrates a beacon block body with correct field length sizes
// to comply with fssz marshalling and unmarshalling rules.
func HydrateBeaconBlockBody(b *ethpb.BeaconBlockBody) *ethpb.BeaconBlockBody {
	if b == nil {
		b = &ethpb.BeaconBlockBody{}
	}
	if b.RandaoReveal == nil {
		b.RandaoReveal = make([]byte, params.BeaconConfig().BLSSignatureLength)
	}
	if b.Graffiti == nil {
		b.Graffiti = make([]byte, 32)
	}
	if b.Eth1Data == nil {
		b.Eth1Data = &ethpb.Eth1Data{
			DepositRoot: make([]byte, 32),
			BlockHash:   make([]byte, 32),
		}
	}
	return b
}

// HydrateV1SignedBeaconBlock hydrates a signed beacon block with correct field length sizes
// to comply with fssz marshalling and unmarshalling rules.
func HydrateV1SignedBeaconBlock(b *v1.SignedBeaconBlock) *v1.SignedBeaconBlock {
	if b.Signature == nil {
		b.Signature = make([]byte, params.BeaconConfig().BLSSignatureLength)
	}
	b.Block = HydrateV1BeaconBlock(b.Block)
	return b
}

// HydrateV1BeaconBlock hydrates a beacon block with correct field length sizes
// to comply with fssz marshalling and unmarshalling rules.
func HydrateV1BeaconBlock(b *v1.BeaconBlock) *v1.BeaconBlock {
	if b == nil {
		b = &v1.BeaconBlock{}
	}
	if b.ParentRoot == nil {
		b.ParentRoot = make([]byte, 32)
	}
	if b.StateRoot == nil {
		b.StateRoot = make([]byte, 32)
	}
	b.Body = HydrateV1BeaconBlockBody(b.Body)
	return b
}

// HydrateV1BeaconBlockBody hydrates a beacon block body with correct field length sizes
// to comply with fssz marshalling and unmarshalling rules.
func HydrateV1BeaconBlockBody(b *v1.BeaconBlockBody) *v1.BeaconBlockBody {
	if b == nil {
		b = &v1.BeaconBlockBody{}
	}
	if b.RandaoReveal == nil {
		b.RandaoReveal = make([]byte, params.BeaconConfig().BLSSignatureLength)
	}
	if b.Graffiti == nil {
		b.Graffiti = make([]byte, 32)
	}
	if b.Eth1Data == nil {
		b.Eth1Data = &v1.Eth1Data{
			DepositRoot: make([]byte, 32),
			BlockHash:   make([]byte, 32),
		}
	}
	return b
}

// getDummyBlock method creates a brand new block with extraData
func NewPandoraBlock(slot types.Slot, proposerIndex uint64) (*gethTypes.Header, *pandora.ExtraData) {
	epoch := types.Epoch(slot / params.BeaconConfig().SlotsPerEpoch)
	extraData := pandora.ExtraData{
		Slot:          uint64(slot),
		Epoch:         uint64(epoch),
		ProposerIndex: proposerIndex,
	}
	extraDataByte, err := rlp.EncodeToBytes(extraData)
	if err != nil {
		return nil, nil
	}

	block := gethTypes.NewBlock(&gethTypes.Header{
		ParentHash:  gethTypes.EmptyRootHash,
		UncleHash:   gethTypes.EmptyUncleHash,
		Coinbase:    common.HexToAddress("8888f1f195afa192cfee860698584c030f4c9db1"),
		Root:        common.HexToHash("ef1552a40b7165c3cd773806b9e0c165b75356e0314bf0706f279c729f51e017"),
		TxHash:      gethTypes.EmptyRootHash,
		ReceiptHash: gethTypes.EmptyRootHash,
		Difficulty:  big.NewInt(131072),
		Number:      big.NewInt(314),
		GasLimit:    uint64(3141592),
		GasUsed:     uint64(21000),
		Time:        uint64(1426516743),
		Extra:       extraDataByte,
		MixDigest:   gethTypes.EmptyRootHash,
		Nonce:       gethTypes.BlockNonce{0x01, 0x02, 0x03},
	}, nil, nil, nil, nil)

	return block.Header(), &extraData
}

// NewBeaconBlockWithPandoraSharding
func NewBeaconBlockWithPandoraSharding(panHeader *gethTypes.Header, slot types.Slot) *ethpb.SignedBeaconBlock {
	beaconBlock := NewBeaconBlock()
	beaconBlock.Block.Slot = slot

	panState := new(ethpb.PandoraShard)
	panState.BlockNumber = panHeader.Number.Uint64() - 1
	panState.Hash = gethTypes.EmptyRootHash.Bytes()
	panState.ParentHash = panHeader.ParentHash.Bytes()
	panState.StateRoot = panHeader.Root.Bytes()
	panState.TxHash = panHeader.TxHash.Bytes()
	panState.ReceiptHash = panHeader.ReceiptHash.Bytes()
	panState.Signature = make([]byte, params.BeaconConfig().BLSSignatureLength)

	pandoraShards := make([]*ethpb.PandoraShard, 1)
	pandoraShards[0] = panState

	beaconBlock.Block.Body.PandoraShard = pandoraShards
	return beaconBlock
}
