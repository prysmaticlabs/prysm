package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	slashing "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/standard-protection-format"
)

// Export func to handle the rpc call returning the json slashing history.
// The format of the export follows an EIP-3076 standard JSON making it
// easy to migrate machines or eth2 clients.
//
// Steps:
// 1. Open the validator database at the default location.
// 2. Call the function which actually exports the data from
//  the validator's db into an EIP standard slashing protection format.
// 3. Format and save the JSON file to a user's specified output directory.
func (s *Server) ExportSlashingProtection(ctx context.Context, _ *empty.Empty) (*pb.ExportSlashingProtectionResponse, error) {
	var err error
	// Default location.
	dataDir := s.walletDir

	// Ensure that the validator.db is found under the specified dir or its subdirectories.
	found, _, err := fileutil.RecursiveFileFind(kv.ProtectionDbFileName, dataDir)
	if err != nil {
		return nil, errors.Wrapf(err, "error finding validator database at path %s", dataDir)
	}
	if !found {
		return nil, errors.New("err finding validator database at path " + dataDir)
	}

	validatorDB, err := kv.NewKVStore(ctx, dataDir, &kv.Config{})
	if err != nil {
		return nil, errors.Wrapf(err, "could not access validator database at path %s", dataDir)
	}
	defer func() {
		if err := validatorDB.Close(); err != nil {
			log.WithError(err).Errorf("Could not close validator DB")
		}
	}()
	eipJSON, err := slashing.ExportStandardProtectionJSON(ctx, validatorDB)
	if err != nil {
		return nil, errors.Wrap(err, "could not export slashing protection history")
	}
	encoded, err := json.MarshalIndent(eipJSON, "", "\t")
	if err != nil {
		return nil, errors.Wrap(err, "could not JSON marshal slashing protection history")
	}

	return &pb.ExportSlashingProtectionResponse{
		File: string(encoded),
	}, nil

}

// Import Slashing reads an input slashing protection EIP-3076
// standard JSON string and attempts to insert its data into our validator DB.
//
// Steps:
// 1. Open the validator database using the default datadir.
// 2. Read the JSON string passed through rpc.
// 3. Call the function which actually imports the data from
// from the standard slashing protection JSON file into our database.
func (s *Server) ImportSlashingProtection(ctx context.Context, req *pb.ImportSlashingProtectionRequest) (*emptypb.Empty, error) {
	var err error
	// Slashing Directory.
	dataDir := s.walletDir

	// Ensure that the validator.db is found under the specified dir or its subdirectories.
	found, _, err := fileutil.RecursiveFileFind(kv.ProtectionDbFileName, dataDir)
	if err != nil {
		return nil, errors.Wrapf(err, "err finding validator database at path %s", dataDir)
	}
	if !found {
		return nil, errors.New("err finding validator database at path %s" + dataDir)
	}

	valDB, err := kv.NewKVStore(ctx, dataDir, &kv.Config{})
	if err != nil {
		return nil, errors.Wrapf(err, "could not access validator database at path: %s", dataDir)
	}
	defer func() {
		if err := valDB.Close(); err != nil {
			log.WithError(err).Errorf("Could not close validator DB")
		}
	}()
	if req.SlashingProtectionJSON == "" {
		return nil, status.Errorf(codes.InvalidArgument, "empty slashing_protection json specified")
	}
	enc := []byte(req.SlashingProtectionJSON)

	buf := bytes.NewBuffer(enc)
	if err := slashing.ImportStandardProtectionJSON(ctx, valDB, buf); err != nil {
		return nil, err
	}
	log.Info("Slashing protection JSON successfully imported")
	return &empty.Empty{}, nil
}
