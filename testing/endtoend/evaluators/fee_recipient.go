package evaluators

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
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
	// check if fee recipient is set
	isFeeRecipientPresent := false
	for _, ctr := range blks.BlockContainers {
		switch ctr.Block.(type) {
		case *ethpb.BeaconBlockContainer_BellatrixBlock:
			fr := ctr.GetBellatrixBlock().Block.Body.ExecutionPayload.FeeRecipient
			if len(fr) != 0 && hexutil.Encode(fr) != params.BeaconConfig().EthBurnAddressHex {
				isFeeRecipientPresent = true
			}
		}
		if isFeeRecipientPresent {
			break
		}
	}
	if !isFeeRecipientPresent {
		return errors.New("fee recipient is not set")
	}

	//rpcclient, err := rpc.DialHTTP(fmt.Sprintf("http://127.0.0.1:%d", e2e.TestParams.Ports.Eth1RPCPort))
	//if err != nil {
	//	return err
	//}
	//defer rpcclient.Close()
	//web3 := ethclient.NewClient(rpcclient)
	//ctx := context.Background()
	//latestBlockNum, _ := web3.BlockNumber(ctx)
	//account := common.Address{1}
	//accountBalance, _ := web3.BalanceAt(ctx, account, big.NewInt(int64(latestBlockNum)))
	//
	//if accountBalance.Uint64() < prevAccountBalance {
	//
	//}
	return nil
}
