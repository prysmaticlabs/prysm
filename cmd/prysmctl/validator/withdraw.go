package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/client/beacon"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"
)

func setWithdrawalAddresses(c *cli.Context) error {
	ctx, span := trace.StartSpan(c.Context, "withdrawal.setWithdrawalAddresses")
	defer span.End()
	au := aurora.NewAurora(true)
	beaconNodeHost := c.String(BeaconHostFlag.Name)
	if !c.IsSet(PathFlag.Name) {
		return fmt.Errorf("no --%s flag value was provided", PathFlag.Name)
	}
	setWithdrawalAddressJsons, err := getWithdrawalMessagesFromPathFlag(c)
	if err != nil {
		return err
	}
	for _, request := range setWithdrawalAddressJsons {
		fmt.Println("SETTING VALIDATOR INDEX " + au.Red(request.Message.ValidatorIndex).String() + " TO WITHDRAWAL ADDRESS " + au.Red(request.Message.ToExecutionAddress).String())
	}
	return callWithdrawalEndpoints(ctx, beaconNodeHost, setWithdrawalAddressJsons)
}

func getWithdrawalMessagesFromPathFlag(c *cli.Context) ([]*apimiddleware.SignedBLSToExecutionChangeJson, error) {
	setWithdrawalAddressJsons := make([]*apimiddleware.SignedBLSToExecutionChangeJson, 0)
	foundFilePaths, err := findWithdrawalFiles(c.String(PathFlag.Name))
	if err != nil {
		return setWithdrawalAddressJsons, errors.Wrap(err, "failed to find withdrawal files")
	}
	for _, foundFilePath := range foundFilePaths {
		b, err := os.ReadFile(filepath.Clean(foundFilePath))
		if err != nil {
			return setWithdrawalAddressJsons, errors.Wrap(err, "failed to open file")
		}
		var to []*apimiddleware.SignedBLSToExecutionChangeJson
		if err := json.Unmarshal(b, &to); err != nil {
			log.Warnf("provided file: %s, is not a list of signed withdrawal messages", foundFilePath)
			continue
		}
		// verify 0x from file and add if needed
		for i, obj := range to {
			if len(obj.Message.FromBLSPubkey) == fieldparams.BLSPubkeyLength*2 {
				to[i].Message.FromBLSPubkey = fmt.Sprintf("0x%s", obj.Message.FromBLSPubkey)
			}
			if len(obj.Message.ToExecutionAddress) == common.AddressLength*2 {
				to[i].Message.ToExecutionAddress = fmt.Sprintf("0x%s", obj.Message.ToExecutionAddress)
			}
			if len(obj.Signature) == fieldparams.BLSSignatureLength*2 {
				to[i].Signature = fmt.Sprintf("0x%s", obj.Signature)
			}
			setWithdrawalAddressJsons = append(setWithdrawalAddressJsons, &apimiddleware.SignedBLSToExecutionChangeJson{
				Message: &apimiddleware.BLSToExecutionChangeJson{
					ValidatorIndex:     to[i].Message.ValidatorIndex,
					FromBLSPubkey:      to[i].Message.FromBLSPubkey,
					ToExecutionAddress: to[i].Message.ToExecutionAddress,
				},
				Signature: to[i].Signature,
			})
		}

	}
	if len(setWithdrawalAddressJsons) == 0 {
		return setWithdrawalAddressJsons, errors.New("the list of signed requests is empty")
	}
	return setWithdrawalAddressJsons, nil
}

func callWithdrawalEndpoints(ctx context.Context, host string, request []*apimiddleware.SignedBLSToExecutionChangeJson) error {
	client, err := beacon.NewClient(host)
	if err != nil {
		return err
	}
	fork, err := client.GetFork(ctx, "head")
	if err != nil {
		return errors.Wrap(err, "could not retrieve current fork information")
	}
	if !(params.BeaconConfig().ForkVersionSchedule[bytesutil.ToBytes4(fork.CurrentVersion)] >= params.BeaconConfig().CapellaForkEpoch) {
		return errors.New("setting withdrawals using the BLStoExecutionChange endpoint is only available after the Capella/Shanghai hard fork.")
	}
	err = client.SubmitChangeBLStoExecution(ctx, request)
	if err != nil && strings.Contains(err.Error(), "POST error") {
		// just log the error, so we can check the pool for partial inclusions.
		log.Error(err)
	} else if err != nil {
		return err
	} else {
		log.Infof("Successfully published messages to update %d withdrawal addresses.", len(request))
	}
	return checkIfWithdrawsAreInPool(ctx, client, request)
}

func checkIfWithdrawsAreInPool(ctx context.Context, client *beacon.Client, request []*apimiddleware.SignedBLSToExecutionChangeJson) error {
	log.Info("Verifying requested withdrawal messages known to node...")
	poolResponse, err := client.GetBLStoExecutionChanges(ctx)
	if err != nil {
		return err
	}
	requestMap := make(map[string]string)
	for _, w := range request {
		requestMap[w.Message.ValidatorIndex] = w.Message.ToExecutionAddress
	}
	totalMessages := len(requestMap)
	log.Infof("There are a total of %d messages known to the node's pool.", len(poolResponse.Data))
	for _, resp := range poolResponse.Data {
		value, found := requestMap[resp.Message.ValidatorIndex]
		if found && value == resp.Message.ToExecutionAddress {
			delete(requestMap, resp.Message.ValidatorIndex)
		}
	}
	if len(requestMap) != 0 {
		for key, address := range requestMap {
			log.WithFields(log.Fields{
				"validator_index":    key,
				"execution_address:": address,
			}).Warn("Set withdrawal address message not found in the node's operations pool.")
		}
		log.Warn("Please check before resubmitting. Set withdrawal address messages that were not found in the pool may have been already included into a block.")
	} else {
		log.Infof("All (total:%d) signed withdrawal messages were found in the pool.", totalMessages)
	}
	return nil
}

func findWithdrawalFiles(path string) ([]string, error) {
	var foundpaths []string
	maxdepth := 3
	cleanpath := filepath.Clean(path)
	if err := filepath.WalkDir(cleanpath, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() && strings.Count(cleanpath, string(os.PathSeparator)) > maxdepth {
			return fs.SkipDir
		}

		if filepath.Ext(d.Name()) == ".json" {
			foundpaths = append(foundpaths, s)
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "unable to find compatible files")
	}
	if len(foundpaths) == 0 {
		return nil, errors.New("no compatible files were found")
	}
	log.Infof("found JSON files for setting withdrawals: %v", foundpaths)
	return foundpaths, nil
}

func verifyWithdrawalsInPool(c *cli.Context) error {
	ctx, span := trace.StartSpan(c.Context, "withdrawal.verifyWithdrawalsInPool")
	defer span.End()
	beaconNodeHost := c.String(BeaconHostFlag.Name)
	if !c.IsSet(PathFlag.Name) {
		return fmt.Errorf("no --%s flag value was provided", PathFlag.Name)
	}
	client, err := beacon.NewClient(beaconNodeHost)
	if err != nil {
		return err
	}

	request, err := getWithdrawalMessagesFromPathFlag(c)
	if err != nil {
		return err
	}
	return checkIfWithdrawsAreInPool(ctx, client, request)
}
