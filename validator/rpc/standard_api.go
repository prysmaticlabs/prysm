package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	slashingprotection "github.com/prysmaticlabs/prysm/validator/slashing-protection-history"
	"github.com/prysmaticlabs/prysm/validator/slashing-protection-history/format"
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

// ImportKeystores allows for importing keystores into Prysm with their slashing protection history.
func (s *Server) ImportKeystores(
	ctx context.Context, req *ethpbservice.ImportKeystoresRequest,
) (*ethpbservice.ImportKeystoresResponse, error) {
	if !s.walletInitialized {
		return nil, status.Error(codes.Internal, "Wallet not ready")
	}
	importer, ok := s.keymanager.(keymanager.Importer)
	if !ok {
		return nil, status.Error(codes.Internal, "Keymanager kind cannot import keys")
	}
	if len(req.Passwords) == 0 {
		return nil, status.Error(codes.Internal, "No passwords provided for keystores")
	}
	if len(req.Passwords) != len(req.Keystores) {
		return nil, status.Error(codes.Internal, "Number of passwords does not match number of keystores")
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
			statuses := make([]*ethpbservice.ImportedKeystoreStatus, len(req.Keystores))
			for i := range statuses {
				statuses[i] = &ethpbservice.ImportedKeystoreStatus{
					Status:  ethpbservice.ImportedKeystoreStatus_ERROR,
					Message: fmt.Sprintf("could not import slashing protection: %v", err),
				}
			}
			return &ethpbservice.ImportKeystoresResponse{Statuses: statuses}, nil
		}
	}
	statuses, err := importer.ImportKeystores(ctx, keystores, req.Passwords)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not import keystores: %v", err)
	}

	// If any of the keys imported had a slashing protection history before, we
	// stop marking them as deleted from our validator database.
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
	if len(req.PublicKeys) == 0 {
		return &ethpbservice.DeleteKeystoresResponse{Statuses: make([]*ethpbservice.DeletedKeystoreStatus, 0)}, nil
	}
	statuses, err := deleter.DeleteKeystores(ctx, req.PublicKeys)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not delete keys: %v", err)
	}
	if len(statuses) != len(req.PublicKeys) {
		return nil, status.Errorf(
			codes.Internal,
			"Wanted same amount of statuses %d as public keys %d",
			len(statuses),
			len(req.PublicKeys),
		)
	}

	statuses, err = s.transformDeletedKeysStatuses(ctx, req.PublicKeys, statuses)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not transform deleted keys statuses: %v", err)
	}

	exportedHistory, err := s.slashingProtectionHistoryForDeletedKeys(ctx, req.PublicKeys, statuses)
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
			"Could not JSON marshal slashing protection history: %v",
			err,
		)
	}
	return &ethpbservice.DeleteKeystoresResponse{
		Statuses:           statuses,
		SlashingProtection: string(jsonHist),
	}, nil
}

// For a list of deleted keystore statuses, we check if any NOT_FOUND status actually
// has a corresponding public key in the database. In this case, we transform the status
// to NOT_ACTIVE, as we do have slashing protection history for it and should not mark it
// as NOT_FOUND when returning a response to the caller.
func (s *Server) transformDeletedKeysStatuses(
	ctx context.Context, pubKeys [][]byte, statuses []*ethpbservice.DeletedKeystoreStatus,
) ([]*ethpbservice.DeletedKeystoreStatus, error) {
	pubKeysInDB, err := s.publicKeysInDB(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get public keys from DB: %v", err)
	}
	if len(pubKeysInDB) > 0 {
		for i := 0; i < len(pubKeys); i++ {
			keyExistsInDB := pubKeysInDB[bytesutil.ToBytes48(pubKeys[i])]
			if keyExistsInDB && statuses[i].Status == ethpbservice.DeletedKeystoreStatus_NOT_FOUND {
				statuses[i].Status = ethpbservice.DeletedKeystoreStatus_NOT_ACTIVE
			}
		}
	}
	return statuses, nil
}

// Gets a map of all public keys in the database, useful for O(1) lookups.
func (s *Server) publicKeysInDB(ctx context.Context) (map[[48]byte]bool, error) {
	pubKeysInDB := make(map[[48]byte]bool)
	attestedPublicKeys, err := s.valDB.AttestedPublicKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get attested public keys from DB: %v", err)
	}
	proposedPublicKeys, err := s.valDB.ProposedPublicKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get proposed public keys from DB: %v", err)
	}
	for _, pk := range append(attestedPublicKeys, proposedPublicKeys...) {
		pubKeysInDB[pk] = true
	}
	return pubKeysInDB, nil
}

// Exports slashing protection data for a list of DELETED or NOT_ACTIVE keys only to be used
// as part of the DeleteKeystores endpoint.
func (s *Server) slashingProtectionHistoryForDeletedKeys(
	ctx context.Context, pubKeys [][]byte, statuses []*ethpbservice.DeletedKeystoreStatus,
) (*format.EIPSlashingProtectionFormat, error) {
	// We select the keys that were DELETED or NOT_ACTIVE from the previous action
	// and use that to filter our slashing protection export.
	filteredKeys := make([][]byte, 0, len(pubKeys))
	for i, pk := range pubKeys {
		if statuses[i].Status == ethpbservice.DeletedKeystoreStatus_DELETED ||
			statuses[i].Status == ethpbservice.DeletedKeystoreStatus_NOT_ACTIVE {
			filteredKeys = append(filteredKeys, pk)
		}
	}
	return slashingprotection.ExportStandardProtectionJSON(ctx, s.valDB, filteredKeys...)
}
