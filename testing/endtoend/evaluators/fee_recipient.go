package evaluators

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/interop"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/components"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/v3/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/policies"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var FeeRecipientIsPresent = types.Evaluator{
	Name:       "fee_recipient_is_present_%d",
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

	validatorNum := int(params.BeaconConfig().MinGenesisActiveValidatorCount)
	_, pubs, err := interop.DeterministicallyGenerateKeys(uint64(0), uint64(validatorNum+int(e2e.DepositCount))) // matches validator start in validator component + validators used for deposits
	if err != nil {
		return err
	}
	lighthouseKeys := []bls.PublicKey{}
	if e2e.TestParams.LighthouseBeaconNodeCount != 0 {
		totalNodecount := e2e.TestParams.BeaconNodeCount + e2e.TestParams.LighthouseBeaconNodeCount
		valPerNode := validatorNum / totalNodecount
		lighthouseOffset := valPerNode * e2e.TestParams.BeaconNodeCount

		_, lighthouseKeys, err = interop.DeterministicallyGenerateKeys(uint64(lighthouseOffset), uint64(valPerNode*e2e.TestParams.LighthouseBeaconNodeCount))
		if err != nil {
			return err
		}
	}

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
			isDeterministicKey := false
			isLighthouseKey := false

			// If lighthouse keys are present, we skip the check.
			for _, pub := range lighthouseKeys {
				if hexutil.Encode(publickey) == hexutil.Encode(pub.Marshal()) {
					isLighthouseKey = true
					break
				}
			}
			if isLighthouseKey {
				continue
			}
			for _, pub := range pubs {
				if hexutil.Encode(publickey) == hexutil.Encode(pub.Marshal()) {
					isDeterministicKey = true
					break
				}
			}
			// calculate deterministic fee recipient using first 20 bytes of public key
			deterministicFeeRecipient := common.HexToAddress(hexutil.Encode(publickey[:fieldparams.FeeRecipientLength])).Hex()
			if isDeterministicKey && deterministicFeeRecipient != account.Hex() {
				return fmt.Errorf("publickey %s, fee recipient %s does not match the proposer settings fee recipient %s",
					hexutil.Encode(publickey), account.Hex(), deterministicFeeRecipient)
			}
			if !isDeterministicKey && components.DefaultFeeRecipientAddress != account.Hex() {
				return fmt.Errorf("publickey %s, fee recipient %s does not match the default fee recipient %s",
					hexutil.Encode(publickey), account.Hex(), components.DefaultFeeRecipientAddress)
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
				return errors.Errorf("account balance didn't change after applying fee recipient for account: %s", account.Hex())
			}
		}
	}

	return nil
}
