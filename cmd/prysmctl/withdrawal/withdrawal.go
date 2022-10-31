package withdrawal

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/io/prompt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"
	"gopkg.in/yaml.v2"
)

var withdrawalFlags = struct {
	BeaconNodeHost string
	File           string
}{}

func setWithdrawlAddress(c *cli.Context, r io.Reader) error {
	apiPath := "/blsToExecutionAddress"
	f := withdrawalFlags
	cleanpath := filepath.Clean(f.File)
	b, err := os.ReadFile(cleanpath)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	var to BlsToExecutionEngineFile
	if err := yaml.Unmarshal(b, to); err != nil {
		return errors.Wrap(err, "failed to unmarshal file")
	}
	if to.Message == nil {
		return errors.New("the message field in file is empty")
	}
	u, err := url.ParseRequestURI(f.BeaconNodeHost)
	if err != nil {
		return errors.Wrap(err, "invalid format, unable to parse url")
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("url must be in the format of http(s)://host:port url used: %v", f.BeaconNodeHost)
	}

	withdrawalConfirmation := to.Message.ToExecutionAddress

	withdraw, err := withdrawalPrompt(withdrawalConfirmation, r)
	if err != nil {
		return err
	}

	if !strings.EqualFold(withdraw, "n") {
		log.Warn("Did not provide the correct acceptance message")
		return nil
	}

	ctx, span := trace.StartSpan(c.Context, "withdrawal.blsToExecutionAddress")
	defer span.End()

	fullpath := f.BeaconNodeHost + apiPath

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullpath, bytes.NewBuffer(b)) //TODO:change this b
	if err != nil {
		return errors.Wrap(err, "invalid format, failed to create new Post Request Object")
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	//start := time.Now()
	resp, err := client.Do(req)
	//duration := time.Since(start)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request to %s , responded with a status other than OK, status: %v", fullpath, resp.Status)
	}
	log.Info("Successfully published message to update withdrawal address.")
	return nil
}

func withdrawalPrompt(confirmationMessage string, r io.Reader) (string, error) {

	au := aurora.NewAurora(true)
	promptHeader := au.Red("===============IMPORTANT===============")
	promptDescription := "Withdrawing funds is not yet possible. " +
		"Please navigate to the following website and make sure you understand the current implications " +
		"of a voluntary exit before making the final decision:"
	promptURL := au.Blue("https://docs.prylabs.network/docs/wallet/exiting-a-validator/#withdrawal-delay-warning")
	promptQuestion := "If you still want to continue with the voluntary exit, please input a phrase found at the end " +
		"of the page from the above URL"
	promptText := fmt.Sprintf("%s\n%s\n%s\n%s", promptHeader, promptDescription, promptURL, promptQuestion)
	return prompt.ValidatePrompt(r, promptText, func(input string) error {
		return prompt.ValidatePhrase(input, confirmationMessage)
	})

}
