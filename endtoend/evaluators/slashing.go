package evaluators

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"google.golang.org/grpc"
)

// InjectDoubleVote broadcasts a double vote into the beacon node pool for the slasher to detect.
var InjectDoubleVote = types.Evaluator{
	Name:       "inject_double_vote_%d",
	Policy:     beforeEpoch(2),
	Evaluation: insertDoubleAttestationIntoPool,
}

var slashedIndices []uint64

// Not including first epoch because of issues with genesis.
func beforeEpoch(epoch uint64) func(uint64) bool {
	return func(currentEpoch uint64) bool {
		return currentEpoch < epoch
	}
}

func insertDoubleAttestationIntoPool(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	valClient := eth.NewBeaconNodeValidatorClient(conn)
	beaconClient := eth.NewBeaconChainClient(conn)

	ctx := context.Background()
	chainHead, err := beaconClient.GetChainHead(ctx, &ptypes.Empty{})
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

	req := &eth.DomainRequest{
		Epoch:  chainHead.HeadEpoch,
		Domain: params.BeaconConfig().DomainBeaconAttester[:],
	}
	resp, err := valClient.DomainData(ctx, req)
	if err != nil {
		return err
	}
	signingRoot, err := helpers.ComputeSigningRoot(attData, resp.SignatureDomain)
	if err != nil {
		return err
	}

	valsToSlash := uint64(2)
	for i := uint64(0); i < valsToSlash && i < uint64(len(committee)); i++ {
		if len(sliceutil.IntersectionUint64(slashedIndices, []uint64{committee[i]})) > 0 {
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
		for _, conn := range conns {
			client := eth.NewBeaconNodeValidatorClient(conn)
			_, err = client.ProposeAttestation(ctx, att)
			if err != nil {
				return err
			}
		}
		slashedIndices = append(slashedIndices, committee[i])
	}
	return nil
}
