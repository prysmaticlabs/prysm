package validator_service_config

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/validator"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func Test_Proposer_Setting_Cloning(t *testing.T) {
	key1, err := hexutil.Decode("0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a")
	require.NoError(t, err)
	settings := &ProposerSettings{
		ProposeConfig: map[[fieldparams.BLSPubkeyLength]byte]*ProposerOption{
			bytesutil.ToBytes48(key1): {
				FeeRecipientConfig: &FeeRecipientConfig{
					FeeRecipient: common.HexToAddress("0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3"),
				},
				BuilderConfig: &BuilderConfig{
					Enabled:  true,
					GasLimit: validator.Uint64(40000000),
					Relays:   []string{"https://example-relay.com"},
				},
			},
		},
		DefaultConfig: &ProposerOption{
			FeeRecipientConfig: &FeeRecipientConfig{
				FeeRecipient: common.HexToAddress("0x6e35733c5af9B61374A128e6F85f553aF09ff89A"),
			},
			BuilderConfig: &BuilderConfig{
				Enabled:  false,
				GasLimit: validator.Uint64(params.BeaconConfig().DefaultBuilderGasLimit),
				Relays:   []string{"https://example-relay.com"},
			},
		},
	}
	t.Run("Happy Path Cloning", func(t *testing.T) {
		clone := settings.Clone()
		require.DeepEqual(t, settings, clone)
		option, ok := settings.ProposeConfig[bytesutil.ToBytes48(key1)]
		require.Equal(t, true, ok)
		newFeeRecipient := "0x44455530FCE8a85ec7055A5F8b2bE214B3DaeFd3"
		option.FeeRecipientConfig.FeeRecipient = common.HexToAddress(newFeeRecipient)
		coption, k := clone.ProposeConfig[bytesutil.ToBytes48(key1)]
		require.Equal(t, true, k)
		require.NotEqual(t, option.FeeRecipientConfig.FeeRecipient.Hex(), coption.FeeRecipientConfig.FeeRecipient.Hex())
		require.Equal(t, "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3", coption.FeeRecipientConfig.FeeRecipient.Hex())
	})
}
