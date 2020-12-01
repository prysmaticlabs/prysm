package tos

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const (
	acceptTosFilename   = "tosaccepted"
	acceptTosPromptText = `
Prysmatic Labs Terms of Use

By downloading, accessing or using the Prysm implementation (“Prysm”), you (referenced herein
as “you” or the “user”) certify that you have read and agreed to the terms and conditions below.

TERMS AND CONDITIONS: https://github.com/prysmaticlabs/prysm/blob/master/TERMS_OF_SERVICE.md


Type "accept" to accept this terms and conditions [accept/decline]:`
	acceptTosPromptErrText = `could not scan text input, if you are trying to run in non-interactive environment, you
can use the --accept-terms-of-use flag after reading the terms and conditions here: 
https://github.com/prysmaticlabs/prysm/blob/master/TERMS_OF_SERVICE.md`
)

var (
	au  = aurora.NewAurora(true)
	log = logrus.WithField("prefix", "tos")
)

// VerifyTosAcceptedOrPrompt check if Tos was accepted before or asks to accept.
func VerifyTosAcceptedOrPrompt(ctx *cli.Context) error {
	if ctx.Bool(cmd.E2EConfigFlag.Name) {
		return nil
	}

	if fileutil.FileExists(filepath.Join(ctx.String(cmd.DataDirFlag.Name), acceptTosFilename)) {
		return nil
	}
	if ctx.Bool(cmd.AcceptTosFlag.Name) {
		saveTosAccepted(ctx)
		return nil
	}

	input, err := promptutil.DefaultPrompt(au.Bold(acceptTosPromptText).String(), "decline")
	if err != nil {
		return errors.New(acceptTosPromptErrText)
	}
	if strings.ToLower(input) != "accept" {
		return errors.New("you have to accept Terms and Conditions in order to continue")
	}

	saveTosAccepted(ctx)
	return nil
}

// saveTosAccepted creates a file when Tos accepted.
func saveTosAccepted(ctx *cli.Context) {
	if err := fileutil.MkdirAll(ctx.String(cmd.DataDirFlag.Name)); err != nil {
		log.WithError(err).Warnf("error creating directory: %s", ctx.String(cmd.DataDirFlag.Name))
	}
	err := fileutil.WriteFile(filepath.Join(ctx.String(cmd.DataDirFlag.Name), acceptTosFilename), []byte(""))
	if err != nil {
		log.WithError(err).Warnf("error writing %s to file: %s", cmd.AcceptTosFlag.Name, filepath.Join(ctx.String(cmd.DataDirFlag.Name), acceptTosFilename))
	}
}
