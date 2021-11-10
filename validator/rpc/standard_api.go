package rpc

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//// DeleteKeystores implements the standard validator key management API.
//func (s Server) DeleteKeystores(
//	ctx context.Context, req *ethpbservice.DeleteKeystoresRequest,
//) (*ethpbservice.DeleteKeystoresResponse, error) {
//	if !s.walletInitialized {
//		return nil, status.Error(codes.Internal, "Wallet not ready")
//	}
//	im, ok := s.keymanager.(keymanager.Deleter)
//	if !ok {
//		return nil, status.Error(codes.Internal, "Keymanager kind cannot delete keys")
//	}
//	if err := im.DeleteAccounts(ctx, req.PublicKeys); err != nil {
//		return nil, status.Error(codes.Internal, "Could not delete keystores")
//	}
//	keysToFilter := req.PublicKeys
//
//	//pubKeys, err := s.keymanager.FetchValidatingPublicKeys(ctx)
//	//if err != nil {
//	//	return nil, status.Errorf(codes.Internal, "Could not list keystores: %v", err)
//	//}
//	//keystoreResponse := make([]*ethpbservice.ListKeystoresResponse_Keystore, len(pubKeys))
//	//for i := 0; i < len(pubKeys); i++ {
//	//	keystoreResponse[i] = &ethpbservice.ListKeystoresResponse_Keystore{
//	//		ValidatingPubkey: pubKeys[i][:],
//	//	}
//	//	if s.wallet.KeymanagerKind() == keymanager.Derived {
//	//		keystoreResponse[i].DerivationPath = fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, i)
//	//	}
//	//}
//	return &ethpbservice.DeleteKeystoresResponse{}, nil
//}
//
// ListKeystores implements the standard validator key management API.
func (s Server) ListKeystores(
	ctx context.Context, _ *empty.Empty,
) (*ethpbservice.ListKeystoresResponse, error) {
	if !s.walletInitialized {
		return nil, status.Error(codes.Internal, "Wallet not ready")
	}
	pubKeys, err := s.keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not list keystores: %v", err)
	}
	keystoreResponse := make([]*ethpbservice.ListKeystoresResponse_Keystore, len(pubKeys))
	for i := 0; i < len(pubKeys); i++ {
		keystoreResponse[i] = &ethpbservice.ListKeystoresResponse_Keystore{
			ValidatingPubkey: pubKeys[i][:],
		}
		if s.wallet.KeymanagerKind() == keymanager.Derived {
			keystoreResponse[i].DerivationPath = fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, i)
		}
	}
	return &ethpbservice.ListKeystoresResponse{
		Keystores: keystoreResponse,
	}, nil
}
