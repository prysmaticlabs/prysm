package evaluators

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/endtoend/components"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/testing/endtoend/policies"
	"github.com/prysmaticlabs/prysm/testing/endtoend/types"
	log "github.com/sirupsen/logrus"
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

	for _, ctr := range blks.BlockContainers {
		switch ctr.Block.(type) {
		case *ethpb.BeaconBlockContainer_BellatrixBlock:
			var account common.Address

			fr := ctr.GetBellatrixBlock().Block.Body.ExecutionPayload.FeeRecipient
			if len(fr) != 0 && hexutil.Encode(fr) != params.BeaconConfig().EthBurnAddressHex {
				account = common.BytesToAddress(fr)
			} else {
				return errors.New("fee recipient is not set")
			}
			validatorRequest := &ethpb.GetValidatorRequest{
				QueryFilter: &ethpb.GetValidatorRequest_Index{
					Index: ctr.GetBellatrixBlock().Block.ProposerIndex,
				},
			}
			validator, err := client.GetValidator(context.Background(), validatorRequest)
			if err != nil {
				return errors.Wrap(err, "failed to get validators")
			}
			publickey := validator.GetPublicKey()
			// calculate deterministic fee recipient using first 20 bytes of public key
			deterministicFeeRecipient := common.HexToAddress(hexutil.Encode(publickey[:fieldparams.FeeRecipientLength])).Hex()
			if deterministicFeeRecipient != account.Hex() {
				return fmt.Errorf("fee recipient %s does not match the proposer settings fee recipient %s", account.Hex(), deterministicFeeRecipient)
			} else {

				if components.DefaultFeeRecipientAddress != account.Hex() {
					return fmt.Errorf("fee recipient %s does not match the default fee recipient %s", account.Hex(), components.DefaultFeeRecipientAddress)
				}
			}
			currentBlock, err := web3.BlockByHash(ctx, common.BytesToHash(ctr.GetBellatrixBlock().GetBlock().GetBody().GetExecutionPayload().BlockHash))
			if err != nil {
				return err
			}

			accountBalance, err := web3.BalanceAt(ctx, account, currentBlock.Number())
			if err != nil {
				return err
			}
			previousBlock, err := web3.BlockByHash(ctx, common.BytesToHash(ctr.GetBellatrixBlock().GetBlock().GetBody().GetExecutionPayload().ParentHash))
			if err != nil {
				return err
			}
			prevAccountBalance, err := web3.BalanceAt(ctx, account, previousBlock.Number())
			if err != nil {
				return err
			}
			if currentBlock.GasUsed() > 0 && accountBalance.Uint64() <= prevAccountBalance.Uint64() {
				log.Infof("current block num: %d , previous block num: %d , account balance: %d,  pre account balance %d", currentBlock.Number(), previousBlock.Number(), accountBalance, prevAccountBalance)
				return errors.Errorf("account balance didn't change after applying fee recipient for account: %s", account.Hex())
			} else {
				log.Infof("current gas used: %v current account balance %v ,increased from previous account balance %v ", currentBlock.GasUsed(), accountBalance, prevAccountBalance)
			}
		}

	}

	return nil
}
