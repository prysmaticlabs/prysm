package enginev1_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestGetDecodedExecutionRequests(t *testing.T) {
	// Positive case
	t.Run("Positive case: valid execution requests", func(t *testing.T) {
		dr := &enginev1.DepositRequest{
			Pubkey:                make([]byte, 48),
			WithdrawalCredentials: make([]byte, 32),
			Amount:                32000000000,
			Signature:             make([]byte, 96),
			Index:                 1,
		}
		drBytes, _ := dr.MarshalSSZ()
		executionRequestBytes := append([]byte{0}, drBytes...)
		eb := &enginev1.ExecutionBundleElectra{
			ExecutionRequests: executionRequestBytes,
		}
		er, err := eb.GetDecodedExecutionRequests()
		require.NoError(t, err)
		require.Equal(t, len(er.Deposits), 1)
	})

	// Negative case: empty ExecutionRequests
	t.Run("Negative case: empty ExecutionRequests", func(t *testing.T) {
		eb := &enginev1.ExecutionBundleElectra{
			ExecutionRequests: []byte{},
		}
		_, err := eb.GetDecodedExecutionRequests()
		require.ErrorContains(t, "no execution requests found", err)
	})

	// Negative case: invalid execution request type
	t.Run("Negative case: invalid execution request type", func(t *testing.T) {
		eb := &enginev1.ExecutionBundleElectra{
			ExecutionRequests: []byte{99},
		}
		_, err := eb.GetDecodedExecutionRequests()
		require.ErrorContains(t, "invalid execution request type", err)
	})

	// Negative case: verifyExecutionRequests returns error
	t.Run("Negative case: verifyExecutionRequests error", func(t *testing.T) {
		var executionRequestBytes []byte
		for i := 0; i < int(params.BeaconConfig().MaxDepositRequestsPerPayload)+1; i++ {
			dr := &enginev1.DepositRequest{
				Pubkey:                make([]byte, 48),
				WithdrawalCredentials: make([]byte, 32),
				Amount:                32000000000,
				Signature:             make([]byte, 96),
				Index:                 uint64(i),
			}
			drBytes, _ := dr.MarshalSSZ()
			executionRequestBytes = append(executionRequestBytes, byte(0))
			executionRequestBytes = append(executionRequestBytes, drBytes...)
		}
		eb := &enginev1.ExecutionBundleElectra{
			ExecutionRequests: executionRequestBytes,
		}
		_, err := eb.GetDecodedExecutionRequests()
		require.ErrorContains(t, "too many deposits requested", err)
	})
}

func TestVerifyExecutionRequests(t *testing.T) {
	// Positive case: counts within limits
	t.Run("Positive case: counts within limits", func(t *testing.T) {
		er := &enginev1.ExecutionRequests{
			Deposits:       make([]*enginev1.DepositRequest, params.BeaconConfig().MaxDepositRequestsPerPayload),
			Withdrawals:    make([]*enginev1.WithdrawalRequest, params.BeaconConfig().MaxWithdrawalRequestsPerPayload),
			Consolidations: make([]*enginev1.ConsolidationRequest, params.BeaconConfig().MaxConsolidationsRequestsPerPayload),
		}
		require.NoError(t, enginev1.VerifyExecutionRequests(er))
	})

	// Negative case: too many deposits
	t.Run("Negative case: too many deposits", func(t *testing.T) {
		er := &enginev1.ExecutionRequests{
			Deposits: make([]*enginev1.DepositRequest, params.BeaconConfig().MaxDepositRequestsPerPayload+1),
		}
		err := enginev1.VerifyExecutionRequests(er)
		require.ErrorContains(t, "too many deposits requested", err)
	})

	// Negative case: too many withdrawals
	t.Run("Negative case: too many withdrawals", func(t *testing.T) {
		er := &enginev1.ExecutionRequests{
			Withdrawals: make([]*enginev1.WithdrawalRequest, params.BeaconConfig().MaxWithdrawalRequestsPerPayload+1),
		}
		err := enginev1.VerifyExecutionRequests(er)
		require.ErrorContains(t, "too many withdrawals requested", err)
	})

	// Negative case: too many consolidations
	t.Run("Negative case: too many consolidations", func(t *testing.T) {
		er := &enginev1.ExecutionRequests{
			Consolidations: make([]*enginev1.ConsolidationRequest, params.BeaconConfig().MaxConsolidationsRequestsPerPayload+1),
		}
		err := enginev1.VerifyExecutionRequests(er)
		require.ErrorContains(t, "too many consolidations requested", err)
	})
}

func TestProcessRequestBytes(t *testing.T) {
	// Positive case: valid deposit request
	t.Run("Positive case: valid mix of requests", func(t *testing.T) {
		dr := &enginev1.DepositRequest{
			Pubkey:                make([]byte, 48),
			WithdrawalCredentials: make([]byte, 32),
			Amount:                32000000000,
			Signature:             make([]byte, 96),
			Index:                 1,
		}
		drBytes, err := dr.MarshalSSZ()
		require.NoError(t, err)
		requestBytes := append([]byte{0}, drBytes...)

		wr := &enginev1.WithdrawalRequest{
			SourceAddress:   make([]byte, 20),
			ValidatorPubkey: make([]byte, 48),
			Amount:          16000000000,
		}
		wrBytes, err := wr.MarshalSSZ()
		require.NoError(t, err)
		requestBytes = append(requestBytes, byte(1))
		requestBytes = append(requestBytes, wrBytes...)

		cr := enginev1.ConsolidationRequest{
			SourceAddress: make([]byte, 20),
			SourcePubkey:  make([]byte, 48),
			TargetPubkey:  make([]byte, 48),
		}
		crBytes, err := cr.MarshalSSZ()
		require.NoError(t, err)
		requestBytes = append(requestBytes, byte(2))
		requestBytes = append(requestBytes, crBytes...)

		requests := &enginev1.ExecutionRequests{}
		err = enginev1.ProcessRequestBytes(requestBytes, requests)
		require.NoError(t, err)
		require.Equal(t, len(requests.Deposits), 1)
		require.Equal(t, len(requests.Withdrawals), 1)
		require.Equal(t, len(requests.Consolidations), 1)
	})

	// Negative case: invalid request type
	t.Run("Negative case: invalid request type", func(t *testing.T) {
		requestBytes := []byte{99}
		requests := &enginev1.ExecutionRequests{}
		err := enginev1.ProcessRequestBytes(requestBytes, requests)
		require.ErrorContains(t, "invalid execution request type", err)
	})

	// Negative case: insufficient bytes for request
	t.Run("Negative case: insufficient bytes for deposit request", func(t *testing.T) {
		dr := &enginev1.DepositRequest{}
		requestBytes := append([]byte{0}, make([]byte, dr.SizeSSZ()-1)...)
		requests := &enginev1.ExecutionRequests{}
		err := enginev1.ProcessRequestBytes(requestBytes, requests)
		require.ErrorContains(t, "invalid deposit request size", err)
	})
}

func TestEncodeExecutionRequests(t *testing.T) {
	// Positive case: valid requests
	t.Run("Positive case: valid requests", func(t *testing.T) {
		eb := &enginev1.ExecutionRequests{
			Deposits: []*enginev1.DepositRequest{
				{
					Pubkey:                make([]byte, 48),
					WithdrawalCredentials: make([]byte, 32),
					Amount:                32000000000,
					Signature:             make([]byte, 96),
					Index:                 1,
				},
			},
			Withdrawals: []*enginev1.WithdrawalRequest{
				{
					SourceAddress:   make([]byte, 20),
					ValidatorPubkey: make([]byte, 48),
					Amount:          16000000000,
				},
			},
			Consolidations: []*enginev1.ConsolidationRequest{
				{
					SourceAddress: make([]byte, 20),
					SourcePubkey:  make([]byte, 48),
					TargetPubkey:  make([]byte, 48),
				},
			},
		}
		hash, err := enginev1.EncodeExecutionRequests(eb)
		require.NoError(t, err)

		require.NotEqual(t, hash.Cmp(common.Hash{}), 0)
	})

	// Negative case: MarshalSSZ returns error
	t.Run("Negative case: deposit request MarshalSSZ error", func(t *testing.T) {
		dr := &enginev1.DepositRequest{
			Pubkey: []byte{}, // Invalid length
		}
		eb := &enginev1.ExecutionRequests{
			Deposits: []*enginev1.DepositRequest{dr},
		}
		_, err := enginev1.EncodeExecutionRequests(eb)
		require.ErrorContains(t, "failed to encode deposit requests", err)
	})

	t.Run("Negative case: withdrawal request MarshalSSZ error", func(t *testing.T) {
		wr := &enginev1.WithdrawalRequest{
			ValidatorPubkey: []byte{}, // Invalid length
		}
		eb := &enginev1.ExecutionRequests{
			Withdrawals: []*enginev1.WithdrawalRequest{wr},
		}
		_, err := enginev1.EncodeExecutionRequests(eb)
		require.ErrorContains(t, "failed to encode withdrawal requests", err)
	})

	t.Run("Negative case: consolidation request MarshalSSZ error", func(t *testing.T) {
		cr := &enginev1.ConsolidationRequest{
			TargetPubkey: []byte{}, // Invalid length
		}
		eb := &enginev1.ExecutionRequests{
			Consolidations: []*enginev1.ConsolidationRequest{cr},
		}
		_, err := enginev1.EncodeExecutionRequests(eb)
		require.ErrorContains(t, "failed to encode consolidation requests", err)
	})
}
