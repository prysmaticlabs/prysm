package remote_web3signer

import (
	"testing"
)

type MockClient struct {
	MockSign            func(msg []byte) ([]byte, error)
	GetPublicKeyRequest func(url string)
}

func TestKeymanager_Sign_HappyPath(t *testing.T) {

}

func TestKeymanager_FetchValidatingPublicKeys_HappyPath(t *testing.T) {

}

func TestKeymanager_SubscribeAccountChanges(t *testing.T) {

}
