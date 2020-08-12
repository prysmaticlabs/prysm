package rpc

import (
	"context"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/client"
)

func TestServer_ListBalancesHappy(t *testing.T) {
	ctx := context.Background()
	fv := setupFakeClient()
	vs, err := client.NewValidatorService(ctx, &client.Config{Validator: fv})
	require.NoError(t, err)
	s := &Server{validatorService: vs}
	req := &pb.AccountRequest{
		PublicKeys: [][]byte{{'a'}, {'b'}, {'c'}},
		Indices:    []uint64{4, 5, 6},
	}
	got, err := s.ListBalances(ctx, req)
	require.NoError(t, err)

	want := &pb.ListBalancesResponse{
		PublicKeys: [][]byte{
			bytesutil.PadTo([]byte{'a'}, 48),
			bytesutil.PadTo([]byte{'b'}, 48),
			bytesutil.PadTo([]byte{'c'}, 48),
			bytesutil.PadTo([]byte{'d'}, 48),
			bytesutil.PadTo([]byte{'e'}, 48),
			bytesutil.PadTo([]byte{'f'}, 48)},
		Indices:  []uint64{1, 2, 3, 4, 5, 6},
		Balances: []uint64{11, 12, 13, 14, 15, 16},
	}
	require.DeepEqual(t, want, got)
}

func TestServer_ListBalancesOverlaps(t *testing.T) {
	ctx := context.Background()
	fv := setupFakeClient()
	vs, err := client.NewValidatorService(ctx, &client.Config{Validator: fv})
	require.NoError(t, err)
	s := &Server{validatorService: vs}
	req := &pb.AccountRequest{
		PublicKeys: [][]byte{{'a'}, {'b'}, {'c'}},
		Indices:    []uint64{1, 2, 4},
	}
	got, err := s.ListBalances(ctx, req)
	require.NoError(t, err)

	want := &pb.ListBalancesResponse{
		PublicKeys: [][]byte{
			bytesutil.PadTo([]byte{'a'}, 48),
			bytesutil.PadTo([]byte{'b'}, 48),
			bytesutil.PadTo([]byte{'c'}, 48),
			bytesutil.PadTo([]byte{'d'}, 48)},
		Indices:  []uint64{1, 2, 3, 4},
		Balances: []uint64{11, 12, 13, 14},
	}
	require.DeepEqual(t, want, got)
}

func TestServer_ListBalancesMissing(t *testing.T) {
	ctx := context.Background()
	fv := setupFakeClient()
	vs, err := client.NewValidatorService(ctx, &client.Config{Validator: fv})
	require.NoError(t, err)
	s := &Server{validatorService: vs}
	req := &pb.AccountRequest{
		PublicKeys: [][]byte{{'a'}, {'x'}, {'y'}},
		Indices:    []uint64{1, 200, 400},
	}
	got, err := s.ListBalances(ctx, req)
	require.NoError(t, err)

	want := &pb.ListBalancesResponse{
		PublicKeys: [][]byte{
			bytesutil.PadTo([]byte{'a'}, 48)},
		Indices:  []uint64{1},
		Balances: []uint64{11},
	}
	require.DeepEqual(t, want, got)
}

func setupFakeClient() *client.FakeValidator {
	return &client.FakeValidator{
		IndexToPubkeyMap: map[uint64][48]byte{
			1: {'a'},
			2: {'b'},
			3: {'c'},
			4: {'d'},
			5: {'e'},
			6: {'f'},
		},
		PubkeyToIndexMap: map[[48]byte]uint64{
			[48]byte{'a'}: 1,
			[48]byte{'b'}: 2,
			[48]byte{'c'}: 3,
			[48]byte{'d'}: 4,
			[48]byte{'e'}: 5,
			[48]byte{'f'}: 6,
		},
		Balances: map[[48]byte]uint64{
			[48]byte{'a'}: 11,
			[48]byte{'b'}: 12,
			[48]byte{'c'}: 13,
			[48]byte{'d'}: 14,
			[48]byte{'e'}: 15,
			[48]byte{'f'}: 16,
		},
	}
}
