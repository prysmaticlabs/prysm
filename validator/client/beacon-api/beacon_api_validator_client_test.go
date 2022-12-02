//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

// Check that the DomainData() returns whatever the internal getDomainData() returns
func TestBeaconApiValidatorClient_DomainDataValid(t *testing.T) {
	const genesisValidatorRoot = "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
	epoch := params.BeaconConfig().AltairForkEpoch
	domainType := params.BeaconConfig().DomainSyncCommittee[:]

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	genesisProvider := mock.NewMockgenesisProvider(ctrl)
	genesisProvider.EXPECT().GetGenesis().Return(
		&rpcmiddleware.GenesisResponse_GenesisJson{GenesisValidatorsRoot: genesisValidatorRoot},
		nil,
		nil,
	).Times(2)

	validatorClient := beaconApiValidatorClient{genesisProvider: genesisProvider}
	resp, err := validatorClient.DomainData(context.Background(), &ethpb.DomainRequest{Epoch: epoch, Domain: domainType})

	domainTypeArray := bytesutil.ToBytes4(domainType)
	expectedResp, expectedErr := validatorClient.getDomainData(epoch, domainTypeArray)
	assert.DeepEqual(t, expectedErr, err)
	assert.DeepEqual(t, expectedResp, resp)
}

func TestBeaconApiValidatorClient_DomainDataError(t *testing.T) {
	epoch := params.BeaconConfig().AltairForkEpoch
	domainType := make([]byte, 3)
	validatorClient := beaconApiValidatorClient{}
	_, err := validatorClient.DomainData(context.Background(), &ethpb.DomainRequest{Epoch: epoch, Domain: domainType})
	assert.ErrorContains(t, fmt.Sprintf("invalid domain type: %s", hexutil.Encode(domainType)), err)
}
