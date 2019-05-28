package backend

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

// Generates a simulated beacon block to use
// in the next state transition given the current state,
// the previous beacon block, and previous beacon block root.
func generateSimulatedBlock(
	beaconState *pb.BeaconState,
	prevBlockRoot [32]byte,
	historicalDeposits []*pb.Deposit,
	simObjects *SimulatedObjects,
	privKeys []*bls.SecretKey,
) (*pb.BeaconBlock, [32]byte, error) {
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("could not tree hash state: %v", err)
	}
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		return nil, [32]byte{}, err
	}
	epoch := helpers.SlotToEpoch(beaconState.Slot + 1)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain := helpers.DomainVersion(beaconState, epoch, params.BeaconConfig().DomainRandao)
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx].Sign(buf, domain)
	block := &pb.BeaconBlock{
		Slot:       beaconState.Slot + 1,
		ParentRoot: prevBlockRoot[:],
		StateRoot:  stateRoot[:],
		Body: &pb.BeaconBlockBody{
			Eth1Data: &pb.Eth1Data{
				DepositRoot: []byte{1},
				BlockRoot:   []byte{2},
			},
			RandaoReveal:      epochSignature.Marshal(),
			ProposerSlashings: []*pb.ProposerSlashing{},
			AttesterSlashings: []*pb.AttesterSlashing{},
			Attestations:      []*pb.Attestation{},
			Deposits:          []*pb.Deposit{},
			VoluntaryExits:    []*pb.VoluntaryExit{},
		},
	}
	if simObjects.simDeposit != nil {
		depositData := &pb.DepositData{
			Pubkey:                []byte(simObjects.simDeposit.Pubkey),
			WithdrawalCredentials: make([]byte, 32),
			Signature:             make([]byte, 96),
		}

		data, err := hashutil.DepositHash(depositData)
		if err != nil {
			return nil, [32]byte{}, fmt.Errorf("could not hash deposit: %v", err)
		}

		// We then update the deposits Merkle trie with the deposit data and return
		// its Merkle branch leading up to the root of the trie.
		historicalDepositData := make([][]byte, len(historicalDeposits))
		for i := range historicalDeposits {
			depHash, err := hashutil.DepositHash(historicalDeposits[i].Data)
			if err != nil {
				return nil, [32]byte{}, fmt.Errorf("could not hash deposit item %v", err)
			}
			historicalDepositData[i] = depHash[:]
		}
		newTrie, err := trieutil.GenerateTrieFromItems(append(historicalDepositData, data[:]), int(params.BeaconConfig().DepositContractTreeDepth))
		if err != nil {
			return nil, [32]byte{}, fmt.Errorf("could not regenerate trie: %v", err)
		}
		proof, err := newTrie.MerkleProof(int(simObjects.simDeposit.MerkleIndex))
		if err != nil {
			return nil, [32]byte{}, fmt.Errorf("could not generate proof: %v", err)
		}

		root := newTrie.Root()
		block.Body.Eth1Data.DepositRoot = root[:]
		block.Body.Deposits = append(block.Body.Deposits, &pb.Deposit{
			Data:  depositData,
			Proof: proof,
			Index: simObjects.simDeposit.MerkleIndex,
		})
	}
	if simObjects.simProposerSlashing != nil {
		block.Body.ProposerSlashings = append(block.Body.ProposerSlashings, &pb.ProposerSlashing{
			ProposerIndex: simObjects.simProposerSlashing.ProposerIndex,
			Header_1: &pb.BeaconBlockHeader{
				Slot:     simObjects.simProposerSlashing.Proposal1Slot,
				BodyRoot: []byte(simObjects.simProposerSlashing.Proposal1Root),
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:     simObjects.simProposerSlashing.Proposal2Slot,
				BodyRoot: []byte(simObjects.simProposerSlashing.Proposal2Root),
			},
		})
	}
	if simObjects.simAttesterSlashing != nil {
		block.Body.AttesterSlashings = append(block.Body.AttesterSlashings, &pb.AttesterSlashing{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					SourceEpoch: simObjects.simAttesterSlashing.SlashableAttestation1JustifiedEpoch,
					Crosslink: &pb.Crosslink{
						Shard: simObjects.simAttesterSlashing.SlashableAttestation1Slot,
					},
				},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					SourceEpoch: simObjects.simAttesterSlashing.SlashableAttestation2JustifiedEpoch,
					Crosslink: &pb.Crosslink{
						Shard: simObjects.simAttesterSlashing.SlashableAttestation1Slot,
					},
				},
			},
		})
	}
	if simObjects.simValidatorExit != nil {
		block.Body.VoluntaryExits = append(block.Body.VoluntaryExits, &pb.VoluntaryExit{
			Epoch:          simObjects.simValidatorExit.Epoch,
			ValidatorIndex: simObjects.simValidatorExit.ValidatorIndex,
		})
	}
	blockRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("could not tree hash new block: %v", err)
	}
	return block, blockRoot, nil
}

// generateInitialSimulatedDeposits generates initial deposits for creating a beacon state in the simulated
// backend based on the yaml configuration.
func generateInitialSimulatedDeposits(numDeposits uint64) ([]*pb.Deposit, []*bls.SecretKey, error) {
	deposits := make([]*pb.Deposit, numDeposits)
	privKeys := make([]*bls.SecretKey, numDeposits)
	for i := 0; i < len(deposits); i++ {
		priv, err := bls.RandKey(rand.Reader)
		if err != nil {
			return nil, nil, fmt.Errorf("could not initialize key: %v", err)
		}
		depositData := &pb.DepositData{
			Pubkey:                priv.PublicKey().Marshal(),
			WithdrawalCredentials: make([]byte, 32),
			Signature:             make([]byte, 96),
			Amount:                params.BeaconConfig().MaxDepositAmount,
		}
		deposits[i] = &pb.Deposit{Data: depositData, Index: uint64(i)}
		privKeys[i] = priv
	}
	return deposits, privKeys, nil
}
