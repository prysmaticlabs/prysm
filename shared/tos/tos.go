package tos

import (
	"io/ioutil"
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
Prysmatic Labs 

TERMS AND CONDITIONS: https://docs.prylabs.network/docs/licenses/prysmatic-labs

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

Type "yes" to accept this terms and conditions[yes/no]:`
)

var (
	au  = aurora.NewAurora(true)
	log = logrus.WithField("prefix", "prompt")
)

// VerifyTosAcceptedOrPrompt check if Tos was accepted before or asks to accept
func VerifyTosAcceptedOrPrompt(ctx *cli.Context) (bool, error) {
	if fileutil.FileExists(ctx.String(cmd.DataDirFlag.Name) + "/" + acceptTosFilename) {
		return true, nil
	}
	if ctx.Bool(cmd.AcceptTosFlag.Name) {
		saveTosAccepted(ctx)
		return true, nil
	}
	input, err := promptutil.DefaultPrompt(au.Bold(acceptTosPromptText).String(), "no")
	if strings.ToLower(input) == "yes" && err == nil {
		saveTosAccepted(ctx)
		return true, nil
	}
	return false, err
}

// saveTosAccepted creates a file when Tos accepted
func saveTosAccepted(ctx *cli.Context) {
	err := ioutil.WriteFile(ctx.String(cmd.DataDirFlag.Name)+"/"+acceptTosFilename, []byte(""), 0644)
	if err != nil {
		log.WithError(err).Warnf("error writing %s to file: %s", cmd.AcceptTosFlag.Name, ctx.String(cmd.DataDirFlag.Name)+"/"+acceptTosFilename)
	}
}
