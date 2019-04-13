package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	pb "github.com/prysmaticlabs/prysm/proto/cluster"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

var gasLimit = uint64(4000000)

type server struct {
	contract      *contracts.DepositContract
	db            *db
	depositAmount *big.Int
	txPk          *ecdsa.PrivateKey
	client        *ethclient.Client

	clientLock sync.Mutex
}

func newServer(
	db *db,
	rpcAddr string,
	depositContractAddr string,
	funderPK string,
	validatorDepositAmount int64,
) *server {
	rpcClient, err := rpc.Dial(rpcAddr)
	if err != nil {
		panic(err)
	}
	client := ethclient.NewClient(rpcClient)

	contract, err := contracts.NewDepositContract(common.HexToAddress(depositContractAddr), client)
	if err != nil {
		panic(err)
	}

	txPk, err := crypto.HexToECDSA(funderPK)
	if err != nil {
		panic(err)
	}

	depositAmount := big.NewInt(validatorDepositAmount)

	return &server{
		contract:      contract,
		client:        client,
		db:            db,
		depositAmount: depositAmount,
		txPk:          txPk,
	}
}

func (s *server) makeDeposit(data []byte) error {
	txOps := bind.NewKeyedTransactor(s.txPk)
	txOps.Value = s.depositAmount
	txOps.GasLimit = gasLimit
	tx, err := s.contract.Deposit(txOps, data)
	if err != nil {
		return fmt.Errorf("deposit failed: %v", err)
	}
	log.WithField("tx", tx.Hash().Hex()).Info("Deposit transaction sent")

	return nil
}

func (s *server) Request(ctx context.Context, req *pb.PrivateKeyRequest) (*pb.PrivateKeyResponse, error) {
	s.clientLock.Lock()
	defer s.clientLock.Unlock()

	if req.NumberOfKeys == 0 {
		req.NumberOfKeys = 1
	}

	// build the list of PKs in the following order, until the requested
	// amount is ready to return.
	// - PKs already assigned to the pod
	// - PKs that have not yet been allocated
	// - PKs that are newly initialized with deposits

	pks, err := s.db.PodPKs(ctx, req.PodName)
	if err != nil {
		return nil, err
	}
	if pks != nil && len(pks.PrivateKeys) > 0 {
		log.WithField("pod", req.PodName).Debug("Returning existing assignment(s)")
		return &pb.PrivateKeyResponse{
			PrivateKeys: pks,
		}, nil
	}

	unallocated, err := s.db.UnallocatedPKs(ctx, req.NumberOfKeys)
	if err != nil {
		return nil, err
	}
	log.WithField(
		"pod", req.PodName,
	).WithField(
		"keys", len(unallocated.PrivateKeys),
	).Debug("Recycling existing private key(s)")

	pks.PrivateKeys = append(pks.PrivateKeys, unallocated.PrivateKeys...)

	if len(pks.PrivateKeys) < int(req.NumberOfKeys) {
		c := int(req.NumberOfKeys) - len(pks.PrivateKeys)
		newKeys, err := s.allocateNewKeys(ctx, req.PodName, c)
		if err != nil {
			return nil, err
		}
		pks.PrivateKeys = append(pks.PrivateKeys, newKeys.PrivateKeys...)
	}

	if err := s.db.AssignExistingPKs(ctx, pks, req.PodName); err != nil {
		return nil, err
	}

	return &pb.PrivateKeyResponse{PrivateKeys: pks}, nil
}

func (s *server) allocateNewKeys(ctx context.Context, podName string, numKeys int) (*pb.PrivateKeys, error) {
	pks := make([][]byte, numKeys)

	for i := 0; i < numKeys; i++ {
		key, err := keystore.NewKey(rand.Reader)
		if err != nil {
			return nil, err
		}

		// Make the validator deposit
		// NOTE: This uses the validator key as the withdrawal key
		di, err := keystore.DepositInput(key /*depositKey*/, key /*withdrawalKey*/)
		if err != nil {
			return nil, err
		}
		serializedData := new(bytes.Buffer)
		if err := ssz.Encode(serializedData, di); err != nil {
			return nil, fmt.Errorf("could not serialize deposit data: %v", err)
		}

		// Do the actual deposit
		if err := s.makeDeposit(serializedData.Bytes()); err != nil {
			return nil, err
		}
		// Store in database
		if err := s.db.AllocateNewPkToPod(ctx, key, podName); err != nil {
			return nil, err
		}
		secret := key.SecretKey.Marshal()
		pks[i] = secret
	}

	return &pb.PrivateKeys{PrivateKeys: pks}, nil
}
