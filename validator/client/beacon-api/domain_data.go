//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

type domainDataProvider interface {
	GetDomainData(epoch types.Epoch, domainType []byte) (*ethpb.DomainResponse, error)
}

type beaconApiDomainDataProvider struct {
	genesisProvider genesisProvider
}

func (c beaconApiDomainDataProvider) GetDomainData(epoch types.Epoch, domainType []byte) (*ethpb.DomainResponse, error) {
	if len(domainType) != 4 {
		return nil, errors.Errorf("invalid domain type: %s", hexutil.Encode(domainType))
	}

	// Get the fork version from the given epoch
	forkVersion, err := getForkVersion(epoch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get fork version for epoch %d", epoch)
	}

	// Get the genesis validator root
	genesis, _, err := c.genesisProvider.GetGenesis()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get genesis info")
	}

	if !validRoot(genesis.GenesisValidatorsRoot) {
		return nil, errors.Errorf("invalid genesis validators root: %s", genesis.GenesisValidatorsRoot)
	}

	genesisValidatorRoot, err := hexutil.Decode(genesis.GenesisValidatorsRoot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode genesis validators root")
	}

	// Compute the hash tree root of the fork version and the genesis validator root
	forkDataRoot, err := (&ethpb.ForkData{
		CurrentVersion:        forkVersion[:],
		GenesisValidatorsRoot: genesisValidatorRoot,
	}).HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to hash the fork data")
	}

	// Append the last 28 bytes of the fork data root to the domain type
	signatureDomain := make([]byte, 0, 32)
	signatureDomain = append(signatureDomain, domainType...)
	signatureDomain = append(signatureDomain, forkDataRoot[:28]...)

	return &ethpb.DomainResponse{SignatureDomain: signatureDomain}, nil
}
