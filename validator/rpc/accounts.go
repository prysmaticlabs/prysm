package rpc

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/prysmaticlabs/prysm/crypto/bls"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/pagination"
	"github.com/prysmaticlabs/prysm/shared/petnames"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListAccounts allows retrieval of validating keys and their petnames
// for a user's wallet via RPC.
func (s *Server) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	if !s.walletInitialized {
		return nil, status.Error(codes.FailedPrecondition, "Wallet not yet initialized")
	}
	if int(req.PageSize) > cmd.Get().MaxRPCPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "Requested page size %d can not be greater than max size %d",
			req.PageSize, cmd.Get().MaxRPCPageSize)
	}
	keys, err := s.keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return nil, err
	}
	accs := make([]*pb.Account, len(keys))
	for i := 0; i < len(keys); i++ {
		accs[i] = &pb.Account{
			ValidatingPublicKey: keys[i][:],
			AccountName:         petnames.DeterministicName(keys[i][:], "-"),
		}
		if s.wallet.KeymanagerKind() == keymanager.Derived {
			accs[i].DerivationPath = fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, i)
		}
	}
	if req.All {
		return &pb.ListAccountsResponse{
			Accounts:      accs,
			TotalSize:     int32(len(keys)),
			NextPageToken: "",
		}, nil
	}
	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), len(keys))
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"Could not paginate results: %v",
			err,
		)
	}
	return &pb.ListAccountsResponse{
		Accounts:      accs[start:end],
		TotalSize:     int32(len(keys)),
		NextPageToken: nextPageToken,
	}, nil
}

// BackupAccounts creates a zip file containing EIP-2335 keystores for the user's
// specified public keys by encrypting them with the specified password.
func (s *Server) BackupAccounts(
	ctx context.Context, req *pb.BackupAccountsRequest,
) (*pb.BackupAccountsResponse, error) {
	if req.PublicKeys == nil || len(req.PublicKeys) < 1 {
		return nil, status.Error(codes.InvalidArgument, "No public keys specified to backup")
	}
	if req.BackupPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "Backup password cannot be empty")
	}
	if s.wallet == nil || s.keymanager == nil {
		return nil, status.Error(codes.FailedPrecondition, "No wallet nor keymanager found")
	}
	if s.wallet.KeymanagerKind() != keymanager.Imported && s.wallet.KeymanagerKind() != keymanager.Derived {
		return nil, status.Error(codes.FailedPrecondition, "Only HD or imported wallets can backup accounts")
	}
	pubKeys := make([]bls.PublicKey, len(req.PublicKeys))
	for i, key := range req.PublicKeys {
		pubKey, err := bls.PublicKeyFromBytes(key)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%#x Not a valid BLS public key: %v", key, err)
		}
		pubKeys[i] = pubKey
	}

	var err error
	var keystoresToBackup []*keymanager.Keystore
	switch s.wallet.KeymanagerKind() {
	case keymanager.Imported:
		km, ok := s.keymanager.(*imported.Keymanager)
		if !ok {
			return nil, status.Error(codes.FailedPrecondition, "Could not assert keymanager interface to concrete type")
		}
		keystoresToBackup, err = km.ExtractKeystores(ctx, pubKeys, req.BackupPassword)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not backup accounts for imported keymanager: %v", err)
		}
	case keymanager.Derived:
		km, ok := s.keymanager.(*derived.Keymanager)
		if !ok {
			return nil, status.Error(codes.FailedPrecondition, "Could not assert keymanager interface to concrete type")
		}
		keystoresToBackup, err = km.ExtractKeystores(ctx, pubKeys, req.BackupPassword)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not backup accounts for derived keymanager: %v", err)
		}
	}
	if len(keystoresToBackup) == 0 {
		return nil, status.Error(codes.InvalidArgument, "No keystores to backup")
	}

	buf := new(bytes.Buffer)
	writer := zip.NewWriter(buf)
	for i, k := range keystoresToBackup {
		encodedFile, err := json.MarshalIndent(k, "", "\t")
		if err != nil {
			if err := writer.Close(); err != nil {
				log.WithError(err).Error("Could not close zip file after writing")
			}
			return nil, status.Errorf(codes.Internal, "could not marshal keystore to JSON file: %v", err)
		}
		f, err := writer.Create(fmt.Sprintf("keystore-%d.json", i))
		if err != nil {
			if err := writer.Close(); err != nil {
				log.WithError(err).Error("Could not close zip file after writing")
			}
			return nil, status.Errorf(codes.Internal, "Could not write keystore file to zip: %v", err)
		}
		if _, err = f.Write(encodedFile); err != nil {
			if err := writer.Close(); err != nil {
				log.WithError(err).Error("Could not close zip file after writing")
			}
			return nil, status.Errorf(codes.Internal, "Could not write keystore file contents")
		}
	}
	if err := writer.Close(); err != nil {
		log.WithError(err).Error("Could not close zip file after writing")
	}
	return &pb.BackupAccountsResponse{
		ZipFile: buf.Bytes(),
	}, nil
}

// DeleteAccounts deletes accounts from a user's wallet is an imported or derived wallet.
func (s *Server) DeleteAccounts(
	ctx context.Context, req *pb.DeleteAccountsRequest,
) (*pb.DeleteAccountsResponse, error) {
	if req.PublicKeysToDelete == nil || len(req.PublicKeysToDelete) < 1 {
		return nil, status.Error(codes.InvalidArgument, "No public keys specified to delete")
	}
	if s.wallet == nil || s.keymanager == nil {
		return nil, status.Error(codes.FailedPrecondition, "No wallet found")
	}
	if s.wallet.KeymanagerKind() != keymanager.Imported && s.wallet.KeymanagerKind() != keymanager.Derived {
		return nil, status.Error(codes.FailedPrecondition, "Only Imported or Derived wallets can delete accounts")
	}
	if err := accounts.DeleteAccount(ctx, &accounts.Config{
		Wallet:           s.wallet,
		Keymanager:       s.keymanager,
		DeletePublicKeys: req.PublicKeysToDelete,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not delete public keys: %v", err)
	}
	return &pb.DeleteAccountsResponse{
		DeletedKeys: req.PublicKeysToDelete,
	}, nil
}

// VoluntaryExit performs a voluntary exit for the validator keys specified in a request.
func (s *Server) VoluntaryExit(
	ctx context.Context, req *pb.VoluntaryExitRequest,
) (*pb.VoluntaryExitResponse, error) {
	if len(req.PublicKeys) == 0 {
		return nil, status.Error(codes.InvalidArgument, "No public keys specified to delete")
	}
	if s.wallet == nil || s.keymanager == nil {
		return nil, status.Error(codes.FailedPrecondition, "No wallet found")
	}
	if s.wallet.KeymanagerKind() != keymanager.Imported && s.wallet.KeymanagerKind() != keymanager.Derived {
		return nil, status.Error(
			codes.FailedPrecondition, "Only Imported or Derived wallets can submit voluntary exits",
		)
	}
	formattedKeys := make([]string, len(req.PublicKeys))
	for i, key := range req.PublicKeys {
		formattedKeys[i] = fmt.Sprintf("%#x", key)
	}
	cfg := accounts.PerformExitCfg{
		ValidatorClient:  s.beaconNodeValidatorClient,
		NodeClient:       s.beaconNodeClient,
		Keymanager:       s.keymanager,
		RawPubKeys:       req.PublicKeys,
		FormattedPubKeys: formattedKeys,
	}
	rawExitedKeys, _, err := accounts.PerformVoluntaryExit(ctx, cfg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not perform voluntary exit: %v", err)
	}
	return &pb.VoluntaryExitResponse{
		ExitedKeys: rawExitedKeys,
	}, nil
}
