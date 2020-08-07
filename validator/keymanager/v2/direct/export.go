package direct

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/bls"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
)

// ExportKeystores --
func (dr *Keymanager) ExportKeystores(
	ctx context.Context, publicKeys []bls.PublicKey,
) ([]*v2keymanager.Keystore, error) {
	for _, pk := range publicKeys {
		fmt.Printf("Done exporting %#x\n", pk.Marshal())
	}
	return nil, nil
}
