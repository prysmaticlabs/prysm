package payloadattribute

import (
	"testing"

	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestPayloadAttributeGetters(t *testing.T) {
	tests := []struct {
		name string
		tc   func(t *testing.T)
	}{
		{
			name: "Get version",
			tc: func(t *testing.T) {
				a := EmptyWithVersion(version.Capella)
				require.Equal(t, version.Capella, a.Version())
			},
		},
		{
			name: "Get prev randao",
			tc: func(t *testing.T) {
				r := []byte{1, 2, 3}
				a, err := New(&enginev1.PayloadAttributes{PrevRandao: r})
				require.NoError(t, err)
				require.DeepEqual(t, r, a.PrevRandao())
			},
		},
		{
			name: "Get suggested fee recipient",
			tc: func(t *testing.T) {
				r := []byte{4, 5, 6}
				a, err := New(&enginev1.PayloadAttributes{SuggestedFeeRecipient: r})
				require.NoError(t, err)
				require.DeepEqual(t, r, a.SuggestedFeeRecipient())
			},
		},
		{
			name: "Get timestamp",
			tc: func(t *testing.T) {
				r := uint64(123)
				a, err := New(&enginev1.PayloadAttributes{Timestamp: r})
				require.NoError(t, err)
				require.Equal(t, r, a.Timestamps())
			},
		},
		{
			name: "Get withdrawals (bellatrix)",
			tc: func(t *testing.T) {
				a := EmptyWithVersion(version.Bellatrix)
				_, err := a.Withdrawals()
				require.ErrorContains(t, "Withdrawals is not supported for bellatrix: unsupported getter", err)
			},
		},
		{
			name: "Get withdrawals (capella)",
			tc: func(t *testing.T) {
				wd := []*enginev1.Withdrawal{{WithdrawalIndex: 1}, {WithdrawalIndex: 2}, {WithdrawalIndex: 3}}
				a, err := New(&enginev1.PayloadAttributesV2{Withdrawals: wd})
				require.NoError(t, err)
				got, err := a.Withdrawals()
				require.NoError(t, err)
				require.DeepEqual(t, wd, got)
			},
		},
		{
			name: "Get PbV1 (bad version)",
			tc: func(t *testing.T) {
				a, err := New(&enginev1.PayloadAttributes{})
				require.NoError(t, err)
				_, err = a.PbV2()
				require.ErrorContains(t, "PayloadAttributePbV2 is not supported for bellatrix: unsupported getter", err)
			},
		},
		{
			name: "Get PbV2 (bad version)",
			tc: func(t *testing.T) {
				a, err := New(&enginev1.PayloadAttributesV2{})
				require.NoError(t, err)
				_, err = a.PbV1()
				require.ErrorContains(t, "PayloadAttributePbV1 is not supported for capella: unsupported getter", err)
			},
		},
		{
			name: "Get PbV1 (nil)",
			tc: func(t *testing.T) {
				a, err := New(&enginev1.PayloadAttributes{})
				require.NoError(t, err)
				got, err := a.PbV1()
				require.NoError(t, err)
				require.Equal(t, (*enginev1.PayloadAttributes)(nil), got)
			},
		},
		{
			name: "Get PbV2 (nil)",
			tc: func(t *testing.T) {
				a, err := New(&enginev1.PayloadAttributesV2{})
				require.NoError(t, err)
				got, err := a.PbV2()
				require.NoError(t, err)
				require.Equal(t, (*enginev1.PayloadAttributesV2)(nil), got)
			},
		},
		{
			name: "Get PbV1",
			tc: func(t *testing.T) {
				p := &enginev1.PayloadAttributes{
					Timestamp:             1,
					PrevRandao:            []byte{1, 2, 3},
					SuggestedFeeRecipient: []byte{4, 5, 6},
				}
				a, err := New(p)
				require.NoError(t, err)
				got, err := a.PbV1()
				require.NoError(t, err)
				require.DeepEqual(t, p, got)
			},
		},
		{
			name: "Get PbV2",
			tc: func(t *testing.T) {
				p := &enginev1.PayloadAttributesV2{
					Timestamp:             1,
					PrevRandao:            []byte{1, 2, 3},
					SuggestedFeeRecipient: []byte{4, 5, 6},
					Withdrawals:           []*enginev1.Withdrawal{{WithdrawalIndex: 1}, {WithdrawalIndex: 2}, {WithdrawalIndex: 3}},
				}
				a, err := New(p)
				require.NoError(t, err)
				got, err := a.PbV2()
				require.NoError(t, err)
				require.DeepEqual(t, p, got)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, test.tc)
	}
}
