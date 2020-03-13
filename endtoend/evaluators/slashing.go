package evaluators

import (
	"context"

	"github.com/gogo/protobuf/types"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"google.golang.org/grpc"
)

// InjectDoubleVote broadcasts a double vote for the slasher to detect.
var InjectDoubleVote = Evaluator{
	Name:       "inject_double_vote_%d",
	Policy:     beforeEpoch(3),
	Evaluation: insertDoubleAttestationIntoPool,
}

// InjectSurroundVote broadcasts a surround vote for the slasher to detect.
var InjectSurroundVote = Evaluator{
	Name:       "inject_surround_vote_%d",
	Policy:     beforeEpoch(3),
	Evaluation: insertSurroundAttestationIntoPool,
}

// Not including first epoch because of issues with genesis.
func beforeEpoch(epoch uint64) func(uint64) bool {
	return func(currentEpoch uint64) bool {
		return currentEpoch < epoch
	}
}

func insertDoubleAttestationIntoPool(conn *grpc.ClientConn) error {
	valClient := eth.NewBeaconNodeValidatorClient(conn)
	beaconClient := eth.NewBeaconChainClient(conn)

	ctx := context.Background()
	chainHead, err := beaconClient.GetChainHead(ctx, &types.Empty{})
	if err != nil {
		return err
	}

	_, privKeys, err := testutil.DeterministicDepositsAndKeys(64)
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
		return err
	}

	var committeeIndex uint64
	var committee []uint64
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

	dataRoot, err := ssz.HashTreeRoot(attData)
	if err != nil {
		return err
	}

	domainResp, err := valClient.DomainData(ctx, &eth.DomainRequest{
		Epoch:  attData.Target.Epoch,
		Domain: params.BeaconConfig().DomainBeaconAttester[:],
	})
	if err != nil {
		return err
	}

	for i := uint64(0); i < 4; i++ {
		// Set the bits of half the committee to be slashed.
		attBitfield := bitfield.NewBitlist(uint64(len(committee)))
		attBitfield.SetBitAt(i, true)

		att := &eth.Attestation{
			AggregationBits: attBitfield,
			Data:            attData,
			Signature:       privKeys[committee[i]].Sign(dataRoot[:], domainResp.SignatureDomain).Marshal(),
		}
		_, err = valClient.ProposeAttestation(ctx, att)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertSurroundAttestationIntoPool(conn *grpc.ClientConn) error {
	valClient := eth.NewBeaconNodeValidatorClient(conn)
	beaconClient := eth.NewBeaconChainClient(conn)

	ctx := context.Background()
	chainHead, err := beaconClient.GetChainHead(ctx, &types.Empty{})
	if err != nil {
		return err
	}

	_, privKeys, err := testutil.DeterministicDepositsAndKeys(64)
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
		return err
	}

	var committeeIndex uint64
	var committee []uint64
	for _, duty := range duties.Duties {
		if duty.AttesterSlot == startSlot(chainHead.HeadEpoch) {
			committeeIndex = duty.CommitteeIndex
			committee = duty.Committee
			break
		}
	}

	// Set the bits of half the committee to be slashed.
	attBitfield := bitfield.NewBitlist(uint64(len(committee)))
	attBitfield.SetBitAt(1, true)

	attDataReq := &eth.AttestationDataRequest{
		CommitteeIndex: committeeIndex,
		Slot:           startSlot(chainHead.HeadEpoch),
	}
	attData, err := valClient.GetAttestationData(ctx, attDataReq)
	if err != nil {
		return err
	}
	attData.Source.Epoch -= 1
	attData.Target.Epoch += 1
	dataRoot, err := ssz.HashTreeRoot(attData)
	if err != nil {
		return err
	}

	domainResp, err := valClient.DomainData(ctx, &eth.DomainRequest{
		Epoch:  attData.Target.Epoch + 1,
		Domain: params.BeaconConfig().DomainBeaconAttester[:],
	})
	if err != nil {
		return err
	}

	att := &eth.Attestation{
		AggregationBits: attBitfield,
		Data:            attData,
		Signature:       privKeys[committee[1]].Sign(dataRoot[:], domainResp.SignatureDomain).Marshal(),
	}
	_, err = valClient.ProposeAttestation(ctx, att)
	if err != nil {
		return err
	}
	return nil
}

func startSlot(epoch uint64) uint64 {
	return epoch * params.BeaconConfig().SlotsPerEpoch
}
