package evaluators

import (
	"bytes"
	"context"
	"fmt"

	"github.com/gogo/protobuf/types"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"google.golang.org/grpc"
)

func InsertDoubleAttestationIntoPool(conn *grpc.ClientConn) error {
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

	// Set the bits of half the committee to be slashed.
	attBitfield := bitfield.NewBitlist(uint64(len(committee)))
	attBitfield.SetBitAt(0, true)

	attDataReq := &eth.AttestationDataRequest{
		CommitteeIndex: committeeIndex,
		Slot:           chainHead.HeadSlot - 1,
	}
	attData, err := valClient.GetAttestationData(ctx, attDataReq)
	if err != nil {
		return err
	}
	attData.BeaconBlockRoot = []byte("muahahahaha I'm an evil validator")
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

	att := &eth.Attestation{
		AggregationBits: attBitfield,
		Data:            attData,
		Signature:       privKeys[committee[0]].Sign(dataRoot[:], domainResp.SignatureDomain).Marshal(),
	}
	attResp, err := valClient.ProposeAttestation(ctx, att)
	if err != nil {
		return err
	}
	fmt.Println(attResp)
	fmt.Println(bytes.Equal(attResp.AttestationDataRoot, dataRoot[:]))
	return nil
}
