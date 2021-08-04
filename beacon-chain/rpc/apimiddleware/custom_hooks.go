package apimiddleware

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/gateway"
)

// https://ethereum.github.io/eth2.0-APIs/#/Beacon/submitPoolAttestations expects posting a top-level array.
// We make it more proto-friendly by wrapping it in a struct with a 'data' field.
func wrapAttestationsArray(endpoint gateway.Endpoint, _ http.ResponseWriter, req *http.Request) gateway.ErrorJson {
	if _, ok := endpoint.PostRequest.(*submitAttestationRequestJson); ok {
		atts := make([]*attestationJson, 0)
		if err := json.NewDecoder(req.Body).Decode(&atts); err != nil {
			return gateway.InternalServerErrorWithMessage(err, "could not decode body")
		}
		j := &submitAttestationRequestJson{Data: atts}
		b, err := json.Marshal(j)
		if err != nil {
			return gateway.InternalServerErrorWithMessage(err, "could not marshal wrapped body")
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(b))
	}
	return nil
}

// https://ethereum.github.io/eth2.0-APIs/#/Validator/getAttesterDuties expects posting a top-level array.
// We make it more proto-friendly by wrapping it in a struct with an 'index' field.
func wrapValidatorIndicesArray(endpoint gateway.Endpoint, _ http.ResponseWriter, req *http.Request) gateway.ErrorJson {
	if _, ok := endpoint.PostRequest.(*attesterDutiesRequestJson); ok {
		indices := make([]string, 0)
		if err := json.NewDecoder(req.Body).Decode(&indices); err != nil {
			return gateway.InternalServerErrorWithMessage(err, "could not decode body")
		}
		j := &attesterDutiesRequestJson{Index: indices}
		b, err := json.Marshal(j)
		if err != nil {
			return gateway.InternalServerErrorWithMessage(err, "could not marshal wrapped body")
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(b))
	}
	return nil
}

// https://ethereum.github.io/eth2.0-APIs/#/Validator/publishAggregateAndProofs expects posting a top-level array.
// We make it more proto-friendly by wrapping it in a struct with a 'data' field.
func wrapSignedAggregateAndProofArray(endpoint gateway.Endpoint, _ http.ResponseWriter, req *http.Request) gateway.ErrorJson {
	if _, ok := endpoint.PostRequest.(*submitAggregateAndProofsRequestJson); ok {
		data := make([]*signedAggregateAttestationAndProofJson, 0)
		if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
			return gateway.InternalServerErrorWithMessage(err, "could not decode body")
		}
		j := &submitAggregateAndProofsRequestJson{Data: data}
		b, err := json.Marshal(j)
		if err != nil {
			return gateway.InternalServerErrorWithMessage(err, "could not marshal wrapped body")
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(b))
	}
	return nil
}

// https://ethereum.github.io/eth2.0-APIs/#/Validator/prepareBeaconCommitteeSubnet expects posting a top-level array.
// We make it more proto-friendly by wrapping it in a struct with a 'data' field.
func wrapBeaconCommitteeSubscriptionsArray(endpoint gateway.Endpoint, _ http.ResponseWriter, req *http.Request) gateway.ErrorJson {
	if _, ok := endpoint.PostRequest.(*submitBeaconCommitteeSubscriptionsRequestJson); ok {
		data := make([]*beaconCommitteeSubscribeJson, 0)
		if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
			return gateway.InternalServerErrorWithMessage(err, "could not decode body")
		}
		j := &submitBeaconCommitteeSubscriptionsRequestJson{Data: data}
		b, err := json.Marshal(j)
		if err != nil {
			return gateway.InternalServerErrorWithMessage(err, "could not marshal wrapped body")
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(b))
	}
	return nil
}

// Posted graffiti needs to have length of 32 bytes, but client is allowed to send data of any length.
func prepareGraffiti(endpoint gateway.Endpoint, _ http.ResponseWriter, _ *http.Request) gateway.ErrorJson {
	if block, ok := endpoint.PostRequest.(*beaconBlockContainerJson); ok {
		b := bytesutil.ToBytes32([]byte(block.Message.Body.Graffiti))
		block.Message.Body.Graffiti = hexutil.Encode(b[:])
	}
	return nil
}
