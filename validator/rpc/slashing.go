package rpc

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client"
	slashing "github.com/prysmaticlabs/prysm/v3/validator/slashing-protection-history"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ExportSlashingProtection handles the rpc call returning the json slashing history.
// The format of the export follows the EIP-3076 standard which makes it
// easy to migrate machines or Ethereum consensus clients.
//
// Steps:
// 1. Call the function which exports the data from
//  the validator's db into an EIP standard slashing protection format.
// 2. Format and send JSON in the response.
func (s *Server) ExportSlashingProtection(ctx context.Context, _ *empty.Empty) (*pb.ExportSlashingProtectionResponse, error) {
	if s.valDB == nil {
		return nil, errors.New("err finding validator database at path")
	}

	eipJSON, err := slashing.ExportStandardProtectionJSON(ctx, s.valDB)
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

// ImportSlashingProtection reads an input slashing protection EIP-3076
// standard JSON string and inserts the data into validator DB.
//
// Read the JSON string passed through rpc, then call the func
// which actually imports the data from the JSON file into our database.
func (s *Server) ImportSlashingProtection(ctx context.Context, req *pb.ImportSlashingProtectionRequest) (*emptypb.Empty, error) {
	if s.valDB == nil {
		return nil, errors.New("err finding validator database at path")
	}

	if req.SlashingProtectionJson == "" {
		return nil, status.Errorf(codes.InvalidArgument, "empty slashing_protection json specified")
	}
	enc := []byte(req.SlashingProtectionJson)

	buf := bytes.NewBuffer(enc)
	if err := slashing.ImportStandardProtectionJSON(ctx, s.valDB, buf); err != nil {
		return nil, err
	}
	log.Info("Slashing protection JSON successfully imported")
	return &empty.Empty{}, nil
}
