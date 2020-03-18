package keymanager

import (
	"context"
	"encoding/json"
	"fmt"
	RW "github.com/alonmuroch/validatorremotewallet/wallet/v1alpha1"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	grpc "google.golang.org/grpc"
	"time"
)

type RemotewalletOpts struct {
	Url    string   `json:"url"`
}

// TODO
var RemotewalletOptsHelp = `The wallet key manager stores keys in a local encrypted store.  The options are:
  - location This is the location to look for wallets.  If not supplied it will
    use the standard (operating system-dependent) path.
  - accounts This is a list of account specifiers.  An account specifier is of
    the form <wallet name>/[account name],  where the account name can be a
    regular expression.  If the account specifier is just <wallet name> all
    accounts in that wallet will be used.  Multiple account specifiers can be
    supplied if required.
  - passphrase This is the passphrase used to encrypt the accounts when they
    were created.  Multiple passphrases can be supplied if required.

An sample keymanager options file (with annotations; these should be removed if
using this as a template) is:

  {
    "location":    "/wallets",               // Look for wallets in the directory '/wallets'
    "accounts":    ["Validators/Account.*"], // Use all accounts in the 'Validators' wallet starting with 'Account'
    "passphrases": ["secret1","secret2"]     // Use the passphrases 'secret1' and 'secret2' to decrypt accounts
  }`

func NewRemoteWallet(input string) (KeyManager, string, error) {
	// parse config file
	opts := &RemotewalletOpts{}
	err := json.Unmarshal([]byte(input), opts)
	if err != nil {
		return nil, walletOptsHelp, err
	}

	// try and connect to remote wallet
	conn, err := grpc.Dial(opts.Url, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect to remote wallet: %v", err)
	} else {
		log.Printf("connected to remote wallet at: %v", opts.Url)
	}
	//defer conn.Close() TODO
	c := RW.NewRemoteWalletClient(conn)

	km := &RemoteWallet{
		keys: make([][48]byte,0),
		rw: c,
	}

	return km, RemotewalletOptsHelp, nil
}

// Wallet is a key manager that loads keys from a local Ethereum 2 wallet.
type RemoteWallet struct {
	keys [][48]byte
	rw RW.RemoteWalletClient
}

// FetchValidatingKeys fetches the list of public keys that should be used to validate with.
func (km *RemoteWallet) FetchValidatingKeys() ([][48]byte, error) {
	// TODO - how can i dynamically controll keys?

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := km.rw.FetchValidatingKeys(ctx, &RW.FetchValidatingKeysRequest{})
	if err != nil {
		return nil, err
	}
	return km.updateAccounts(r), nil
}

func (km *RemoteWallet) updateAccounts (response *RW.FetchValidatingKeysResponse) ([][48]byte){
	km.keys = make([][48]byte, len(response.PublicKeys))
	for index, key := range response.PublicKeys {
		km.keys[index] = bytesutil.ToBytes48(key)
	}
	return km.keys
}

// Sign signs a message for the validator to broadcast.
func (km *RemoteWallet) Sign(pubKey [48]byte, root [32]byte, domain uint64) (*bls.Signature, error) {
	return nil, fmt.Errorf("This is a protected signer, please use SignProposal or SignAttestation")
}

// SignProposal signs a block proposal for the validator to broadcast.
func (km *RemoteWallet) SignProposal(pubKey [48]byte, domain uint64, data *ethpb.BeaconBlockHeader) (*bls.Signature, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := km.rw.SignProposal(ctx, &RW.SignProposalRequest{PublicKey:pubKey[:], Data:data, Domain:domain})
	if err != nil {
		return nil, err
	}

	return bls.SignatureFromBytes(r.Sig)
}

// SignAttestation signs an attestation for the validator to broadcast.
func (km *RemoteWallet) SignAttestation(pubKey [48]byte, domain uint64, data *ethpb.AttestationData) (*bls.Signature, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := km.rw.SignAttestation(ctx, &RW.SignAttestationRequest{PublicKey:pubKey[:], Data:data, Domain:domain})
	if err != nil {
		return nil, err
	}

	return bls.SignatureFromBytes(r.Sig)
}

// SignSlot signs an aggregate attestation for the validator to broadcast.
func (km *RemoteWallet) SignSlot(pubKey [48]byte, domain uint64, slot [32]byte) (*bls.Signature, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := km.rw.SignSlot(ctx, &RW.SignSlotRequest{PublicKey:pubKey[:], Slot:slot[:], Domain:domain})
	if err != nil {
		return nil, err
	}

	return bls.SignatureFromBytes(r.Sig)
}