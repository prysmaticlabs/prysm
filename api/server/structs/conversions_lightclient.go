package structs

import (
	"encoding/json"
)

func (h *LightClientHeader) ToRawMessage() (json.RawMessage, error) {
	return json.Marshal(h)
}

func (h *LightClientHeaderCapella) ToRawMessage() (json.RawMessage, error) {
	return json.Marshal(h)
}

func (h *LightClientHeaderDeneb) ToRawMessage() (json.RawMessage, error) {
	return json.Marshal(h)
}

func LightClientBootstrapResponseFromJson(data []byte) (*LightClientBootstrapResponse, error) {
	var result LightClientBootstrapResponse
	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func LightClientUpdateWithVersionFromJson(data []byte) (*LightClientUpdateWithVersion, error) {
	var result LightClientUpdateWithVersion
	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func LightClientUpdatesByRangeResponseFromJson(data []byte) (*LightClientUpdatesByRangeResponse, error) {
	var updates []*LightClientUpdateWithVersion
	var result LightClientUpdatesByRangeResponse

	err := json.Unmarshal(data, &updates)
	if err != nil {
		return nil, err
	}

	result.Updates = updates

	return &result, nil
}
