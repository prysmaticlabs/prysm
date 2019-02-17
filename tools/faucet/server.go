package main

import (
	"context"
	"errors"
	"fmt"

	recaptcha "github.com/prestonvanloon/go-recaptcha"
	faucetpb "github.com/prysmaticlabs/prysm/tools/faucet/proto"
	"google.golang.org/grpc/peer"
)

var minScore float64 = 0.5

type faucetServer struct {
	r recaptcha.Recaptcha
}

// RequestFunds from the ethereum 1.x faucet. Requires a valid captcha
// response.
func (s *faucetServer) RequestFunds(ctx context.Context, req *faucetpb.FundingRequest) (*faucetpb.FundingResponse, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, errors.New("peer from ctx not ok")
	}
	fmt.Printf("Sending captcha request for peer %s\n", p.Addr.String())

	rr, err := s.r.Check(p.Addr.String(), req.RecaptchaResponse)
	if err != nil {
		return nil, err
	}
	if !rr.Success {
		fmt.Printf("Unsuccessful recaptcha request. Error codes: %+v\n", rr.ErrorCodes)
		return &faucetpb.FundingResponse{Error: "Recaptcha failed"}, nil
	}
	if rr.Score < minScore {
		return &faucetpb.FundingResponse{Error: "Recaptcha score too low"}, nil
	}

	return &faucetpb.FundingResponse{
		Amount:          "500000000000000000",
		TransactionHash: "0xfake",
	}, nil
}
