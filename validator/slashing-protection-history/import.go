package history

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/validator/db"
)

// ImportStandardProtectionJSON takes in EIP-3076 compliant JSON file used for slashing protection
// by Ethereum validators and imports its data into Prysm's internal representation of slashing
// protection in the validator client's database. For more information, see the EIP document here:
// https://eips.ethereum.org/EIPS/eip-3076.
func ImportStandardProtectionJSON(ctx context.Context, validatorDB db.Database, r io.Reader) error {
	if validatorDB == nil {
		return errors.New("validatorDB is nil")
	}

	if err := validatorDB.ImportStandardProtectionJSON(ctx, r); err != nil {
		return errors.Wrap(err, "could not import slashing protection JSON file")
	}

	return nil
}
