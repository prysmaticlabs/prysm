package evaluators

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/params"
	validator_service_config "github.com/prysmaticlabs/prysm/config/validator/service"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
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

	var configFiles []*validator_service_config.ProposerSettingsPayload
	testNetDir := e2e.TestParams.TestPath + "/proposer-settings"
	dirs, err := ioutil.ReadDir(filepath.Clean(testNetDir))
	if err != nil {
		return err
	}
	for _, f := range dirs {
		if !f.IsDir() {
			configPath := filepath.Join(testNetDir, filepath.Clean(f.Name()+"/config.json"))
			jsonFile, err := os.Open(filepath.Clean(configPath))
			if err != nil {
				return err
			}
			var configFile validator_service_config.ProposerSettingsPayload
			if err := json.NewDecoder(jsonFile).Decode(&configFile); err != nil {
				return err
			}
			configFiles = append(configFiles, &configFile)
		}
	}

	var account common.Address
	for _, ctr := range blks.BlockContainers {
		switch ctr.Block.(type) {
		case *ethpb.BeaconBlockContainer_BellatrixBlock:
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
			usesDefaultFeeRecipient := false
			for _, config := range configFiles {
				option, ok := config.ProposerConfig[hexutil.Encode(publickey)]
				if ok {
					if option.FeeRecipient != account.Hex() {
						return fmt.Errorf("fee recipient %s does not match the proposer settings fee recipient %s", account.Hex(), option.FeeRecipient)
					}
					usesDefaultFeeRecipient = true
				}
			}
			if !usesDefaultFeeRecipient {
				if configFiles[0].DefaultConfig.FeeRecipient != account.Hex() {
					return fmt.Errorf("fee recipient %s does not match the default fee recipient %s", account.Hex(), configFiles[0].DefaultConfig.FeeRecipient)
				}
			}

		}

	}

	latestBlockNum, err := web3.BlockNumber(ctx)
	if err != nil {
		return err
	}
	accountBalance, err := web3.BalanceAt(ctx, account, big.NewInt(0).SetUint64(latestBlockNum))
	if err != nil {
		return err
	}
	prevAccountBalance, err := web3.BalanceAt(ctx, account, big.NewInt(0).SetUint64(latestBlockNum-1))
	if err != nil {
		return err
	}
	if accountBalance.Uint64() <= prevAccountBalance.Uint64() {
		return errors.Errorf("account balance didn't change after applying fee recipient for account: %s", account.Hex())
	} else {
		log.Infof("current account balance %v ,increased from previous account balance %v ", accountBalance, prevAccountBalance)
	}

	return nil
}
