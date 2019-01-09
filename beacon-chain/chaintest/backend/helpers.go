package backend

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"time"

	"github.com/prysmaticlabs/prysm/shared/ssz"

	"github.com/gogo/protobuf/proto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Generates a simulated beacon block to use
// in the next state transition given the current state,
// the previous beacon block, and previous beacon block root.
func generateSimulatedBlock(
	beaconState *pb.BeaconState,
	prevBlockRoot [32]byte,
	randaoReveal [32]byte,
	simulatedDeposit *StateTestDeposit,
) (*pb.BeaconBlock, [32]byte, error) {
	encodedState, err := proto.Marshal(beaconState)
	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("could not marshal beacon state: %v", err)
	}
	stateRoot := hashutil.Hash(encodedState)
	block := &pb.BeaconBlock{
		Slot:               beaconState.Slot + 1,
		RandaoRevealHash32: randaoReveal[:],
		ParentRootHash32:   prevBlockRoot[:],
		StateRootHash32:    stateRoot[:],
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: []*pb.ProposerSlashing{},
			CasperSlashings:   []*pb.CasperSlashing{},
			Attestations:      []*pb.Attestation{},
			Deposits:          []*pb.Deposit{},
			Exits:             []*pb.Exit{},
		},
	}
	if simulatedDeposit != nil {
		depositInput := &pb.DepositInput{
			Pubkey:                      []byte{},
			WithdrawalCredentialsHash32: []byte{},
			ProofOfPossession:           []byte{},
			// TODO: Fix based on hash onions.
			RandaoCommitmentHash32:  []byte{},
			CustodyCommitmentHash32: []byte{},
		}
		wBuf := new(bytes.Buffer)
		if err := ssz.Encode(wBuf, depositInput); err != nil {
			return nil, [32]byte{}, fmt.Errorf("failed to encode deposit input: %v", err)
		}
		encodedInput := wBuf.Bytes()
		data := []byte{}

		// We set a deposit value of 1000.
		value := make([]byte, 8)
		binary.BigEndian.PutUint64(value, simulatedDeposit.Amount)

		// We then serialize a unix time into the timestamp []byte slice
		// and ensure it has size of 8 bytes.
		timestamp := make([]byte, 8)

		// Set deposit time to 1000 seconds since unix time 0.
		depositTime := time.Now().Unix()
		binary.BigEndian.PutUint64(timestamp, uint64(depositTime))

		// We then create a serialized deposit data slice of type []byte
		// by appending all 3 items above together.
		data = append(data, encodedInput...)
		data = append(data, value...)
		data = append(data, timestamp...)

		// We then create a merkle branch for the test and derive its root.
		branch := [][]byte{}
		var powReceiptRoot [32]byte
		copy(powReceiptRoot[:], data)
		for i := uint64(0); i < params.BeaconConfig().DepositContractTreeDepth; i++ {
			branch = append(branch, []byte{1})
			if i%2 == 0 {
				powReceiptRoot = hashutil.Hash(append(branch[i], powReceiptRoot[:]...))
			} else {
				powReceiptRoot = hashutil.Hash(append(powReceiptRoot[:], branch[i]...))
			}
		}

		block.Body.Deposits = append(block.Body.Deposits, &pb.Deposit{
			DepositData:         data,
			MerkleBranchHash32S: branch,
		})
	}
	encodedBlock, err := proto.Marshal(block)
	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("could not marshal new block: %v", err)
	}
	return block, hashutil.Hash(encodedBlock), nil
}

// Given a number of slots, we create a list of hash onions from an underlying randao reveal. For example,
// if we have N slots, we create a list of [secret, hash(secret), hash(hash(secret)), hash(...(prev N-1 hashes))].
func generateSimulatedRandaoHashOnions(numSlots uint64) [][32]byte {
	// We create a list of randao hash onions for the given number of epochs
	// we run the state transition.
	numEpochs := numSlots % params.BeaconConfig().EpochLength
	hashOnions := [][32]byte{params.BeaconConfig().SimulatedBlockRandao}

	// We make the length of the hash onions list equal to the number of epochs + 10 to be safe.
	for i := uint64(0); i < numEpochs+10; i++ {
		prevHash := hashOnions[i]
		hashOnions = append(hashOnions, hashutil.Hash(prevHash[:]))
	}
	return hashOnions
}

// This function determines the block randao reveal assuming there are no skipped slots,
// given a list of randao hash onions such as [pre-image, 0x01, 0x02, 0x03], for the
// 0th epoch, the block randao reveal will be 0x02 and the proposer commitment 0x03.
// The next epoch, the block randao reveal will be 0x01 and the commitment 0x02,
// so on and so forth until all randao layers are peeled off.
func determineSimulatedBlockRandaoReveal(layersPeeled int, hashOnions [][32]byte) [32]byte {
	if layersPeeled == 0 {
		return hashOnions[len(hashOnions)-2]
	}
	return hashOnions[len(hashOnions)-layersPeeled-2]
}

// Generates initial deposits for creating a beacon state in the simulated
// backend based on the yaml configuration.
func generateInitialSimulatedDeposits(randaoCommit [32]byte) ([]*pb.Deposit, error) {
	genesisTime := params.BeaconConfig().GenesisTime.Unix()
	deposits := make([]*pb.Deposit, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(deposits); i++ {
		depositInput := &pb.DepositInput{
			Pubkey:                 []byte(strconv.Itoa(i)),
			RandaoCommitmentHash32: randaoCommit[:],
		}
		depositData, err := b.EncodeDepositData(
			depositInput,
			params.BeaconConfig().MaxDepositInGwei,
			genesisTime,
		)
		if err != nil {
			return nil, fmt.Errorf("could not encode initial block deposits: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
	}
	return deposits, nil
}

// Finds the index of the next slot's proposer in the beacon state's
// validator set.
func findNextSlotProposerIndex(beaconState *pb.BeaconState) (uint32, error) {
	nextSlot := beaconState.Slot + 1
	epochLength := params.BeaconConfig().EpochLength
	var earliestSlot uint64

	// If the state slot is less than epochLength, then the earliestSlot would
	// result in a negative number. Therefore we should default to
	// earliestSlot = 0 in this case.
	if nextSlot > epochLength {
		earliestSlot = nextSlot - (nextSlot % epochLength) - epochLength
	}

	if nextSlot < earliestSlot || nextSlot >= earliestSlot+(epochLength*2) {
		return 0, fmt.Errorf("slot %d out of bounds: %d <= slot < %d",
			nextSlot,
			earliestSlot,
			earliestSlot+(epochLength*2),
		)
	}
	committeeArray := beaconState.ShardCommitteesAtSlots[nextSlot-earliestSlot]
	firstCommittee := committeeArray.ArrayShardCommittee[0].Committee
	return firstCommittee[nextSlot%uint64(len(firstCommittee))], nil
}
