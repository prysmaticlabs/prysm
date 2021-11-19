package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	slashingprotection "github.com/prysmaticlabs/prysm/validator/slashing-protection-history"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListKeystores implements the standard validator key management API.
func (s *Server) ListKeystores(
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

// ImportKeystoresStandard allows for importing keystores into Prysm with their slashing protection history.
func (s *Server) ImportKeystoresStandard(
	ctx context.Context, req *ethpbservice.ImportKeystoresRequest,
) (*ethpbservice.ImportKeystoresResponse, error) {
	if !s.walletInitialized {
		return nil, status.Error(codes.Internal, "Wallet not ready")
	}
	importer, ok := s.keymanager.(keymanager.Importer)
	if !ok {
		return nil, status.Error(codes.Internal, "Keymanager kind cannot import keys")
	}
	keystores := make([]*keymanager.Keystore, len(req.Keystores))
	for i := 0; i < len(req.Keystores); i++ {
		k := &keymanager.Keystore{}
		if err := json.Unmarshal([]byte(req.Keystores[i]), k); err != nil {
			return nil, status.Errorf(
				codes.Internal, "Invalid keystore at index %d in request: %v", i, err,
			)
		}
		keystores[i] = k
	}
	if req.SlashingProtection != "" {
		if err := slashingprotection.ImportStandardProtectionJSON(
			ctx, s.valDB, bytes.NewBuffer([]byte(req.SlashingProtection)),
		); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not import slashing protection JSON: %v", err)
		}
	}
	statuses, err := importer.ImportKeystores(ctx, keystores, req.Passwords)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not import keystores: %v", err)
	}
	return &ethpbservice.ImportKeystoresResponse{Statuses: statuses}, nil
}

// DeleteKeystores allows for deleting specified public keys from Prysm.
func (s *Server) DeleteKeystores(
	ctx context.Context, req *ethpbservice.DeleteKeystoresRequest,
) (*ethpbservice.DeleteKeystoresResponse, error) {
	if !s.walletInitialized {
		return nil, status.Error(codes.Internal, "Wallet not ready")
	}
	deleter, ok := s.keymanager.(keymanager.Deleter)
	if !ok {
		return nil, status.Error(codes.Internal, "Keymanager kind cannot delete keys")
	}
	statuses, err := deleter.DeleteKeystores(ctx, req.PublicKeys)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not delete keys: %v", err)
	}
	keysToFilter := req.PublicKeys
	exportedHistory, err := slashingprotection.ExportStandardProtectionJSON(ctx, s.valDB, keysToFilter...)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"Could not export slashing protection history: %v",
			err,
		)
	}
	jsonHist, err := json.Marshal(exportedHistory)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"Could not export slashing protection history: %v",
			err,
		)
	}
	return &ethpbservice.DeleteKeystoresResponse{
		Statuses:           statuses,
		SlashingProtection: string(jsonHist),
	}, nil
}
