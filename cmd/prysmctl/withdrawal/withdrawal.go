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
	basePath = "/eth/v1"
	apiPath  = "/beacon/pool/bls_to_execution_changes"
)

var withdrawalFlags = struct {
	BeaconNodeHost string
	File           string
}{}

func setWithdrawalAddress(c *cli.Context, r io.Reader) error {
	ctx, span := trace.StartSpan(c.Context, "withdrawal.blsToExecutionAddress")
	defer span.End()
	f := withdrawalFlags
	u, err := url.ParseRequestURI(f.BeaconNodeHost)
	if err != nil {
		return errors.Wrap(err, "invalid format, unable to parse url")
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("url must be in the format of http(s)://host:port url used: %v", f.BeaconNodeHost)
	}
	foundFilePaths, err := findWithdrawalFiles(f.File)
	if err != nil {
		return err
	}
	if len(foundFilePaths) == 0 {
		return errors.New("no compatible files were found")
	}
	au := aurora.NewAurora(true)
	fmt.Println(au.Red("===============IMPORTANT==============="))
	fmt.Println(au.Red("Please read the following carefully"))
	fmt.Println("This action will allow you to partially withdraw any amount over the 32 staked eth in your validator balance. " +
		"You will also be entitled to the full withdrawal if your validator has exited. " +
		"Please navigate to the following website and make sure you understand the current implications " +
		"of changing your bls withdrawal address to an ethereum address. " +
		"THIS ACTION WILL NOT BE REVERSIBLE ONCE INCLUDED. " +
		"You will NOT be able to change the address again once changed. ")
	for _, foundFilePath := range foundFilePaths {
		b, err := os.ReadFile(filepath.Clean(foundFilePath))
		if err != nil {
			return errors.Wrap(err, "failed to open file")
		}
		switch string(b)[0:1] {
		case "[":
			var to []*apimiddleware.SignedBLSToExecutionChangeJson
			if err := json.Unmarshal(b, &to); err != nil {
				return errors.Wrap(err, "failed to unmarshal file")
			}
			if len(to) == 0 {
				return errors.New("the list of signed requests is empty")
			}
			for _, jsonOb := range to {
				if err := callWithdrawalEndpoint(ctx, f.BeaconNodeHost, r, jsonOb); err != nil {
					return err
				}
			}
		case "{":
			var to *apimiddleware.SignedBLSToExecutionChangeJson
			if err := json.Unmarshal(b, &to); err != nil {
				return errors.Wrap(err, "failed to unmarshal file")
			}
			if to == nil || to.Message == nil {
				return errors.New("the object or object's message field in file is empty")
			}
			return callWithdrawalEndpoint(ctx, f.BeaconNodeHost, r, to)
		default:
			return errors.New("the provided file is not a json object or list of jason objects")
		}
	}
	return nil
}

func callWithdrawalEndpoint(ctx context.Context, host string, r io.Reader, request *apimiddleware.SignedBLSToExecutionChangeJson) error {
	au := aurora.NewAurora(true)
	withdrawalConfirmation := request.Message.ToExecutionAddress
	fmt.Println(au.Red("===================================="))
	fmt.Println("YOU ARE ATTEMPTING TO CHANGE THE BLS WITHDRAWAL(" + au.Red(request.Message.FromBLSPubkey).String() + ") ADDRESS " +
		"TO AN ETHEREUM ADDRESS(" + au.Red(request.Message.ToExecutionAddress).String() + ") FOR VALIDATOR INDEX(" + au.Red(request.Message.ValidatorIndex).String() + "). ")
	_, err := withdrawalPrompt(withdrawalConfirmation, r)
	if err != nil {
		return err
	}
	body, err := json.Marshal(request)
	if err != nil {
		return errors.Wrap(err, "failed to marshal json")
	}
	fullpath := host + basePath + apiPath
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
		return fmt.Errorf("API request to %s , responded with a status other than OK, status: %v body: %v", fullpath, resp.Status, resp.Body)
	}
	log.Info("Successfully published message to update withdrawal address.")
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
