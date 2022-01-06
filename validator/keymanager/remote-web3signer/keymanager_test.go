package remote_web3signer

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/stretchr/testify/assert"
)

type MockClient struct {
	Signature  string
	PublicKeys []string
}

func (mc *MockClient) Sign(_ context.Context, _ string, _ *SignRequest) (bls.Signature, error) {
	decoded, err := hex.DecodeString(strings.TrimPrefix(mc.Signature, "0x"))
	if err != nil {
		return nil, err
	}
	return bls.SignatureFromBytes(decoded)
}
func (mc *MockClient) GetPublicKeys(_ context.Context, _ string) ([][48]byte, error) {
	var keys [][48]byte
	for _, pk := range mc.PublicKeys {
		decoded, err := hex.DecodeString(strings.TrimPrefix(pk, "0x"))
		if err != nil {
			return nil, err
		}
		keys = append(keys, bytesutil.ToBytes48(decoded))
	}
	return keys, nil
}

func getMockSignRequest() *validatorpb.SignRequest {
	return &validatorpb.SignRequest{
		Object:          &validatorpb.SignRequest_Block{},
		Fork:            &v1alpha1.Fork{Epoch: 0},
		AggregationSlot: 9999,
	}
}

func TestKeymanager_Sign_HappyPath(t *testing.T) {
	client := &MockClient{
		Signature: "0xb3baa751d0a9132cfe93e4e3d5ff9075111100e3789dca219ade5a24d27e19d16b3353149da1833e9b691bb38634e8dc04469be7032132906c927d7e1a49b414730612877bc6b2810c8f202daf793d1ab0d6b5cb21d52f9e52e883859887a5d9",
	}
	ctx := context.Background()
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	config := &SetupConfig{
		BaseEndpoint:          "example.com",
		GenesisValidatorsRoot: root,
		PublicKeysURL:         "example2.com/api/v1/eth2/publicKeys",
	}
	km, err := NewKeymanager(ctx, config)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	km.client = client

	resp, err := km.Sign(ctx, getMockSignRequest())
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, "0xb3baa751d0a9132cfe93e4e3d5ff9075111100e3789dca219ade5a24d27e19d16b3353149da1833e9b691bb38634e8dc04469be7032132906c927d7e1a49b414730612877bc6b2810c8f202daf793d1ab0d6b5cb21d52f9e52e883859887a5d9", fmt.Sprintf("%#x", resp.Marshal()))
}

func TestKeymanager_FetchValidatingPublicKeys_HappyPath_WithKeyList(t *testing.T) {
	ctx := context.Background()
	decodedKey, err := hexutil.Decode("0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820")
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	keys := [][48]byte{
		bytesutil.ToBytes48(decodedKey),
	}
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	config := &SetupConfig{
		BaseEndpoint:          "example.com",
		GenesisValidatorsRoot: root,
		ProvidedPublicKeys:    keys,
	}
	km, err := NewKeymanager(ctx, config)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	resp, err := km.FetchValidatingPublicKeys(ctx)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp, keys)
}

func TestKeymanager_FetchValidatingPublicKeys_HappyPath_WithExternalURL(t *testing.T) {
	ctx := context.Background()
	client := &MockClient{
		PublicKeys: []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"},
	}
	decodedKey, err := hexutil.Decode("0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820")
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	keys := [][48]byte{
		bytesutil.ToBytes48(decodedKey),
	}
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	config := &SetupConfig{
		BaseEndpoint:          "example.com",
		GenesisValidatorsRoot: root,
		PublicKeysURL:         "example2.com/api/v1/eth2/publicKeys",
	}
	km, err := NewKeymanager(ctx, config)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	km.client = client
	resp, err := km.FetchValidatingPublicKeys(ctx)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp, keys)
}
