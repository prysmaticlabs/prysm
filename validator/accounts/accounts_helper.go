package accounts

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/urfave/cli/v2"
)

func filterPublicKeysFromUserInput(
	cliCtx *cli.Context,
	publicKeysFlag *cli.StringFlag,
	validatingPublicKeys [][fieldparams.BLSPubkeyLength]byte,
	selectionPrompt string,
) ([]bls.PublicKey, error) {
	var filteredPubKeys []bls.PublicKey
	if cliCtx.IsSet(publicKeysFlag.Name) {
		pubKeyStrings := strings.Split(cliCtx.String(publicKeysFlag.Name), ",")
		if len(pubKeyStrings) == 0 {
			return nil, fmt.Errorf(
				"could not parse %s. It must be a string of comma-separated hex strings",
				publicKeysFlag.Name,
			)
		}
		for _, str := range pubKeyStrings {
			pkString := str
			if strings.Contains(pkString, "0x") {
				pkString = pkString[2:]
			}
			pubKeyBytes, err := hex.DecodeString(pkString)
			if err != nil {
				return nil, errors.Wrapf(err, "could not decode string %s as hex", pkString)
			}
			blsPublicKey, err := bls.PublicKeyFromBytes(pubKeyBytes)
			if err != nil {
				return nil, errors.Wrapf(err, "%#x is not a valid BLS public key", pubKeyBytes)
			}
			filteredPubKeys = append(filteredPubKeys, blsPublicKey)
		}
		return filteredPubKeys, nil
	}
	return selectAccounts(selectionPrompt, validatingPublicKeys)
}
