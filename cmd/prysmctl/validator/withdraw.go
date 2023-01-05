package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/client/beacon"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/io/prompt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"
)

func setWithdrawalAddresses(c *cli.Context, r io.Reader) error {
	ctx, span := trace.StartSpan(c.Context, "withdrawal.setWithdrawalAddress")
	defer span.End()
	au := aurora.NewAurora(true)
	beaconNodeHost := c.String(BeaconHostFlag.Name)
	if !c.IsSet(PathFlag.Name) {
		return fmt.Errorf("no --%s flag value was provided", PathFlag.Name)
	}
	foundFilePaths, err := findWithdrawalFiles(c.String(PathFlag.Name))
	if err != nil {
		return errors.Wrap(err, "failed to find withdrawal files")
	}
	setWithdrawalAddressJsons := make([]*apimiddleware.SignedBLSToExecutionChangeJson, 0)
	for _, foundFilePath := range foundFilePaths {
		b, err := os.ReadFile(filepath.Clean(foundFilePath))
		if err != nil {
			return errors.Wrap(err, "failed to open file")
		}
		var to []*apimiddleware.SignedBLSToExecutionChangeJson
		if err := json.Unmarshal(b, &to); err != nil {
			log.Warnf("provided file: %s, is not a list of signed withdrawal messages", foundFilePath)
			continue
		}
		setWithdrawalAddressJsons = append(setWithdrawalAddressJsons, to...)
	}
	if len(setWithdrawalAddressJsons) == 0 {
		return errors.New("the list of signed requests is empty")
	}
	for _, request := range setWithdrawalAddressJsons {
		fmt.Println("SETTING VALIDATOR INDEX " + au.Red(request.Message.ValidatorIndex).String() + " TO WITHDRAWAL ADDRESS " + au.Red(request.Message.ToExecutionAddress).String())
	}
	return callWithdrawalEndpoints(ctx, beaconNodeHost, setWithdrawalAddressJsons)
}

func callWithdrawalEndpoints(ctx context.Context, host string, request []*apimiddleware.SignedBLSToExecutionChangeJson) error {
	client, err := beacon.NewClient(host)
	if err != nil {
		return err
	}
	if err := client.SubmitChangeBLStoExecution(ctx, request); err != nil {
		return err
	}
	log.Infof("Successfully published messages to update %d withdrawal addresses.", len(request))
	return checkIfWithdrawsAreInPool(ctx, client, request)
}

func checkIfWithdrawsAreInPool(ctx context.Context, client *beacon.Client, request []*apimiddleware.SignedBLSToExecutionChangeJson) error {
	log.Info("verifying requested withdrawal messages known to node...")
	poolResponse, err := client.GetBLStoExecutionChanges(ctx)
	if err != nil {
		return err
	}
	missingInPool := false
	for _, w := range request {
		index := sort.Search(len(poolResponse.Data), func(i int) bool {
			return poolResponse.Data[i].Message.ValidatorIndex == w.Message.ValidatorIndex &&
				poolResponse.Data[i].Message.ToExecutionAddress == w.Message.ToExecutionAddress
		})
		if index == -1 {
			log.WithFields(log.Fields{
				"validator_index":    w.Message.ValidatorIndex,
				"execution_address:": w.Message.ToExecutionAddress,
			}).Warn("set withdrawal address message not found in the node's operations pool.")
			missingInPool = true
		} else {
			log.WithFields(log.Fields{
				"validator_index":    w.Message.ValidatorIndex,
				"execution_address:": w.Message.ToExecutionAddress,
			}).Info("set withdrawal address message was found in the node's operations pool.")
		}
	}
	if missingInPool {
		log.Info("set withdrawal address messages that were not found in the pool may have been included into a block. please check before resubmitting.")
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

func withdrawalPrompt(confirmationMessage string, r io.Reader) (string, error) {
	promptQuestion := "If you still want to continue with changing the bls withdrawal address, please reenter the address you'd like to withdraw to"
	return prompt.ValidatePrompt(r, promptQuestion, func(input string) error {
		return prompt.ValidatePhrase(strings.ToLower(input), strings.ToLower(confirmationMessage))
	})
}
