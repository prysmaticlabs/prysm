package structs

import (
	"encoding/json"
	"testing"

	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/require"
	"github.com/OffchainLabs/prysm/v7/testing/util"
)

func TestBeaconStateHezeFromConsensus(t *testing.T) {
	for _, nilBid := range []bool{false, true} {
		name := "bid"
		if nilBid {
			name = "nil bid"
		}
		t.Run(name, func(t *testing.T) {
			state, err := util.NewBeaconStateHeze(func(s *ethpb.BeaconStateHeze) error {
				if nilBid {
					s.LatestExecutionPayloadBid = nil
				} else {
					s.LatestExecutionPayloadBid.InclusionListBits = []byte{0x01, 0x80}
					s.LatestExecutionPayloadBid.GasLimit = 123
				}
				return nil
			})
			require.NoError(t, err)
			converted, err := BeaconStateHezeFromConsensus(state)
			require.NoError(t, err)
			encoded, err := json.Marshal(converted)
			require.NoError(t, err)
			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(encoded, &fields))
			if nilBid {
				require.Equal(t, "null", string(fields["latest_execution_payload_bid"]))
				return
			}
			var bid map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(fields["latest_execution_payload_bid"], &bid))
			require.Equal(t, `"0x0180"`, string(bid["inclusion_list_bits"]))
			require.Equal(t, `"123"`, string(bid["gas_limit"]))
		})
	}
}
