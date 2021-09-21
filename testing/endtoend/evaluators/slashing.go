package evaluators

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	corehelpers "github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/container/slice"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/testing/endtoend/policies"
	e2eTypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// InjectDoubleVote broadcasts a double vote into the beacon node pool for the slasher to detect.
var InjectDoubleVote = e2eTypes.Evaluator{
	Name:       "inject_double_vote_%d",
	Policy:     policies.OnEpoch(1),
	Evaluation: insertDoubleAttestationIntoPool,
}

// ProposeDoubleBlock broadcasts a double block to the beacon node for the slasher to detect.
var ProposeDoubleBlock = e2eTypes.Evaluator{
	Name:       "propose_double_block_%d",
	Policy:     policies.OnEpoch(1),
	Evaluation: proposeDoubleBlock,
}

// ValidatorsSlashed ensures the expected amount of validators are slashed.
var ValidatorsSlashed = e2eTypes.Evaluator{
	Name:       "validators_slashed_epoch_%d",
	Policy:     policies.AfterNthEpoch(1),
	Evaluation: validatorsSlashed,
}

// SlashedValidatorsLoseBalance checks if the validators slashed lose the right balance.
var SlashedValidatorsLoseBalance = e2eTypes.Evaluator{
	Name:       "slashed_validators_lose_valance_epoch_%d",
	Policy:     policies.AfterNthEpoch(1),
	Evaluation: validatorsLoseBalance,
}

var slashedIndices []uint64

func validatorsSlashed(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	ctx := context.Background()
	client := eth.NewBeaconChainClient(conn)
	req := &eth.GetValidatorActiveSetChangesRequest{}
	changes, err := client.GetValidatorActiveSetChanges(ctx, req)
	if err != nil {
		return err
	}
	if len(changes.SlashedIndices) != len(slashedIndices) {
		return fmt.Errorf("expected %d indices to be slashed, received %d", len(slashedIndices), len(changes.SlashedIndices))
	}
	return nil
}

func validatorsLoseBalance(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	ctx := context.Background()
	client := eth.NewBeaconChainClient(conn)

	for i, slashedIndex := range slashedIndices {
		req := &eth.GetValidatorRequest{
			QueryFilter: &eth.GetValidatorRequest_Index{
				Index: types.ValidatorIndex(slashedIndex),
			},
		}
		valResp, err := client.GetValidator(ctx, req)
		if err != nil {
			return err
		}

		slashedPenalty := params.BeaconConfig().MaxEffectiveBalance / params.BeaconConfig().MinSlashingPenaltyQuotient
		slashedBal := params.BeaconConfig().MaxEffectiveBalance - slashedPenalty + params.BeaconConfig().EffectiveBalanceIncrement/10
		if valResp.EffectiveBalance >= slashedBal {
			return fmt.Errorf(
				"expected slashed validator %d to balance less than %d, received %d",
				i,
				slashedBal,
				valResp.EffectiveBalance,
			)
		}

	}
	return nil
}

func insertDoubleAttestationIntoPool(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	valClient := eth.NewBeaconNodeValidatorClient(conn)
	beaconClient := eth.NewBeaconChainClient(conn)

	ctx := context.Background()
	chainHead, err := beaconClient.GetChainHead(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.Wrap(err, "could not get chain head")
	}

	_, privKeys, err := testutil.DeterministicDepositsAndKeys(params.BeaconConfig().MinGenesisActiveValidatorCount)
	if err != nil {
		return err
	}
	pubKeys := make([][]byte, len(privKeys))
	for i, priv := range privKeys {
		pubKeys[i] = priv.PublicKey().Marshal()
	}
	duties, err := valClient.GetDuties(ctx, &eth.DutiesRequest{
		Epoch:      chainHead.HeadEpoch,
		PublicKeys: pubKeys,
	})
	if err != nil {
		return errors.Wrap(err, "could not get duties")
	}

	var committeeIndex types.CommitteeIndex
	var committee []types.ValidatorIndex
	for _, duty := range duties.Duties {
		if duty.AttesterSlot == chainHead.HeadSlot-1 {
			committeeIndex = duty.CommitteeIndex
			committee = duty.Committee
			break
		}
	}

	attDataReq := &eth.AttestationDataRequest{
		CommitteeIndex: committeeIndex,
		Slot:           chainHead.HeadSlot - 1,
	}

	attData, err := valClient.GetAttestationData(ctx, attDataReq)
	if err != nil {
		return err
	}
	blockRoot := bytesutil.ToBytes32([]byte("muahahahaha I'm an evil validator"))
	attData.BeaconBlockRoot = blockRoot[:]

	req := &eth.DomainRequest{
		Epoch:  chainHead.HeadEpoch,
		Domain: params.BeaconConfig().DomainBeaconAttester[:],
	}
	resp, err := valClient.DomainData(ctx, req)
	if err != nil {
		return errors.Wrap(err, "could not get domain data")
	}
	signingRoot, err := corehelpers.ComputeSigningRoot(attData, resp.SignatureDomain)
	if err != nil {
		return errors.Wrap(err, "could not compute signing root")
	}

	valsToSlash := uint64(2)
	for i := uint64(0); i < valsToSlash && i < uint64(len(committee)); i++ {
		if len(slice.IntersectionUint64(slashedIndices, []uint64{uint64(committee[i])})) > 0 {
			valsToSlash++
			continue
		}
		// Set the bits of half the committee to be slashed.
		attBitfield := bitfield.NewBitlist(uint64(len(committee)))
		attBitfield.SetBitAt(i, true)

		att := &eth.Attestation{
			AggregationBits: attBitfield,
			Data:            attData,
			Signature:       privKeys[committee[i]].Sign(signingRoot[:]).Marshal(),
		}
		// We only broadcast to conns[0] here since we can trust that at least 1 node will be online.
		// Only broadcasting the attestation to one node also helps test slashing propagation.
		client := eth.NewBeaconNodeValidatorClient(conns[0])
		if _, err = client.ProposeAttestation(ctx, att); err != nil {
			return errors.Wrap(err, "could not propose attestation")
		}
		slashedIndices = append(slashedIndices, uint64(committee[i]))
	}
	return nil
}

func proposeDoubleBlock(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	valClient := eth.NewBeaconNodeValidatorClient(conn)
	beaconClient := eth.NewBeaconChainClient(conn)

	ctx := context.Background()
	chainHead, err := beaconClient.GetChainHead(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.Wrap(err, "could not get chain head")
	}

	_, privKeys, err := testutil.DeterministicDepositsAndKeys(params.BeaconConfig().MinGenesisActiveValidatorCount)
	if err != nil {
		return err
	}
	pubKeys := make([][]byte, len(privKeys))
	for i, priv := range privKeys {
		pubKeys[i] = priv.PublicKey().Marshal()
	}
	duties, err := valClient.GetDuties(ctx, &eth.DutiesRequest{
		Epoch:      chainHead.HeadEpoch,
		PublicKeys: pubKeys,
	})
	if err != nil {
		return errors.Wrap(err, "could not get duties")
	}

	var proposerIndex types.ValidatorIndex
	for i, duty := range duties.CurrentEpochDuties {
		if slice.IsInSlots(chainHead.HeadSlot-1, duty.ProposerSlots) {
			proposerIndex = types.ValidatorIndex(i)
			break
		}
	}

	hashLen := 32
	blk := &eth.BeaconBlock{
		Slot:          chainHead.HeadSlot - 1,
		ParentRoot:    bytesutil.PadTo([]byte("bad parent root"), hashLen),
		StateRoot:     bytesutil.PadTo([]byte("bad state root"), hashLen),
		ProposerIndex: proposerIndex,
		Body: &eth.BeaconBlockBody{
			Eth1Data: &eth.Eth1Data{
				BlockHash:    bytesutil.PadTo([]byte("bad block hash"), hashLen),
				DepositRoot:  bytesutil.PadTo([]byte("bad deposit root"), hashLen),
				DepositCount: 1,
			},
			RandaoReveal:      bytesutil.PadTo([]byte("bad randao"), params.BeaconConfig().BLSSignatureLength),
			Graffiti:          bytesutil.PadTo([]byte("teehee"), hashLen),
			ProposerSlashings: []*eth.ProposerSlashing{},
			AttesterSlashings: []*eth.AttesterSlashing{},
			Attestations:      []*eth.Attestation{},
			Deposits:          []*eth.Deposit{},
			VoluntaryExits:    []*eth.SignedVoluntaryExit{},
		},
	}

	req := &eth.DomainRequest{
		Epoch:  chainHead.HeadEpoch,
		Domain: params.BeaconConfig().DomainBeaconProposer[:],
	}
	resp, err := valClient.DomainData(ctx, req)
	if err != nil {
		return errors.Wrap(err, "could not get domain data")
	}
	signingRoot, err := corehelpers.ComputeSigningRoot(blk, resp.SignatureDomain)
	if err != nil {
		return errors.Wrap(err, "could not compute signing root")
	}
	sig := privKeys[proposerIndex].Sign(signingRoot[:]).Marshal()
	signedBlk := &eth.SignedBeaconBlock{
		Block:     blk,
		Signature: sig,
	}

	// We only broadcast to conns[0] here since we can trust that at least 1 node will be online.
	// Only broadcasting the attestation to one node also helps test slashing propagation.
	client := eth.NewBeaconNodeValidatorClient(conns[0])
	if _, err = client.ProposeBlock(ctx, signedBlk); err == nil {
		return errors.New("expected block to fail processing")
	}
	slashedIndices = append(slashedIndices, uint64(proposerIndex))
	return nil
}
