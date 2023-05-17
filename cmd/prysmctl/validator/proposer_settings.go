package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/v4/api/client"
	"github.com/prysmaticlabs/prysm/v4/api/client/validator"
	"github.com/prysmaticlabs/prysm/v4/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/io/file"
	"github.com/prysmaticlabs/prysm/v4/io/prompt"
	validatorpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1/validator-client"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"
)

func getProposerSettings(c *cli.Context, r io.Reader) error {
	ctx, span := trace.StartSpan(c.Context, "prysmctl.getProposerSettings")
	defer span.End()
	if !c.IsSet(ValidatorHostFlag.Name) {
		return fmt.Errorf("no --%s flag value was provided", ValidatorHostFlag.Name)
	}
	if !c.IsSet(TokenFlag.Name) {
		return fmt.Errorf("no --%s flag value was provided", TokenFlag.Name)
	}
	defaultFeeRecipient := params.BeaconConfig().DefaultFeeRecipient.Hex()
	if c.IsSet(ProposerSettingsOutputFlag.Name) {
		if c.IsSet(DefaultFeeRecipientFlag.Name) {
			recipient := c.String(DefaultFeeRecipientFlag.Name)
			if err := validateIsExecutionAddress(recipient); err != nil {
				return err
			}
			defaultFeeRecipient = recipient
		} else {
			promptText := "please enter a default fee recipient address (an ethereum address in hex format)"
			resp, err := prompt.ValidatePrompt(r, promptText, validateIsExecutionAddress)
			if err != nil {
				return err
			}
			defaultFeeRecipient = resp
		}
	}

	cl, err := validator.NewClient(c.String(ValidatorHostFlag.Name), client.WithAuthenticationToken(c.String(TokenFlag.Name)))
	if err != nil {
		return err
	}
	validators, err := cl.GetValidatorPubKeys(ctx)
	if err != nil {
		return err
	}
	feeRecipients, err := cl.GetFeeRecipientAddresses(ctx, validators)
	if err != nil {
		return err
	}

	log.Infoln("===============DISPLAYING CURRENT PROPOSER SETTINGS===============")

	for index := range validators {
		log.Infof("validator: %s. fee-recipient: %s", validators[index], feeRecipients[index])
	}

	if c.IsSet(ProposerSettingsOutputFlag.Name) {
		log.Infof("the default fee recipient is set to %s", defaultFeeRecipient)
		proposerConfig := make(map[string]*validatorpb.ProposerOptionPayload)
		for index, val := range validators {
			proposerConfig[val] = &validatorpb.ProposerOptionPayload{
				FeeRecipient: feeRecipients[index],
			}
		}
		fileConfig := &validatorpb.ProposerSettingsPayload{
			ProposerConfig: proposerConfig,
			DefaultConfig: &validatorpb.ProposerOptionPayload{
				FeeRecipient: defaultFeeRecipient,
			},
		}
		b, err := json.Marshal(fileConfig)
		if err != nil {
			return err
		}
		if err := file.WriteFile(c.String(ProposerSettingsOutputFlag.Name), b); err != nil {
			return err
		}
		log.Infof("successfully created `%s`. settings can be imported into validator client using --%s flag. does not include custom builder settings.", c.String(ProposerSettingsOutputFlag.Name), flags.ProposerSettingsFlag.Name)
	}

	return nil
}

func validateIsExecutionAddress(input string) error {
	if input[0:2] != "0x" || !(len(input) == common.AddressLength*2+2) {
		return errors.New("no default address entered")
	}
	return nil
}
