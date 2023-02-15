package util

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/altair"
	b "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/container/trie"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/interop"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

var errUnsupportedVersion = errors.New("schema version not supported by premineGenesisConfig")

type premineGenesisConfig struct {
	GenesisTime     uint64
	NVals           uint64
	PregenesisCreds uint64
	Version         int          // as in "github.com/prysmaticlabs/prysm/v3/runtime/version"
	GB              *types.Block // geth genesis block
}

// NewPreminedGenesis creates a genesis BeaconState at the given fork version, suitable for using as an e2e genesis.
func NewPreminedGenesis(ctx context.Context, t, nvals, pCreds uint64, version int, gb *types.Block) (state.BeaconState, error) {
	return (&premineGenesisConfig{
		GenesisTime:     t,
		NVals:           nvals,
		PregenesisCreds: pCreds,
		Version:         version,
		GB:              gb,
	}).prepare(ctx)
}

func (s *premineGenesisConfig) prepare(ctx context.Context) (state.BeaconState, error) {
	switch s.Version {
	case version.Phase0, version.Altair, version.Bellatrix:
	default:
		return nil, errors.Wrapf(errUnsupportedVersion, "version=%s", version.String(s.Version))
	}

	st, err := s.empty()
	if err != nil {
		return nil, err
	}
	if err = s.processDeposits(ctx, st); err != nil {
		return nil, err
	}
	if err = s.populate(st); err != nil {
		return nil, err
	}

	return st, nil
}

func (s *premineGenesisConfig) empty() (state.BeaconState, error) {
	var e state.BeaconState
	var err error
	switch s.Version {
	case version.Phase0:
		e, err = state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{})
		if err != nil {
			return nil, err
		}
	case version.Altair:
		e, err = state_native.InitializeFromProtoAltair(&ethpb.BeaconStateAltair{})
		if err != nil {
			return nil, err
		}
	case version.Bellatrix:
		e, err = state_native.InitializeFromProtoBellatrix(&ethpb.BeaconStateBellatrix{})
		if err != nil {
			return nil, err
		}
	default:
		return nil, errUnsupportedVersion
	}
	if err = e.SetSlot(0); err != nil {
		return nil, err
	}
	if err = e.SetValidators([]*ethpb.Validator{}); err != nil {
		return nil, err
	}
	if err = e.SetBalances([]uint64{}); err != nil {
		return nil, err
	}
	if err = e.SetJustificationBits([]byte{0}); err != nil {
		return nil, err
	}
	if err = e.SetHistoricalRoots([][]byte{}); err != nil {
		return nil, err
	}
	zcp := &ethpb.Checkpoint{
		Epoch: 0,
		Root:  params.BeaconConfig().ZeroHash[:],
	}
	if err = e.SetPreviousJustifiedCheckpoint(zcp); err != nil {
		return nil, err
	}
	if err = e.SetCurrentJustifiedCheckpoint(zcp); err != nil {
		return nil, err
	}
	if err = e.SetFinalizedCheckpoint(zcp); err != nil {
		return nil, err
	}
	if err = e.SetEth1DataVotes([]*ethpb.Eth1Data{}); err != nil {
		return nil, err
	}
	if s.Version == version.Phase0 {
		if err = e.SetCurrentEpochAttestations([]*ethpb.PendingAttestation{}); err != nil {
			return nil, err
		}
		if err = e.SetPreviousEpochAttestations([]*ethpb.PendingAttestation{}); err != nil {
			return nil, err
		}
	}
	return e.Copy(), nil
}

func (s *premineGenesisConfig) processDeposits(ctx context.Context, g state.BeaconState) error {
	deposits, err := s.deposits()
	if err != nil {
		return err
	}
	if err = s.setEth1Data(g); err != nil {
		return err
	}
	if _, err = helpers.UpdateGenesisEth1Data(g, deposits, g.Eth1Data()); err != nil {
		return err
	}
	_, err = b.ProcessPreGenesisDeposits(ctx, g, deposits)
	if err != nil {
		return errors.Wrap(err, "could not process validator deposits")
	}
	return nil
}

func (s *premineGenesisConfig) deposits() ([]*ethpb.Deposit, error) {
	prv, pub, err := s.keys()
	if err != nil {
		return nil, err
	}
	items, roots, err := interop.DepositDataFromKeysWithExecCreds(prv, pub, s.PregenesisCreds)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate deposit data from keys")
	}
	t, err := trie.GenerateTrieFromItems(roots, params.BeaconConfig().DepositContractTreeDepth)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate Merkle trie for deposit proofs")
	}
	deposits, err := interop.GenerateDepositsFromData(items, t)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate deposits from the deposit data provided")
	}
	return deposits, nil
}

func (s *premineGenesisConfig) keys() ([]bls.SecretKey, []bls.PublicKey, error) {
	prv, pub, err := interop.DeterministicallyGenerateKeys(0, s.NVals)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not deterministically generate keys for %d validators", s.NVals)
	}
	return prv, pub, nil
}

func (s *premineGenesisConfig) setEth1Data(g state.BeaconState) error {
	if err := g.SetEth1DepositIndex(0); err != nil {
		return err
	}
	dr, err := emptyDepositRoot()
	if err != nil {
		return err
	}
	return g.SetEth1Data(&ethpb.Eth1Data{DepositRoot: dr[:], BlockHash: s.GB.Hash().Bytes()})
}

func emptyDepositRoot() ([32]byte, error) {
	t, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	if err != nil {
		return [32]byte{}, err
	}
	return t.HashTreeRoot()
}

func (s *premineGenesisConfig) populate(g state.BeaconState) error {
	if err := g.SetGenesisTime(s.GenesisTime); err != nil {
		return err
	}
	if err := s.setGenesisValidatorsRoot(g); err != nil {
		return err
	}
	if err := s.setFork(g); err != nil {
		return err
	}
	rao := nSetRoots(uint64(params.BeaconConfig().EpochsPerHistoricalVector), s.GB.Hash().Bytes())
	if err := g.SetRandaoMixes(rao); err != nil {
		return err
	}
	if err := g.SetBlockRoots(nZeroRoots(uint64(params.BeaconConfig().SlotsPerHistoricalRoot))); err != nil {
		return err
	}
	if err := g.SetStateRoots(nZeroRoots(uint64(params.BeaconConfig().SlotsPerHistoricalRoot))); err != nil {
		return err
	}
	if err := g.SetSlashings(make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)); err != nil {
		return err
	}
	if err := s.setLatestBlockHeader(g); err != nil {
		return err
	}
	if err := s.setInactivityScores(g); err != nil {
		return err
	}
	if err := s.setSyncCommittees(g); err != nil {
		return err
	}
	if err := s.setExecutionPayload(g); err != nil {
		return err
	}

	// For pre-mined genesis, we want to keep the deposit root set to the root of an empty trie.
	// This needs to be set again because the methods used by processDeposits mutate the state's eth1data.
	return s.setEth1Data(g)
}

func (s *premineGenesisConfig) setGenesisValidatorsRoot(g state.BeaconState) error {
	vroot, err := stateutil.ValidatorRegistryRoot(g.Validators())
	if err != nil {
		return err
	}
	return g.SetGenesisValidatorsRoot(vroot[:])
}

func (s *premineGenesisConfig) setFork(g state.BeaconState) error {
	var pv, cv []byte
	switch s.Version {
	case version.Phase0:
		pv, cv = params.BeaconConfig().GenesisForkVersion, params.BeaconConfig().GenesisForkVersion
	case version.Altair:
		pv, cv = params.BeaconConfig().GenesisForkVersion, params.BeaconConfig().AltairForkVersion
	case version.Bellatrix:
		pv, cv = params.BeaconConfig().AltairForkVersion, params.BeaconConfig().BellatrixForkVersion
	default:
		return errUnsupportedVersion
	}
	fork := &ethpb.Fork{
		PreviousVersion: pv,
		CurrentVersion:  cv,
		Epoch:           0,
	}
	return g.SetFork(fork)
}

func (s *premineGenesisConfig) setInactivityScores(g state.BeaconState) error {
	if s.Version < version.Altair {
		return nil
	}

	scores, err := g.InactivityScores()
	if err != nil {
		return err
	}
	scoresMissing := len(g.Validators()) - len(scores)
	if scoresMissing > 0 {
		for i := 0; i < scoresMissing; i++ {
			scores = append(scores, 0)
		}
	}
	return g.SetInactivityScores(scores)
}

func (s *premineGenesisConfig) setSyncCommittees(g state.BeaconState) error {
	if s.Version < version.Altair {
		return nil
	}
	sc, err := altair.NextSyncCommittee(context.Background(), g)
	if err != nil {
		return err
	}
	if err = g.SetNextSyncCommittee(sc); err != nil {
		return err
	}
	return g.SetCurrentSyncCommittee(sc)
}

type rooter interface {
	HashTreeRoot() ([32]byte, error)
}

func (s *premineGenesisConfig) setLatestBlockHeader(g state.BeaconState) error {
	var body rooter
	switch s.Version {
	case version.Phase0:
		body = &ethpb.BeaconBlockBody{
			RandaoReveal: make([]byte, 96),
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: make([]byte, 32),
				BlockHash:   make([]byte, 32),
			},
			Graffiti: make([]byte, 32),
		}
	case version.Altair:
		body = &ethpb.BeaconBlockBodyAltair{
			RandaoReveal: make([]byte, 96),
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: make([]byte, 32),
				BlockHash:   make([]byte, 32),
			},
			Graffiti: make([]byte, 32),
			SyncAggregate: &ethpb.SyncAggregate{
				SyncCommitteeBits:      make([]byte, fieldparams.SyncCommitteeLength/8),
				SyncCommitteeSignature: make([]byte, fieldparams.BLSSignatureLength),
			},
		}
	case version.Bellatrix:
		body = &ethpb.BeaconBlockBodyBellatrix{
			RandaoReveal: make([]byte, 96),
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: make([]byte, 32),
				BlockHash:   make([]byte, 32),
			},
			Graffiti: make([]byte, 32),
			SyncAggregate: &ethpb.SyncAggregate{
				SyncCommitteeBits:      make([]byte, fieldparams.SyncCommitteeLength/8),
				SyncCommitteeSignature: make([]byte, fieldparams.BLSSignatureLength),
			},
			ExecutionPayload: &enginev1.ExecutionPayload{
				ParentHash:    make([]byte, 32),
				FeeRecipient:  make([]byte, 20),
				StateRoot:     make([]byte, 32),
				ReceiptsRoot:  make([]byte, 32),
				LogsBloom:     make([]byte, 256),
				PrevRandao:    make([]byte, 32),
				BaseFeePerGas: make([]byte, 32),
				BlockHash:     make([]byte, 32),
				Transactions:  make([][]byte, 0),
			},
		}
	default:
		return errUnsupportedVersion
	}

	root, err := body.HashTreeRoot()
	if err != nil {
		return errors.Wrap(err, "could not hash tree root empty block body")
	}
	lbh := &ethpb.BeaconBlockHeader{
		ParentRoot: params.BeaconConfig().ZeroHash[:],
		StateRoot:  params.BeaconConfig().ZeroHash[:],
		BodyRoot:   root[:],
	}
	return g.SetLatestBlockHeader(lbh)
}

func (s *premineGenesisConfig) setExecutionPayload(g state.BeaconState) error {
	if s.Version < version.Bellatrix {
		return nil
	}
	gb := s.GB

	payload := &enginev1.ExecutionPayload{
		ParentHash:    gb.ParentHash().Bytes(),
		FeeRecipient:  gb.Coinbase().Bytes(),
		StateRoot:     gb.Root().Bytes(),
		ReceiptsRoot:  gb.ReceiptHash().Bytes(),
		LogsBloom:     gb.Bloom().Bytes(),
		PrevRandao:    params.BeaconConfig().ZeroHash[:],
		BlockNumber:   gb.NumberU64(),
		GasLimit:      gb.GasLimit(),
		GasUsed:       gb.GasUsed(),
		Timestamp:     gb.Time(),
		ExtraData:     gb.Extra()[:32],
		BaseFeePerGas: bytesutil.PadTo(bytesutil.ReverseByteOrder(gb.BaseFee().Bytes()), fieldparams.RootLength),
		BlockHash:     gb.Hash().Bytes(),
		Transactions:  make([][]byte, 0),
	}
	wep, err := blocks.WrappedExecutionPayload(payload)
	if err != nil {
		return err
	}
	eph, err := blocks.PayloadToHeader(wep)
	if err != nil {
		return err
	}
	wh, err := blocks.WrappedExecutionPayloadHeader(eph)
	if err != nil {
		return err
	}
	return g.SetLatestExecutionPayloadHeader(wh)
}

func nZeroRoots(n uint64) [][]byte {
	roots := make([][]byte, n)
	zh := params.BeaconConfig().ZeroHash[:]
	for i := uint64(0); i < n; i++ {
		roots[i] = zh
	}
	return roots
}

func nSetRoots(n uint64, r []byte) [][]byte {
	roots := make([][]byte, n)
	for i := uint64(0); i < n; i++ {
		h := make([]byte, 32)
		copy(h, r)
		roots[i] = h
	}
	return roots
}
