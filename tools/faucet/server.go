package main

import (
	"context"
	"errors"
	"fmt"

	recaptcha "github.com/prestonvanloon/go-recaptcha"
	faucetpb "github.com/prysmaticlabs/prysm/tools/faucet/proto"
	"google.golang.org/grpc/peer"
)

var MIN_SCORE float64 = 0.5

type faucetServer struct {
	r recaptcha.Recaptcha
}

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
	fmt.Printf("Response: %+v\n", rr)
	if !rr.Success {
		fmt.Printf("Unsuccessful recaptcha request. Error codes: %+v\n", rr.ErrorCodes)
		return &faucetpb.FundingResponse{Error: "Recaptcha failed"}, nil
	}
	if rr.Score < MIN_SCORE {
		return &faucetpb.FundingResponse{Error: "Recaptcha score too low"}, nil
	}

	return &faucetpb.FundingResponse{
		Amount:          "500000000000000000",
		TransactionHash: "0xfake",
	}, nil
}
