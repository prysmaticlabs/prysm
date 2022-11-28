//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

func TestGetDomainData_ValidDomainData(t *testing.T) {
	const genesisValidatorRoot = "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
	forkVersion := params.BeaconConfig().AltairForkVersion
	epoch := params.BeaconConfig().AltairForkEpoch
	domainType := params.BeaconConfig().DomainBeaconProposer

	genesisValidatorRootBytes, err := hexutil.Decode(genesisValidatorRoot)
	require.NoError(t, err)

	expectedForkDataRoot, err := (&ethpb.ForkData{
		CurrentVersion:        forkVersion,
		GenesisValidatorsRoot: genesisValidatorRootBytes,
	}).HashTreeRoot()
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Make sure that GetGenesis() is called exactly once
	genesisProvider := mock.NewMockgenesisProvider(ctrl)
	genesisProvider.EXPECT().GetGenesis().Return(
		&rpcmiddleware.GenesisResponse_GenesisJson{GenesisValidatorsRoot: genesisValidatorRoot},
		nil,
		nil,
	).Times(1)

	// Make sure that GetForkVersion() is called exactly once and with the right arguments
	forkVersionProvider := mock.NewMockforkVersionProvider(ctrl)
	var forkVersionArray [4]byte
	copy(forkVersionArray[:], forkVersion)
	forkVersionProvider.EXPECT().GetForkVersion(
		epoch,
	).Return(
		forkVersionArray,
		nil,
	).Times(1)

	domainDataProvider := &beaconApiDomainDataProvider{
		genesisProvider:     genesisProvider,
		forkVersionProvider: forkVersionProvider,
	}

	resp, err := domainDataProvider.GetDomainData(epoch, domainType[:])
	assert.NoError(t, err)
	require.NotNil(t, resp)

	var expectedSignatureDomain []byte
	expectedSignatureDomain = append(expectedSignatureDomain, domainType[:]...)
	expectedSignatureDomain = append(expectedSignatureDomain, expectedForkDataRoot[:28]...)

	assert.Equal(t, len(expectedSignatureDomain), len(resp.SignatureDomain))
	assert.DeepEqual(t, expectedSignatureDomain, resp.SignatureDomain)
}

func TestGetDomainData_InvalidDomainType(t *testing.T) {
	domainType := make([]byte, 3)
	domainDataProvider := &beaconApiDomainDataProvider{}
	_, err := domainDataProvider.GetDomainData(1, domainType)
	assert.ErrorContains(t, fmt.Sprintf("invalid domain type: %s", hexutil.Encode(domainType)), err)
}

func TestGetDomainData_ForkVersionError(t *testing.T) {
	// Mock the GetForkVersion() call
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	forkVersionProvider := mock.NewMockforkVersionProvider(ctrl)

	epoch := params.BeaconConfig().AltairForkEpoch

	// Make sure that GetForkVersion() is called exactly once and with the right arguments
	forkVersionProvider.EXPECT().GetForkVersion(
		epoch,
	).Return(
		[4]byte{},
		errors.New(""),
	).Times(1)

	domainDataProvider := &beaconApiDomainDataProvider{
		forkVersionProvider: forkVersionProvider,
	}

	_, err := domainDataProvider.GetDomainData(epoch, make([]byte, 4))
	assert.ErrorContains(t, fmt.Sprintf("failed to get fork version for epoch %d", epoch), err)
}

func TestGetDomainData_GenesisError(t *testing.T) {
	const genesisValidatorRoot = "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
	forkVersion := params.BeaconConfig().AltairForkVersion
	epoch := params.BeaconConfig().AltairForkEpoch
	domainType := params.BeaconConfig().DomainBeaconProposer

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Make sure that GetForkVersion() is called exactly once and with the right arguments
	forkVersionProvider := mock.NewMockforkVersionProvider(ctrl)
	var forkVersionArray [4]byte
	copy(forkVersionArray[:], forkVersion)
	forkVersionProvider.EXPECT().GetForkVersion(
		epoch,
	).Return(
		forkVersionArray,
		nil,
	).Times(1)

	// Make sure that GetGenesis() is called exactly once
	genesisProvider := mock.NewMockgenesisProvider(ctrl)
	genesisProvider.EXPECT().GetGenesis().Return(nil, nil, errors.New("")).Times(1)

	domainDataProvider := &beaconApiDomainDataProvider{
		genesisProvider:     genesisProvider,
		forkVersionProvider: forkVersionProvider,
	}

	_, err := domainDataProvider.GetDomainData(epoch, domainType[:])
	assert.ErrorContains(t, "failed to get genesis info", err)
}

func TestGetDomainData_InvalidGenesisRoot(t *testing.T) {
	const genesisValidatorRoot = "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
	forkVersion := params.BeaconConfig().AltairForkVersion
	epoch := params.BeaconConfig().AltairForkEpoch
	domainType := params.BeaconConfig().DomainBeaconProposer

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Make sure that GetForkVersion() is called exactly once and with the right arguments
	forkVersionProvider := mock.NewMockforkVersionProvider(ctrl)
	var forkVersionArray [4]byte
	copy(forkVersionArray[:], forkVersion)
	forkVersionProvider.EXPECT().GetForkVersion(
		epoch,
	).Return(
		forkVersionArray,
		nil,
	).Times(1)

	// Make sure that GetGenesis() is called exactly once
	genesisProvider := mock.NewMockgenesisProvider(ctrl)
	genesisProvider.EXPECT().GetGenesis().Return(
		&rpcmiddleware.GenesisResponse_GenesisJson{GenesisValidatorsRoot: "foo"},
		nil,
		nil,
	).Times(1)

	domainDataProvider := &beaconApiDomainDataProvider{
		genesisProvider:     genesisProvider,
		forkVersionProvider: forkVersionProvider,
	}

	_, err := domainDataProvider.GetDomainData(epoch, domainType[:])
	assert.ErrorContains(t, "invalid genesis validators root: foo", err)
}
