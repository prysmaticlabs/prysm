package withdrawal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/io/prompt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"
)

const (
	apiPath = "/eth/v1/beacon/pool/bls_to_execution_changes"
)

func setWithdrawalAddresses(c *cli.Context, r io.Reader) error {
	ctx, span := trace.StartSpan(c.Context, "withdrawal.setWithdrawalAddress")
	defer span.End()
	beaconNodeHost := "http://localhost:3500"
	if c.String(BeaconHostFlag.Name) != "" {
		beaconNodeHost = c.String(BeaconHostFlag.Name)
	}
	u, err := url.ParseRequestURI(beaconNodeHost)
	if err != nil {
		return errors.Wrap(err, "invalid format, unable to parse url")
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("provided url %s is not in the format of http(s)://host:port", beaconNodeHost)
	}
	foundFilePaths, err := findWithdrawalFiles(c.String(FileFlag.Name))
	if err != nil {
		return errors.Wrap(err, "failed to find withdrawal files")
	}
	if len(foundFilePaths) == 0 {
		return errors.New("no compatible files were found")
	}
	au := aurora.NewAurora(true)
	fmt.Println(au.Red("===============IMPORTANT==============="))
	if !c.Bool(SkipPromptsFlag.Name) {
		fmt.Println(au.Red("Please read the following carefully"))
	}
	fmt.Print("This action will allow the partial withdraw of amounts over the 32 staked eth in your active validator balance. \n" +
		"You will also be entitled to the full withdrawal of the entire validator balance if your validator has exited. \n" +
		"The partial or full withdrawal of the validator balance may require several days of processing. \n" +
		"Please navigate to our website and make sure you understand the full implications of setting your withdrawal address. \n")
	fmt.Println(au.Red("THIS ACTION WILL NOT BE REVERSIBLE ONCE INCLUDED. "))
	fmt.Println(au.Red("You will NOT be able to change the address again once changed. "))

	setWithdrawalAddressJsons := make([]*apimiddleware.SignedBLSToExecutionChangeJson, 0)
	for _, foundFilePath := range foundFilePaths {
		b, err := os.ReadFile(filepath.Clean(foundFilePath))
		if err != nil {
			return errors.Wrap(err, "failed to open file")
		}
		if string(b)[0:1] != "[" {
			log.Warnf("provided file: %s, is not a list \n", foundFilePath)
			continue
		}
		var to []*apimiddleware.SignedBLSToExecutionChangeJson
		if err := json.Unmarshal(b, &to); err != nil {
			return errors.Wrap(err, "failed to unmarshal file")
		}
		setWithdrawalAddressJsons = append(setWithdrawalAddressJsons, to...)
	}
	if len(setWithdrawalAddressJsons) == 0 {
		return errors.New("the list of signed requests is empty")
	}
	if !c.Bool(SkipPromptsFlag.Name) {
		for _, jsonOb := range setWithdrawalAddressJsons {
			if err := verifyWithdrawalCertainty(r, jsonOb); err != nil {
				return err
			}
		}
	}
	return callWithdrawalEndpoint(ctx, beaconNodeHost, setWithdrawalAddressJsons)
}

func verifyWithdrawalCertainty(r io.Reader, request *apimiddleware.SignedBLSToExecutionChangeJson) error {
	au := aurora.NewAurora(true)
	withdrawalConfirmation := request.Message.ToExecutionAddress
	fmt.Println(au.Red("===================================="))
	fmt.Println("YOU ARE ATTEMPTING TO CHANGE THE BLS WITHDRAWAL(" + au.Red(request.Message.FromBLSPubkey).String() + ") ADDRESS " +
		"TO AN ETHEREUM ADDRESS(" + au.Red(request.Message.ToExecutionAddress).String() + ") FOR VALIDATOR INDEX(" + au.Red(request.Message.ValidatorIndex).String() + "). ")
	_, err := withdrawalPrompt(withdrawalConfirmation, r)
	if err != nil {
		return err
	}
	return nil
}

func callWithdrawalEndpoint(ctx context.Context, host string, request []*apimiddleware.SignedBLSToExecutionChangeJson) error {
	body, err := json.Marshal(request)
	if err != nil {
		return errors.Wrap(err, "failed to marshal json")
	}
	fullpath := host + apiPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullpath, bytes.NewBuffer(body))
	if err != nil {
		return errors.Wrap(err, "invalid format, failed to create new Post Request Object")
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request to %s responded with a status other than OK - status %v, body %v", fullpath, resp.Status, resp.Body)
	}
	log.Infof("Successfully published messages to update %d withdrawal addresses.", len(request))

	log.Info("retrieving list of withdrawal messages known to node...")
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, fullpath, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request to %s responded with a status other than OK - status %v, body %v", fullpath, resp.Status, resp.Body)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.WithError(err).Error("could not close response body")
		}
	}(resp.Body)
	poolResponse := &apimiddleware.BLSToExecutionChangesPoolResponseJson{}
	if err := json.NewDecoder(resp.Body).Decode(poolResponse); err != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "failed to read response body")
		}
		return errors.Wrap(err, fmt.Sprintf("invalid format, unable to read response body: %v", string(body)))
	}
	log.Infoln("known withdrawal messages to node, but not necessarily incorporated into any block yet: ")
	for _, signedMessage := range poolResponse.Data {
		log.Infof("validator index: %s with set withdrawal address: 0x%s", signedMessage.Message.ValidatorIndex, signedMessage.Message.ToExecutionAddress)
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

	return foundpaths, nil
}

func withdrawalPrompt(confirmationMessage string, r io.Reader) (string, error) {
	promptQuestion := "If you still want to continue with changing the bls withdrawal address, please reenter the address you'd like to withdraw to"
	return prompt.ValidatePrompt(r, promptQuestion, func(input string) error {
		return prompt.ValidatePhrase(strings.ToLower(input), strings.ToLower(confirmationMessage))
	})
}
