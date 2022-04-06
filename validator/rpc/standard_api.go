package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
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
		return nil, status.Error(codes.FailedPrecondition, "Prysm Wallet not initialized. Please create a new wallet.")
	}
	if s.validatorService == nil {
		return nil, status.Error(codes.FailedPrecondition, "Validator service not ready. Please try again once validator is ready.")
	}
	km, err := s.validatorService.Keymanager()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get Prysm keymanager: %v", err)
	}
	if s.wallet.KeymanagerKind() != keymanager.Derived && s.wallet.KeymanagerKind() != keymanager.Local {
		return nil, status.Errorf(codes.FailedPrecondition, "Prysm validator keys are not stored locally with this keymanager type.")
	}
	pubKeys, err := km.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve keystores: %v", err)
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
		Data: keystoreResponse,
	}, nil
}

// ImportKeystores allows for importing keystores into Prysm with their slashing protection history.
func (s *Server) ImportKeystores(
	ctx context.Context, req *ethpbservice.ImportKeystoresRequest,
) (*ethpbservice.ImportKeystoresResponse, error) {
	if !s.walletInitialized {
		statuses := groupImportErrors(req, "Prysm Wallet not initialized. Please create a new wallet.")
		return &ethpbservice.ImportKeystoresResponse{Data: statuses}, nil
	}
	if s.validatorService == nil {
		statuses := groupImportErrors(req, "Validator service not ready. Please try again once validator is ready.")
		return &ethpbservice.ImportKeystoresResponse{Data: statuses}, nil
	}
	km, err := s.validatorService.Keymanager()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get keymanager: %v", err)
	}
	importer, ok := km.(keymanager.Importer)
	if !ok {
		statuses := groupImportErrors(req, "Keymanager kind cannot import keys")
		return &ethpbservice.ImportKeystoresResponse{Data: statuses}, nil
	}
	if len(req.Keystores) == 0 {
		return &ethpbservice.ImportKeystoresResponse{}, nil
	}
	keystores := make([]*keymanager.Keystore, len(req.Keystores))
	for i := 0; i < len(req.Keystores); i++ {
		k := &keymanager.Keystore{}
		err = json.Unmarshal([]byte(req.Keystores[i]), k)
		if err != nil {
			// we want to ignore unmarshal errors for now, proper status in importKeystore
			k.Pubkey = "invalid format"
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
			return &ethpbservice.ImportKeystoresResponse{Data: statuses}, nil
		}
	}
	if len(req.Passwords) == 0 {
		req.Passwords = make([]string, len(req.Keystores))
	}

	// req.Passwords and req.Keystores are checked for 0 length in code above.
	if len(req.Passwords) > len(req.Keystores) {
		req.Passwords = req.Passwords[:len(req.Keystores)]
	}
	if len(req.Passwords) < len(req.Keystores) {
		passwordList := make([]string, len(req.Keystores))
		copy(passwordList, req.Passwords)
		req.Passwords = passwordList
	}

	statuses, err := importer.ImportKeystores(ctx, keystores, req.Passwords)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not import keystores: %v", err)
	}

	// If any of the keys imported had a slashing protection history before, we
	// stop marking them as deleted from our validator database.
	return &ethpbservice.ImportKeystoresResponse{Data: statuses}, nil
}

func groupImportErrors(req *ethpbservice.ImportKeystoresRequest, errorMessage string) []*ethpbservice.ImportedKeystoreStatus {
	statuses := make([]*ethpbservice.ImportedKeystoreStatus, len(req.Keystores))
	for i := 0; i < len(req.Keystores); i++ {
		statuses[i] = &ethpbservice.ImportedKeystoreStatus{
			Status:  ethpbservice.ImportedKeystoreStatus_ERROR,
			Message: errorMessage,
		}
	}
	return statuses
}

// DeleteKeystores allows for deleting specified public keys from Prysm.
func (s *Server) DeleteKeystores(
	ctx context.Context, req *ethpbservice.DeleteKeystoresRequest,
) (*ethpbservice.DeleteKeystoresResponse, error) {
	if !s.walletInitialized {
		statuses := groupExportErrors(req, "Prysm Wallet not initialized. Please create a new wallet.")
		return &ethpbservice.DeleteKeystoresResponse{Data: statuses}, nil
	}
	if s.validatorService == nil {
		statuses := groupExportErrors(req, "Validator service not ready")
		return &ethpbservice.DeleteKeystoresResponse{Data: statuses}, nil
	}
	km, err := s.validatorService.Keymanager()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get keymanager: %v", err)
	}
	if len(req.Pubkeys) == 0 {
		return &ethpbservice.DeleteKeystoresResponse{Data: make([]*ethpbservice.DeletedKeystoreStatus, 0)}, nil
	}
	statuses, err := km.DeleteKeystores(ctx, req.Pubkeys)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not delete keys: %v", err)
	}

	statuses, err = s.transformDeletedKeysStatuses(ctx, req.Pubkeys, statuses)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not transform deleted keys statuses: %v", err)
	}

	exportedHistory, err := s.slashingProtectionHistoryForDeletedKeys(ctx, req.Pubkeys, statuses)
	if err != nil {
		log.Warnf("Could not get slashing protection history for deleted keys: %v", err)
		statuses := groupExportErrors(req, "Non duplicate keys that were existing were deleted, but could not export slashing protection history.")
		return &ethpbservice.DeleteKeystoresResponse{Data: statuses}, nil
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
		Data:               statuses,
		SlashingProtection: string(jsonHist),
	}, nil
}

func groupExportErrors(req *ethpbservice.DeleteKeystoresRequest, errorMessage string) []*ethpbservice.DeletedKeystoreStatus {
	statuses := make([]*ethpbservice.DeletedKeystoreStatus, len(req.Pubkeys))
	for i := 0; i < len(req.Pubkeys); i++ {
		statuses[i] = &ethpbservice.DeletedKeystoreStatus{
			Status:  ethpbservice.DeletedKeystoreStatus_ERROR,
			Message: errorMessage,
		}
	}
	return statuses
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
func (s *Server) publicKeysInDB(ctx context.Context) (map[[fieldparams.BLSPubkeyLength]byte]bool, error) {
	pubKeysInDB := make(map[[fieldparams.BLSPubkeyLength]byte]bool)
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
