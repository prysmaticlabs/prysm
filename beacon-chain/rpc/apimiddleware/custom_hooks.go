package apimiddleware

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/gateway"
)

// https://ethereum.github.io/beacon-apis/#/Beacon/submitPoolAttestations expects posting a top-level array.
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

// https://ethereum.github.io/beacon-apis/#/Validator/getAttesterDuties expects posting a top-level array.
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

// https://ethereum.github.io/beacon-apis/#/Validator/publishAggregateAndProofs expects posting a top-level array.
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

// https://ethereum.github.io/beacon-apis/#/Validator/prepareBeaconCommitteeSubnet expects posting a top-level array.
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

// https://ethereum.github.io/beacon-APIs/#/Beacon/submitPoolSyncCommitteeSignatures expects posting a top-level array.
// We make it more proto-friendly by wrapping it in a struct with a 'data' field.
func wrapSyncCommitteeSignaturesArray(endpoint gateway.Endpoint, _ http.ResponseWriter, req *http.Request) gateway.ErrorJson {
	if _, ok := endpoint.PostRequest.(*submitSyncCommitteeSignaturesRequestJson); ok {
		data := make([]*syncCommitteeMessageJson, 0)
		if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
			return gateway.InternalServerErrorWithMessage(err, "could not decode body")
		}
		j := &submitSyncCommitteeSignaturesRequestJson{Data: data}
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

type tempSyncCommitteesResponseJson struct {
	Data *tempSyncCommitteeValidatorsJson `json:"data"`
}

type tempSyncCommitteeValidatorsJson struct {
	Validators          []string                              `json:"validators"`
	ValidatorAggregates []*tempSyncSubcommitteeValidatorsJson `json:"validator_aggregates"`
}

type tempSyncSubcommitteeValidatorsJson struct {
	Validators []string `json:"validators"`
}

// https://ethereum.github.io/beacon-APIs/?urls.primaryName=v2.0.0#/Beacon/getEpochSyncCommittees returns validator_aggregates as a nested array.
// grpc-gateway returns a struct with nested fields which we have to transform into a plain 2D array.
func prepareValidatorAggregates(body []byte, responseContainer interface{}) (bool, gateway.ErrorJson) {
	tempContainer := &tempSyncCommitteesResponseJson{}
	if err := json.Unmarshal(body, tempContainer); err != nil {
		return false, gateway.InternalServerErrorWithMessage(err, "could not unmarshal response into temp container")
	}
	container, ok := responseContainer.(*syncCommitteesResponseJson)
	if !ok {
		return false, gateway.InternalServerError(errors.New("container is not of the correct type"))
	}

	container.Data = &syncCommitteeValidatorsJson{}
	container.Data.Validators = tempContainer.Data.Validators
	container.Data.ValidatorAggregates = make([][]string, len(tempContainer.Data.ValidatorAggregates))
	for i, srcValAgg := range tempContainer.Data.ValidatorAggregates {
		dstValAgg := make([]string, len(srcValAgg.Validators))
		copy(dstValAgg, tempContainer.Data.ValidatorAggregates[i].Validators)
		container.Data.ValidatorAggregates[i] = dstValAgg
	}

	return true, nil
}
