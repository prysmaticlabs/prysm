package remote_web3signer

import (
	"context"
	"github.com/prysmaticlabs/prysm/async/event"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
)

type Web3SignerKeyManager interface {
	keymanager.IKeymanager
}

type Keymanager struct {
	client *client
}

func New(endpoint string) *Keymanager {
	client, err := newClient(endpoint)
	if err != nil {
		// fatal error?
	}
	return &Keymanager{
		client: client,
	}
}

func (k *Keymanager) FetchValidatingPublicKeys(ctx context.Context) ([][48]byte, error) {
	panic("implement me")
}

func (k *Keymanager) Sign(ctx context.Context, request *validatorpb.SignRequest) (bls.Signature, error) {
	// where we do we get this info?
	forkData := &Fork{
		PreviousVersion: "",
		CurrentVersion:  "",
		Epoch:           "",
	}
	forkInfoData := &ForkInfo{
		Fork:                  forkData,
		GenesisValidatorsRoot: "",
	}

	randaoRevealData := &RandaoReveal{Epoch: ""}
	// remember to replace signing root with hex encoding remove 0x
	web3SignerRequest := SignRequest{
		Type:         "foo",
		ForkInfo:     forkInfoData,
		SigningRoot:  string(request.SigningRoot),
		RandaoReveal: randaoRevealData,
	}
	return k.client.Sign(string(request.PublicKey), &web3SignerRequest)
}

func (k *Keymanager) SubscribeAccountChanges(pubKeysChan chan [][48]byte) event.Subscription {
	panic("implement me")
}
