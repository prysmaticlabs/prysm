package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	recaptcha "github.com/prestonvanloon/go-recaptcha"
	faucetpb "github.com/prysmaticlabs/prysm/proto/faucet"
	"google.golang.org/grpc/peer"
)

var minScore = 0.5
var fundingAmount = big.NewInt(0.5 * params.Ether)

type faucetServer struct {
	r      recaptcha.Recaptcha
	client *ethclient.Client
	funder common.Address
	pk     *ecdsa.PrivateKey
}

func NewFaucetServer(r recaptcha.Recaptcha, rpcPath, privateKey string) *faucetServer {
	client, err := ethclient.DialContext(context.Background(), rpcPath)
	if err != nil {
		panic(err)
	}

	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		panic(err)
	}

	funder := crypto.PubkeyToAddress(pk.PublicKey)

	bal, err := client.BalanceAt(context.Background(), funder, nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Funder is %s\n", funder.Hex())
	fmt.Printf("Funder has %d\n", bal)

	return &faucetServer{
		r,
		client,
		funder,
		pk,
	}
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

	txHash, err := s.fundAndWait(common.HexToAddress(req.WalletAddress))
	if err != nil {
		return &faucetpb.FundingResponse{Error: fmt.Sprintf("Failed to send transaction %v", err)}, nil
	}

	return &faucetpb.FundingResponse{
		Amount:          fundingAmount.String(),
		TransactionHash: txHash,
	}, nil
}

func (s *faucetServer) fundAndWait(to common.Address) (string, error) {
	nonce := uint64(0)
	nonce, err := s.client.PendingNonceAt(context.Background(), s.funder)
	if err != nil {
		return "", err
	}

	tx := types.NewTransaction(nonce, to, fundingAmount, 40000, big.NewInt(1*params.GWei), nil /*data*/)

	tx, err = types.SignTx(tx, types.NewEIP155Signer(big.NewInt(5)), s.pk)
	if err != nil {
		return "", err
	}

	if err := s.client.SendTransaction(context.Background(), tx); err != nil {
		return "", err
	}

	return tx.Hash().Hex(), nil
}
