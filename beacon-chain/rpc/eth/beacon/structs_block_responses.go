package beacon

import "encoding/json"

// ProduceBlockV3Response is a wrapper json object for the returned block from the ProduceBlockV3 endpoint
type ProduceBlockV3Response struct {
	Version                 string          `json:"version"`
	ExecutionPayloadBlinded bool            `json:"execution_payload_blinded"`
	ExecutionPayloadValue   string          `json:"execution_payload_value"`
	Data                    json.RawMessage `json:"data"` // represents the block values based on the version
}
