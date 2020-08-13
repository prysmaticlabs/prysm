package rpc

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
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

func TestServer_ListStatusesHappy(t *testing.T) {
	ctx := context.Background()
	fv := setupFakeClient()
	vs, err := client.NewValidatorService(ctx, &client.Config{Validator: fv})
	require.NoError(t, err)
	s := &Server{validatorService: vs}
	req := &pb.AccountRequest{
		PublicKeys: [][]byte{{'a'}, {'b'}, {'c'}},
		Indices:    []uint64{4, 5, 6},
	}
	got, err := s.ListStatuses(ctx, req)
	require.NoError(t, err)

	want := &pb.ListStatusesResponse{
		PublicKeys: [][]byte{
			bytesutil.PadTo([]byte{'a'}, 48),
			bytesutil.PadTo([]byte{'b'}, 48),
			bytesutil.PadTo([]byte{'c'}, 48),
			bytesutil.PadTo([]byte{'d'}, 48),
			bytesutil.PadTo([]byte{'e'}, 48),
			bytesutil.PadTo([]byte{'f'}, 48)},
		Indices:  []uint64{1, 2, 3, 4, 5, 6},
		Statuses: []pb.ListStatusesResponse_ValidatorStatus{0, 1, 2, 3, 4, 5},
	}
	require.DeepEqual(t, want, got)
}

func TestServer_ListStatusesOverlaps(t *testing.T) {
	ctx := context.Background()
	fv := setupFakeClient()
	vs, err := client.NewValidatorService(ctx, &client.Config{Validator: fv})
	require.NoError(t, err)
	s := &Server{validatorService: vs}
	req := &pb.AccountRequest{
		PublicKeys: [][]byte{{'a'}, {'b'}, {'c'}},
		Indices:    []uint64{1, 2, 4},
	}
	got, err := s.ListStatuses(ctx, req)
	require.NoError(t, err)

	want := &pb.ListStatusesResponse{
		PublicKeys: [][]byte{
			bytesutil.PadTo([]byte{'a'}, 48),
			bytesutil.PadTo([]byte{'b'}, 48),
			bytesutil.PadTo([]byte{'c'}, 48),
			bytesutil.PadTo([]byte{'d'}, 48)},
		Indices:  []uint64{1, 2, 3, 4},
		Statuses: []pb.ListStatusesResponse_ValidatorStatus{0, 1, 2, 3},
	}
	require.DeepEqual(t, want, got)
}

func TestServer_ListStatusesMissing(t *testing.T) {
	ctx := context.Background()
	fv := setupFakeClient()
	vs, err := client.NewValidatorService(ctx, &client.Config{Validator: fv})
	require.NoError(t, err)
	s := &Server{validatorService: vs}
	req := &pb.AccountRequest{
		PublicKeys: [][]byte{{'a'}, {'x'}, {'y'}},
		Indices:    []uint64{1, 200, 400},
	}
	got, err := s.ListStatuses(ctx, req)
	require.NoError(t, err)

	want := &pb.ListStatusesResponse{
		PublicKeys: [][]byte{
			bytesutil.PadTo([]byte{'a'}, 48)},
		Indices:  []uint64{1},
		Statuses: []pb.ListStatusesResponse_ValidatorStatus{0},
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
			{'a'}: 1,
			{'b'}: 2,
			{'c'}: 3,
			{'d'}: 4,
			{'e'}: 5,
			{'f'}: 6,
		},
		Balances: map[[48]byte]uint64{
			{'a'}: 11,
			{'b'}: 12,
			{'c'}: 13,
			{'d'}: 14,
			{'e'}: 15,
			{'f'}: 16,
		},
		PubkeysToStatusesMap: map[[48]byte]ethpb.ValidatorStatus{
			[48]byte{'a'}: 0,
			[48]byte{'b'}: 1,
			[48]byte{'c'}: 2,
			[48]byte{'d'}: 3,
			[48]byte{'e'}: 4,
			[48]byte{'f'}: 5,
		},
	}
}
