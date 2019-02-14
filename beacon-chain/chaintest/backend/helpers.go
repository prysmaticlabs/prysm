package backend

import (
	"fmt"
	"strconv"
	"time"

	"github.com/prysmaticlabs/prysm/shared/ssz"

	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

// Generates a simulated beacon block to use
// in the next state transition given the current state,
// the previous beacon block, and previous beacon block root.
func generateSimulatedBlock(
	beaconState *pb.BeaconState,
	prevBlockRoot [32]byte,
	depositsTrie *trieutil.DepositTrie,
	simObjects *SimulatedObjects,
) (*pb.BeaconBlock, [32]byte, error) {
	stateRoot, err := ssz.TreeHash(beaconState)
	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("could not tree hash state: %v", err)
	}
	randaoReveal := [32]byte{}
	block := &pb.BeaconBlock{
		Slot:               beaconState.Slot + 1,
		RandaoRevealHash32: randaoReveal[:],
		ParentRootHash32:   prevBlockRoot[:],
		StateRootHash32:    stateRoot[:],
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{1},
			BlockHash32:       []byte{2},
		},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: []*pb.ProposerSlashing{},
			AttesterSlashings: []*pb.AttesterSlashing{},
			Attestations:      []*pb.Attestation{},
			Deposits:          []*pb.Deposit{},
			Exits:             []*pb.Exit{},
		},
	}
	if simObjects.simDeposit != nil {
		depositInput := &pb.DepositInput{
			Pubkey:                      []byte(simObjects.simDeposit.Pubkey),
			WithdrawalCredentialsHash32: make([]byte, 32),
			ProofOfPossession:           make([]byte, 96),
		}

		data, err := b.EncodeDepositData(depositInput, simObjects.simDeposit.Amount, time.Now().Unix())
		if err != nil {
			return nil, [32]byte{}, fmt.Errorf("could not encode deposit data: %v", err)
		}

		// We then update the deposits Merkle trie with the deposit data and return
		// its Merkle branch leading up to the root of the trie.
		depositsTrie.UpdateDepositTrie(data)
		merkleBranch := depositsTrie.Branch()

		block.Body.Deposits = append(block.Body.Deposits, &pb.Deposit{
			DepositData:         data,
			MerkleBranchHash32S: merkleBranch,
			MerkleTreeIndex:     simObjects.simDeposit.MerkleIndex,
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
		block.Body.Exits = append(block.Body.Exits, &pb.Exit{
			Epoch:          simObjects.simValidatorExit.Epoch,
			ValidatorIndex: simObjects.simValidatorExit.ValidatorIndex,
		})
	}
	blockRoot, err := ssz.TreeHash(block)
	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("could not tree hash new block: %v", err)
	}
	return block, blockRoot, nil
}

// Generates initial deposits for creating a beacon state in the simulated
// backend based on the yaml configuration.
func generateInitialSimulatedDeposits(numDeposits uint64) ([]*pb.Deposit, error) {
	genesisTime := time.Date(2018, 9, 0, 0, 0, 0, 0, time.UTC).Unix()
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		depositInput := &pb.DepositInput{
			Pubkey:                      []byte(strconv.Itoa(i)),
			WithdrawalCredentialsHash32: make([]byte, 32),
			ProofOfPossession:           make([]byte, 96),
		}
		depositData, err := b.EncodeDepositData(
			depositInput,
			params.BeaconConfig().MaxDepositAmount,
			genesisTime,
		)
		if err != nil {
			return nil, fmt.Errorf("could not encode initial block deposits: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
	}
	return deposits, nil
}
