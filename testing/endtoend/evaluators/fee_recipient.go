package evaluators

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/testing/endtoend/policies"
	"github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var FeeRecipientIsPresent = types.Evaluator{
	Name:       "Fee_Recipient_Is_Present_%d",
	Policy:     policies.AfterNthEpoch(helpers.BellatrixE2EForkEpoch),
	Evaluation: feeRecipientIsPresent,
}

func feeRecipientIsPresent(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := ethpb.NewBeaconChainClient(conn)
	chainHead, err := client.GetChainHead(context.Background(), &emptypb.Empty{})
	if err != nil {
		return errors.Wrap(err, "failed to get chain head")
	}
	req := &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: chainHead.HeadEpoch.Sub(1)}}
	blks, err := client.ListBeaconBlocks(context.Background(), req)
	if err != nil {
		return errors.Wrap(err, "failed to list blocks")
	}

	rpcclient, err := rpc.DialHTTP(fmt.Sprintf("http://127.0.0.1:%d", e2e.TestParams.Ports.Eth1RPCPort))
	if err != nil {
		return err
	}
	defer rpcclient.Close()
	web3 := ethclient.NewClient(rpcclient)
	ctx := context.Background()
	latestBlockNum, err := web3.BlockNumber(ctx)
	if err != nil {
		return err
	}
	var account common.Address
	// check if fee recipient is set
	isFeeRecipientPresent := false
	for _, ctr := range blks.BlockContainers {
		switch ctr.Block.(type) {
		case *ethpb.BeaconBlockContainer_BellatrixBlock:
			fr := ctr.GetBellatrixBlock().Block.Body.ExecutionPayload.FeeRecipient
			if len(fr) != 0 && hexutil.Encode(fr) != params.BeaconConfig().EthBurnAddressHex {
				isFeeRecipientPresent = true
				account = common.BytesToAddress(fr)
			}
		}
		if isFeeRecipientPresent {
			break
		}
	}
	if !isFeeRecipientPresent {
		return errors.New("fee recipient is not set")
	}

	if !bytesutil.ZeroRoot(account.Bytes()) {
		accountBalance, err := web3.BalanceAt(ctx, account, big.NewInt(int64(latestBlockNum)))
		if err != nil {
			return err
		}
		prevAccountBalance, err := web3.BalanceAt(ctx, account, big.NewInt(int64(latestBlockNum-1)))
		if err != nil {
			return err
		}
		if accountBalance.Uint64() <= prevAccountBalance.Uint64() {
			return errors.Errorf("account balance didn't change after applying fee recipient for account: %s", account.Hex())
		}
	}

	return nil
}
