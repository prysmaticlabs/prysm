package apimiddleware

import (
	"encoding/json"
	"fmt"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestListKeystores_JSONisEqual(t *testing.T) {
	middlewareResponse := &listKeystoresResponseJson{
		Keystores: []*keystoreJson{
			{
				ValidatingPubkey: "0x0",
				DerivationPath:   "m/44'/60'/0'/0/0",
			},
		},
	}

	protoResponse := &service.ListKeystoresResponse{
		Data: []*service.ListKeystoresResponse_Keystore{
			{
				ValidatingPubkey: make([]byte, fieldparams.BLSPubkeyLength),
				DerivationPath:   "m/44'/60'/0'/0/0",
			},
		},
	}

	listResp, err := areJsonPropertyNamesEqual(middlewareResponse, protoResponse)
	require.NoError(t, err)
	require.Equal(t, listResp, true)

	resp, err := areJsonPropertyNamesEqual(middlewareResponse.Keystores[0], protoResponse.Data[0])
	require.NoError(t, err)
	require.Equal(t, resp, true)
}

func TestImportKeystores_JSONisEqual(t *testing.T) {
	importKeystoresRequest := &importKeystoresRequestJson{}

	protoImportRequest := &service.ImportKeystoresRequest{
		Keystores:          []string{""},
		Passwords:          []string{""},
		SlashingProtection: "a",
	}

	requestResp, err := areJsonPropertyNamesEqual(importKeystoresRequest, protoImportRequest)
	require.NoError(t, err)
	require.Equal(t, requestResp, true)

	importKeystoresResponse := &importKeystoresResponseJson{
		Statuses: []*statusJson{
			{
				Status:  "Error",
				Message: "a",
			},
		},
	}

	protoImportKeystoresResponse := &service.ImportKeystoresResponse{
		Data: []*service.ImportedKeystoreStatus{
			{
				Status:  service.ImportedKeystoreStatus_ERROR,
				Message: "a",
			},
		},
	}

	ImportResp, err := areJsonPropertyNamesEqual(importKeystoresResponse, protoImportKeystoresResponse)
	require.NoError(t, err)
	require.Equal(t, ImportResp, true)

	resp, err := areJsonPropertyNamesEqual(importKeystoresResponse.Statuses[0], protoImportKeystoresResponse.Data[0])
	require.NoError(t, err)
	require.Equal(t, resp, true)
}

func TestDeleteKeystores_JSONisEqual(t *testing.T) {
	deleteKeystoresRequest := &deleteKeystoresRequestJson{}

	protoDeleteRequest := &service.DeleteKeystoresRequest{
		Pubkeys: [][]byte{{}},
	}

	requestResp, err := areJsonPropertyNamesEqual(deleteKeystoresRequest, protoDeleteRequest)
	require.NoError(t, err)
	require.Equal(t, requestResp, true)

	deleteKeystoresResponse := &deleteKeystoresResponseJson{
		Statuses: []*statusJson{
			{
				Status:  "Error",
				Message: "a",
			},
		},
		SlashingProtection: "a",
	}
	protoDeleteResponse := &service.DeleteKeystoresResponse{
		Data: []*service.DeletedKeystoreStatus{
			{
				Status:  service.DeletedKeystoreStatus_ERROR,
				Message: "a",
			},
		},
		SlashingProtection: "a",
	}

	deleteResp, err := areJsonPropertyNamesEqual(deleteKeystoresResponse, protoDeleteResponse)
	require.NoError(t, err)
	require.Equal(t, deleteResp, true)

	resp, err := areJsonPropertyNamesEqual(deleteKeystoresResponse.Statuses[0], protoDeleteResponse.Data[0])
	require.NoError(t, err)
	require.Equal(t, resp, true)

}

func TestListRemoteKeys_JSONisEqual(t *testing.T) {
	middlewareResponse := &listRemoteKeysResponseJson{
		Keystores: []*remoteKeysListJson{
			{
				Pubkey:   "0x0",
				Url:      "http://localhost:8080",
				Readonly: true,
			},
		},
	}

	protoResponse := &service.ListRemoteKeysResponse{
		Data: []*service.ListRemoteKeysResponse_Keystore{
			{
				Pubkey:   make([]byte, fieldparams.BLSPubkeyLength),
				Url:      "http://localhost:8080",
				Readonly: true,
			},
		},
	}

	listResp, err := areJsonPropertyNamesEqual(middlewareResponse, protoResponse)
	require.NoError(t, err)
	require.Equal(t, listResp, true)

	resp, err := areJsonPropertyNamesEqual(middlewareResponse.Keystores[0], protoResponse.Data[0])
	require.NoError(t, err)
	require.Equal(t, resp, true)
}

func TestImportRemoteKeys_JSONisEqual(t *testing.T) {
	importKeystoresRequest := &importRemoteKeysRequestJson{}

	protoImportRequest := &service.ImportRemoteKeysRequest{
		RemoteKeys: []*service.ImportRemoteKeysRequest_Keystore{
			{
				Pubkey: make([]byte, fieldparams.BLSPubkeyLength),
				Url:    "http://localhost:8080",
			},
		},
	}

	requestResp, err := areJsonPropertyNamesEqual(importKeystoresRequest, protoImportRequest)
	require.NoError(t, err)
	require.Equal(t, requestResp, true)

	importKeystoresResponse := &importRemoteKeysResponseJson{
		Statuses: []*statusJson{
			{
				Status:  "Error",
				Message: "a",
			},
		},
	}

	protoImportKeystoresResponse := &service.ImportRemoteKeysResponse{
		Data: []*service.ImportedRemoteKeysStatus{
			{
				Status:  service.ImportedRemoteKeysStatus_ERROR,
				Message: "a",
			},
		},
	}

	ImportResp, err := areJsonPropertyNamesEqual(importKeystoresResponse, protoImportKeystoresResponse)
	require.NoError(t, err)
	require.Equal(t, ImportResp, true)

	resp, err := areJsonPropertyNamesEqual(importKeystoresResponse.Statuses[0], protoImportKeystoresResponse.Data[0])
	require.NoError(t, err)
	require.Equal(t, resp, true)
}

func TestDeleteRemoteKeys_JSONisEqual(t *testing.T) {
	deleteKeystoresRequest := &deleteRemoteKeysRequestJson{}

	protoDeleteRequest := &service.DeleteRemoteKeysRequest{
		Pubkeys: [][]byte{{}},
	}

	requestResp, err := areJsonPropertyNamesEqual(deleteKeystoresRequest, protoDeleteRequest)
	require.NoError(t, err)
	require.Equal(t, requestResp, true)

	deleteKeystoresResponse := &deleteRemoteKeysResponseJson{
		Statuses: []*statusJson{
			{
				Status:  "Error",
				Message: "a",
			},
		},
	}
	protoDeleteResponse := &service.DeleteRemoteKeysResponse{
		Data: []*service.DeletedRemoteKeysStatus{
			{
				Status:  service.DeletedRemoteKeysStatus_ERROR,
				Message: "a",
			},
		},
	}

	deleteResp, err := areJsonPropertyNamesEqual(deleteKeystoresResponse, protoDeleteResponse)
	require.NoError(t, err)
	require.Equal(t, deleteResp, true)

	resp, err := areJsonPropertyNamesEqual(deleteKeystoresResponse.Statuses[0], protoDeleteResponse.Data[0])
	require.NoError(t, err)
	require.Equal(t, resp, true)
}

// note: this does not do a deep comparison of the structs
func areJsonPropertyNamesEqual(internal, proto interface{}) (bool, error) {
	internalJSON, err := json.Marshal(internal)
	if err != nil {
		return false, err
	}
	protoJSON, err := json.Marshal(proto)
	if err != nil {
		return false, err
	}

	var internalRaw map[string]json.RawMessage
	err = json.Unmarshal(internalJSON, &internalRaw)
	if err != nil {
		return false, err
	}

	var protoRaw map[string]json.RawMessage
	err = json.Unmarshal(protoJSON, &protoRaw)
	if err != nil {
		return false, err
	}

	internalKeys := make([]string, 0, len(internalRaw))
	protoKeys := make([]string, 0, len(protoRaw))
	for key := range internalRaw {
		internalKeys = append(internalKeys, key)
		if _, ok := protoRaw[key]; !ok {
			fmt.Printf("key: %s not found\n", key)
			fmt.Printf("proto: %v\n", protoRaw)
			return false, nil
		} else {
			protoKeys = append(protoKeys, key)
			fmt.Printf("key: %s\n", key)
		}
	}
	if len(internalKeys) != len(protoKeys) {
		return false, nil
	}
	return true, nil
}
