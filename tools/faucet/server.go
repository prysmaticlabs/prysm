package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	recaptcha "github.com/prestonvanloon/go-recaptcha"
	faucetpb "github.com/prysmaticlabs/prysm/proto/faucet"
	"google.golang.org/grpc/peer"
)

const minScore = 0.8

var fundingAmount = big.NewInt(3.5 * params.Ether)
var funded = make(map[string]bool)
var fundingLock sync.Mutex

type faucetServer struct {
	r      recaptcha.Recaptcha
	client *ethclient.Client
	funder common.Address
	pk     *ecdsa.PrivateKey
}

func newFaucetServer(
	r recaptcha.Recaptcha,
	rpcPath string,
	funderPrivateKey string,
) *faucetServer {
	client, err := ethclient.DialContext(context.Background(), rpcPath)
	if err != nil {
		panic(err)
	}

	pk, err := crypto.HexToECDSA(funderPrivateKey)
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
		r:      r,
		client: client,
		funder: funder,
		pk:     pk,
	}
}

func (s *faucetServer) verifyRecaptcha(ctx context.Context, req *faucetpb.FundingRequest) error {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return errors.New("peer from ctx not ok")
	}
	fmt.Printf("Sending captcha request for peer %s\n", p.Addr.String())

	rr, err := s.r.Check(p.Addr.String(), req.RecaptchaResponse)
	if err != nil {
		return err
	}
	if !rr.Success {
		fmt.Printf("Unsuccessful recaptcha request. Error codes: %+v\n", rr.ErrorCodes)
		return errors.New("failed")
	}
	if rr.Score < minScore {
		return errors.New("recaptcha score too low")
	}

	return nil
}

// RequestFunds from the ethereum 1.x faucet. Requires a valid captcha
// response.
func (s *faucetServer) RequestFunds(ctx context.Context, req *faucetpb.FundingRequest) (*faucetpb.FundingResponse, error) {

	if err := s.verifyRecaptcha(ctx, req); err != nil {
		return &faucetpb.FundingResponse{Error: fmt.Sprintf("Recaptcha failure: %v", err)}, nil
	}

	fundingLock.Lock()
	if funded[req.WalletAddress] {
		fundingLock.Unlock()
		return &faucetpb.FundingResponse{Error: "funded too recently"}, nil
	}
	funded[req.WalletAddress] = true
	fundingLock.Unlock()

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

	// Wait for contract to mine
	for pending := true; pending; _, pending, err = s.client.TransactionByHash(context.Background(), tx.Hash()) {
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(1 * time.Second)
	}

	return tx.Hash().Hex(), nil
}
