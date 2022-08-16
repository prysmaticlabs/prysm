package accounts

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/validator/client"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"google.golang.org/protobuf/types/known/emptypb"
)

// PerformExitCfg for account voluntary exits.
type PerformExitCfg struct {
	ValidatorClient  ethpb.BeaconNodeValidatorClient
	NodeClient       ethpb.NodeClient
	Keymanager       keymanager.IKeymanager
	RawPubKeys       [][]byte
	FormattedPubKeys []string
}

// ExitPassphrase exported for use in test.
const ExitPassphrase = "Exit my validator"

// Exit performs a voluntary exit on one or more accounts.
func (acm *AccountsCLIManager) Exit(ctx context.Context) error {
	// User decided to cancel the voluntary exit.
	if acm.rawPubKeys == nil && acm.formattedPubKeys == nil {
		return nil
	}

	validatorClient, nodeClient, err := acm.prepareBeaconClients(ctx)
	if err != nil {
		return err
	}
	if nodeClient == nil {
		return errors.New("could not prepare beacon node client")
	}
	syncStatus, err := (*nodeClient).GetSyncStatus(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	if syncStatus == nil {
		return errors.New("could not get sync status")
	}

	if (*syncStatus).Syncing {
		return errors.New("could not perform exit: beacon node is syncing.")
	}

	cfg := PerformExitCfg{
		*validatorClient,
		*nodeClient,
		acm.keymanager,
		acm.rawPubKeys,
		acm.formattedPubKeys,
	}
	rawExitedKeys, trimmedExitedKeys, err := PerformVoluntaryExit(ctx, cfg)
	if err != nil {
		return err
	}
	displayExitInfo(rawExitedKeys, trimmedExitedKeys)

	return nil
}

// PerformVoluntaryExit uses gRPC clients to submit a voluntary exit message to a beacon node.
func PerformVoluntaryExit(
	ctx context.Context, cfg PerformExitCfg,
) (rawExitedKeys [][]byte, formattedExitedKeys []string, err error) {
	var rawNotExitedKeys [][]byte
	for i, key := range cfg.RawPubKeys {
		if err := client.ProposeExit(ctx, cfg.ValidatorClient, cfg.NodeClient, cfg.Keymanager.Sign, key); err != nil {
			rawNotExitedKeys = append(rawNotExitedKeys, key)

			msg := err.Error()
			if strings.Contains(msg, blocks.ValidatorAlreadyExitedMsg) ||
				strings.Contains(msg, blocks.ValidatorCannotExitYetMsg) {
				log.Warningf("Could not perform voluntary exit for account %s: %s", cfg.FormattedPubKeys[i], msg)
			} else {
				log.WithError(err).Errorf("voluntary exit failed for account %s", cfg.FormattedPubKeys[i])
			}
		}
	}

	rawExitedKeys = make([][]byte, 0)
	formattedExitedKeys = make([]string, 0)
	for i, key := range cfg.RawPubKeys {
		found := false
		for _, notExited := range rawNotExitedKeys {
			if bytes.Equal(notExited, key) {
				found = true
				break
			}
		}
		if !found {
			rawExitedKeys = append(rawExitedKeys, key)
			formattedExitedKeys = append(formattedExitedKeys, cfg.FormattedPubKeys[i])
		}
	}

	return rawExitedKeys, formattedExitedKeys, nil
}

func prepareAllKeys(validatingKeys [][fieldparams.BLSPubkeyLength]byte) (raw [][]byte, formatted []string) {
	raw = make([][]byte, len(validatingKeys))
	formatted = make([]string, len(validatingKeys))
	for i, pk := range validatingKeys {
		raw[i] = make([]byte, len(pk))
		copy(raw[i], pk[:])
		formatted[i] = fmt.Sprintf("%#x", bytesutil.Trunc(pk[:]))
	}
	return
}

func displayExitInfo(rawExitedKeys [][]byte, trimmedExitedKeys []string) {
	if len(rawExitedKeys) > 0 {
		urlFormattedPubKeys := make([]string, len(rawExitedKeys))
		for i, key := range rawExitedKeys {
			var baseUrl string
			if params.BeaconConfig().ConfigName == params.PraterName {
				baseUrl = "https://prater.beaconcha.in/validator/"
			} else {
				baseUrl = "https://beaconcha.in/validator/"
			}
			// Remove '0x' prefix
			urlFormattedPubKeys[i] = baseUrl + hexutil.Encode(key)[2:]
		}

		ifaceKeys := make([]interface{}, len(urlFormattedPubKeys))
		for i, k := range urlFormattedPubKeys {
			ifaceKeys[i] = k
		}

		info := fmt.Sprintf("Voluntary exit was successful for the accounts listed. "+
			"URLs where you can track each validator's exit:\n"+strings.Repeat("%s\n", len(ifaceKeys)), ifaceKeys...)

		log.WithField("publicKeys", strings.Join(trimmedExitedKeys, ", ")).Info(info)
	} else {
		log.Info("No successful voluntary exits")
	}
}
