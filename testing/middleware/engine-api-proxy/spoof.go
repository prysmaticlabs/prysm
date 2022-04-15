package proxy

import "encoding/json"

// Parses the request from thec consensus client and checks if user desires
// to spoof it based on the JSON-RPC method. If so, it returns the modified
// request bytes which will be proxied to the execution client.
//func spoofRequest(config *spoofingConfig, requestBytes []byte) ([]byte, error) {
//	// If the JSON request is not a JSON-RPC object, return the request as-is.
//	jsonRequest, err := unmarshalRPCObject(requestBytes)
//	if err != nil {
//		switch {
//		case strings.Contains(err.Error(), "cannot unmarshal array"):
//			return requestBytes, nil
//		default:
//			return nil, err
//		}
//	}
//	if len(jsonRequest.Params) == 0 {
//		return requestBytes, nil
//	}
//	desiredMethodsToSpoof := make(map[string]*spoof)
//	for _, spoofReq := range config.Requests {
//		desiredMethodsToSpoof[spoofReq.Method] = spoofReq
//	}
//	// If we don't want to spoof the request, just return the request as-is.
//	spoofDetails, ok := desiredMethodsToSpoof[jsonRequest.Method]
//	if !ok {
//		return requestBytes, nil
//	}
//
//	// TODO: Support methods with multiple params.
//	params := make(map[string]interface{})
//	if err := extractObjectFromJSONRPC(jsonRequest.Params[0], &params); err != nil {
//		return nil, err
//	}
//	for fieldToModify, fieldValue := range spoofDetails.Fields {
//		if _, ok := params[fieldToModify]; !ok {
//			continue
//		}
//		params[fieldToModify] = fieldValue
//	}
//	log.WithField("method", jsonRequest.Method).Infof("Modified request %v", params)
//	jsonRequest.Params[0] = params
//	return json.Marshal(jsonRequest)
//}

// Parses the response body from the execution client and checks if user desires
// to spoof it based on the JSON-RPC method. If so, it returns the modified
// response bytes which will be proxied to the consensus client.
//func spoofResponse(config *spoofingConfig, requestBytes []byte, responseBody io.Reader) ([]byte, error) {
//	responseBytes, err := ioutil.ReadAll(responseBody)
//	if err != nil {
//		return nil, err
//	}
//	// If the JSON request is not a JSON-RPC object, return the request as-is.
//	jsonRequest, err := unmarshalRPCObject(requestBytes)
//	if err != nil {
//		switch {
//		case strings.Contains(err.Error(), "cannot unmarshal array"):
//			return responseBytes, nil
//		default:
//			return nil, err
//		}
//	}
//	jsonResponse, err := unmarshalRPCObject(responseBytes)
//	if err != nil {
//		switch {
//		case strings.Contains(err.Error(), "cannot unmarshal array"):
//			return responseBytes, nil
//		default:
//			return nil, err
//		}
//	}
//	desiredMethodsToSpoof := make(map[string]*spoof)
//	for _, spoofReq := range config.Responses {
//		desiredMethodsToSpoof[spoofReq.Method] = spoofReq
//	}
//	// If we don't want to spoof the request, just return the request as-is.
//	spoofDetails, ok := desiredMethodsToSpoof[jsonRequest.Method]
//	if !ok {
//		return responseBytes, nil
//	}
//
//	// TODO: Support nested objects.
//	params := make(map[string]interface{})
//	if err := extractObjectFromJSONRPC(jsonResponse.Result, &params); err != nil {
//		return nil, err
//	}
//	for fieldToModify, fieldValue := range spoofDetails.Fields {
//		if _, ok := params[fieldToModify]; !ok {
//			continue
//		}
//		params[fieldToModify] = fieldValue
//	}
//	log.WithField("method", jsonRequest.Method).Infof("Modified response %v", params)
//	jsonResponse.Result = params
//	return json.Marshal(jsonResponse)
//}
//

func unmarshalRPCObject(rawBytes []byte) (*jsonRPCObject, error) {
	jsonObj := &jsonRPCObject{}
	if err := json.Unmarshal(rawBytes, jsonObj); err != nil {
		return nil, err
	}
	return jsonObj, nil
}

func extractObjectFromJSONRPC(src interface{}, dst interface{}) error {
	rawResp, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(rawResp, dst)
}
