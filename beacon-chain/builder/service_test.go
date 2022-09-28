package builder

import (
	"context"
	"testing"

	buildertesting "github.com/prysmaticlabs/prysm/v3/api/client/builder/testing"
	blockchainTesting "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	dbtesting "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func Test_NewServiceWithBuilder(t *testing.T) {
	s, err := NewService(context.Background(), WithBuilderClient(&buildertesting.MockClient{}))
	require.NoError(t, err)
	assert.Equal(t, true, s.Configured())
}

func Test_NewServiceWithoutBuilder(t *testing.T) {
	s, err := NewService(context.Background())
	require.NoError(t, err)
	assert.Equal(t, false, s.Configured())
}

func Test_RegisterValidator(t *testing.T) {
	ctx := context.Background()
	db := dbtesting.SetupDB(t)
	headFetcher := &blockchainTesting.ChainService{}
	builder := buildertesting.NewClient()
	s, err := NewService(ctx, WithDatabase(db), WithHeadFetcher(headFetcher), WithBuilderClient(&builder))
	require.NoError(t, err)
	pubkey := bytesutil.ToBytes48([]byte("pubkey"))
	var feeRecipient [20]byte
	require.NoError(t, s.RegisterValidator(ctx, []*eth.SignedValidatorRegistrationV1{{Message: &eth.ValidatorRegistrationV1{Pubkey: pubkey[:], FeeRecipient: feeRecipient[:]}}}))
	assert.Equal(t, true, builder.RegisteredVals[pubkey])
}
