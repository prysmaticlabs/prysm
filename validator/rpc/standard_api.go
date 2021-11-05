package rpc

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"
)

func (s Server) ListKeystores(
	ctx context.Context, _ *empty.Empty,
) (*ethpbservice.ListKeystoresResponse, error) {
	return nil, nil
}

//func (s Server) ImportKeystores(
//	ctx context.Context, req *ethpbservice.ImportKeystoresRequest,
//) (*ethpbservice.ImportKeystoresResponse, error) {
//	return nil, nil
//}

func (s Server) DeleteKeystores(
	ctx context.Context, req *ethpbservice.DeleteKeystoresRequest,
) (*ethpbservice.DeleteKeystoresResponse, error) {
	return nil, nil
}
