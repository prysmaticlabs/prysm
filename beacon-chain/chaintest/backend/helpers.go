package backend

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/forkutil"
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
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState, beaconState.Slot+1)
	if err != nil {
		return nil, [32]byte{}, err
	}
	epoch := helpers.SlotToEpoch(beaconState.Slot + 1)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain := forkutil.DomainVersion(beaconState.Fork, epoch, params.BeaconConfig().DomainRandao)
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx].Sign(buf, domain)
	block := &pb.BeaconBlock{
		Slot:             beaconState.Slot + 1,
		RandaoReveal:     epochSignature.Marshal(),
		ParentRootHash32: prevBlockRoot[:],
		StateRootHash32:  stateRoot[:],
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{1},
			BlockHash32:       []byte{2},
		},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: []*pb.ProposerSlashing{},
			AttesterSlashings: []*pb.AttesterSlashing{},
			Attestations:      []*pb.Attestation{},
			Deposits:          []*pb.Deposit{},
			VoluntaryExits:    []*pb.VoluntaryExit{},
		},
	}
	if simObjects.simDeposit != nil {
		depositInput := &pb.DepositInput{
			Pubkey:                      []byte(simObjects.simDeposit.Pubkey),
			WithdrawalCredentialsHash32: make([]byte, 32),
			ProofOfPossession:           make([]byte, 96),
		}

		data, err := helpers.EncodeDepositData(depositInput, simObjects.simDeposit.Amount, time.Now().Unix())
		if err != nil {
			return nil, [32]byte{}, fmt.Errorf("could not encode deposit data: %v", err)
		}

		// We then update the deposits Merkle trie with the deposit data and return
		// its Merkle branch leading up to the root of the trie.
		historicalDepositData := make([][]byte, len(historicalDeposits))
		for i := range historicalDeposits {
			historicalDepositData[i] = historicalDeposits[i].DepositData
		}
		newTrie, err := trieutil.GenerateTrieFromItems(append(historicalDepositData, data), int(params.BeaconConfig().DepositContractTreeDepth))
		if err != nil {
			return nil, [32]byte{}, fmt.Errorf("could not regenerate trie: %v", err)
		}
		proof, err := newTrie.MerkleProof(int(simObjects.simDeposit.MerkleIndex))
		if err != nil {
			return nil, [32]byte{}, fmt.Errorf("could not generate proof: %v", err)
		}

		root := newTrie.Root()
		block.Eth1Data.DepositRootHash32 = root[:]
		block.Body.Deposits = append(block.Body.Deposits, &pb.Deposit{
			DepositData:        data,
			MerkleProofHash32S: proof,
			MerkleTreeIndex:    simObjects.simDeposit.MerkleIndex,
		})
	}
	if simObjects.simProposerSlashing != nil {
		block.Body.ProposerSlashings = append(block.Body.ProposerSlashings, &pb.ProposerSlashing{
			ProposerIndex: simObjects.simProposerSlashing.ProposerIndex,
			ProposalData_1: &pb.ProposalSignedData{
				Slot:            simObjects.simProposerSlashing.Proposal1Slot,
				Shard:           simObjects.simProposerSlashing.Proposal1Shard,
				BlockRootHash32: []byte(simObjects.simProposerSlashing.Proposal1Root),
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:            simObjects.simProposerSlashing.Proposal2Slot,
				Shard:           simObjects.simProposerSlashing.Proposal2Shard,
				BlockRootHash32: []byte(simObjects.simProposerSlashing.Proposal2Root),
			},
		})
	}
	if simObjects.simAttesterSlashing != nil {
		block.Body.AttesterSlashings = append(block.Body.AttesterSlashings, &pb.AttesterSlashing{
			SlashableAttestation_1: &pb.SlashableAttestation{
				Data: &pb.AttestationData{
					Slot:           simObjects.simAttesterSlashing.SlashableAttestation1Slot,
					JustifiedEpoch: simObjects.simAttesterSlashing.SlashableAttestation1JustifiedEpoch,
				},
				CustodyBitfield:  []byte(simObjects.simAttesterSlashing.SlashableAttestation1CustodyBitField),
				ValidatorIndices: simObjects.simAttesterSlashing.SlashableAttestation1ValidatorIndices,
			},
			SlashableAttestation_2: &pb.SlashableAttestation{
				Data: &pb.AttestationData{
					Slot:           simObjects.simAttesterSlashing.SlashableAttestation2Slot,
					JustifiedEpoch: simObjects.simAttesterSlashing.SlashableAttestation2JustifiedEpoch,
				},
				CustodyBitfield:  []byte(simObjects.simAttesterSlashing.SlashableAttestation2CustodyBitField),
				ValidatorIndices: simObjects.simAttesterSlashing.SlashableAttestation2ValidatorIndices,
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
	genesisTime := time.Date(2018, 9, 0, 0, 0, 0, 0, time.UTC).Unix()
	deposits := make([]*pb.Deposit, numDeposits)
	privKeys := make([]*bls.SecretKey, numDeposits)
	for i := 0; i < len(deposits); i++ {
		priv, err := bls.RandKey(rand.Reader)
		if err != nil {
			return nil, nil, fmt.Errorf("could not initialize key: %v", err)
		}
		depositInput := &pb.DepositInput{
			Pubkey:                      priv.PublicKey().Marshal(),
			WithdrawalCredentialsHash32: make([]byte, 32),
			ProofOfPossession:           make([]byte, 96),
		}
		depositData, err := helpers.EncodeDepositData(
			depositInput,
			params.BeaconConfig().MaxDepositAmount,
			genesisTime,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("could not encode genesis block deposits: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData, MerkleTreeIndex: uint64(i)}
		privKeys[i] = priv
	}
	return deposits, privKeys, nil
}
