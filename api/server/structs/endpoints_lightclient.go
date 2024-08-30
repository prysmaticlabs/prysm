package structs

import (
	"encoding/json"
	"fmt"
)

// the interface is used in other structs to reference a light client header object regardless of its version
type lightClientHeader interface {
	isLightClientHeader()
}

type LightClientHeader struct {
	Beacon *BeaconBlockHeader `json:"beacon"`
}

func (a *LightClientHeader) isLightClientHeader() {}

type LightClientHeaderCapella struct {
	Beacon          *BeaconBlockHeader             `json:"beacon"`
	Execution       *ExecutionPayloadHeaderCapella `json:"execution"`
	ExecutionBranch []string                       `json:"execution_branch"`
}

func (a *LightClientHeaderCapella) isLightClientHeader() {}

type LightClientHeaderDeneb struct {
	Beacon          *BeaconBlockHeader           `json:"beacon"`
	Execution       *ExecutionPayloadHeaderDeneb `json:"execution"`
	ExecutionBranch []string                     `json:"execution_branch"`
}

func (a *LightClientHeaderDeneb) isLightClientHeader() {}

type LightClientBootstrap struct {
	Header                     lightClientHeader `json:"header"`
	CurrentSyncCommittee       *SyncCommittee    `json:"current_sync_committee"`
	CurrentSyncCommitteeBranch []string          `json:"current_sync_committee_branch"`
}

type LightClientBootstrapResponse struct {
	Version string                `json:"version"`
	Data    *LightClientBootstrap `json:"data"`
}

func LightClientBootstrapResponseFromJson(data []byte) (*LightClientBootstrapResponse, error) {
	var aux struct {
		Version string
		Data    struct {
			Header                     json.RawMessage
			CurrentSyncCommittee       *SyncCommittee
			CurrentSyncCommitteeBranch []string
		}
	}
	err := json.Unmarshal(data, &aux)
	if err != nil {
		return nil, err
	}

	result := LightClientBootstrapResponse{
		Version: aux.Version,
		Data: &LightClientBootstrap{
			CurrentSyncCommittee:       aux.Data.CurrentSyncCommittee,
			CurrentSyncCommitteeBranch: aux.Data.CurrentSyncCommitteeBranch,
			Header:                     nil,
		},
	}
	switch aux.Version {
	case "altair", "bellatrix":
		var x LightClientHeader
		err = json.Unmarshal(aux.Data.Header, &x)
		if err != nil {
			return nil, err
		}
		result.Data.Header = &x
	case "capella":
		var x LightClientHeaderCapella
		err = json.Unmarshal(aux.Data.Header, &x)
		if err != nil {
			return nil, err
		}
		result.Data.Header = &x
	case "deneb", "electra":
		var x LightClientHeaderDeneb
		err = json.Unmarshal(aux.Data.Header, &x)
		if err != nil {
			return nil, err
		}
		result.Data.Header = &x

	}

	return &result, nil

}

type LightClientUpdate struct {
	AttestedHeader          lightClientHeader `json:"attested_header"`
	NextSyncCommittee       *SyncCommittee    `json:"next_sync_committee,omitempty"`
	FinalizedHeader         lightClientHeader `json:"finalized_header,omitempty"`
	SyncAggregate           *SyncAggregate    `json:"sync_aggregate"`
	NextSyncCommitteeBranch []string          `json:"next_sync_committee_branch,omitempty"`
	FinalityBranch          []string          `json:"finality_branch,omitempty"`
	SignatureSlot           string            `json:"signature_slot"`
}

type LightClientUpdateWithVersion struct {
	Version string             `json:"version"`
	Data    *LightClientUpdate `json:"data"`
}

func LightClientUpdateWithVersionFromJson(data []byte) (*LightClientUpdateWithVersion, error) {
	fmt.Println("versioned data")
	fmt.Println(string(data))

	var aux struct {
		Version string `json:"version"`
		Data    struct {
			AttestedHeader          json.RawMessage `json:"attested_header"`
			NextSyncCommittee       *SyncCommittee  `json:"next_sync_committee,omitempty"`
			FinalizedHeader         json.RawMessage `json:"finalized_header,omitempty"`
			SyncAggregate           *SyncAggregate  `json:"sync_aggregate"`
			NextSyncCommitteeBranch []string        `json:"next_sync_committee_branch,omitempty"`
			FinalityBranch          []string        `json:"finality_branch,omitempty"`
			SignatureSlot           string          `json:"signature_slot"`
		} `json:"data"`
	}
	err := json.Unmarshal(data, &aux)
	if err != nil {
		return nil, err
	}

	result := LightClientUpdateWithVersion{
		Version: aux.Version,
		Data: &LightClientUpdate{
			NextSyncCommittee:       aux.Data.NextSyncCommittee,
			SyncAggregate:           aux.Data.SyncAggregate,
			NextSyncCommitteeBranch: aux.Data.NextSyncCommitteeBranch,
			FinalityBranch:          aux.Data.FinalityBranch,
			SignatureSlot:           aux.Data.SignatureSlot,
			AttestedHeader:          nil,
			FinalizedHeader:         nil,
		},
	}

	switch aux.Version {
	case "altair", "bellatrix":
		var x LightClientHeader
		err = json.Unmarshal(aux.Data.AttestedHeader, &x)
		if err != nil {
			return nil, err
		}
		result.Data.AttestedHeader = &x
		var y LightClientHeader
		err = json.Unmarshal(aux.Data.FinalizedHeader, &y)
		if err != nil {
			return nil, err
		}
		result.Data.FinalizedHeader = &y
	case "capella":
		var x LightClientHeaderCapella
		fmt.Println("attested header data")
		fmt.Println(string(aux.Data.AttestedHeader))
		err = json.Unmarshal(aux.Data.AttestedHeader, &x)
		if err != nil {
			return nil, err
		}
		result.Data.AttestedHeader = &x
		fmt.Println("attested header is set")
		var y LightClientHeaderCapella
		err = json.Unmarshal(aux.Data.FinalizedHeader, &y)
		if err != nil {
			return nil, err
		}
		result.Data.FinalizedHeader = &y
		fmt.Println("finalized header is set")
	case "deneb", "electra":
		var x LightClientHeaderDeneb
		err = json.Unmarshal(aux.Data.AttestedHeader, &x)
		if err != nil {
			return nil, err
		}
		//TODO handle the case where the finalized header is nil
		result.Data.AttestedHeader = &x
		var y LightClientHeaderDeneb
		err = json.Unmarshal(aux.Data.FinalizedHeader, &y)
		if err != nil {
			return nil, err
		}
		result.Data.FinalizedHeader = &y
	}

	return &result, nil
}

type LightClientUpdatesByRangeResponse struct {
	Updates []*LightClientUpdateWithVersion `json:"updates"`
}

func LightClientUpdatesByRangeResponseFromJson(data []byte) (*LightClientUpdatesByRangeResponse, error) {
	fmt.Println("data ")
	fmt.Println(string(data))
	var Updates []json.RawMessage

	err := json.Unmarshal(data, &Updates)
	if err != nil {
		return nil, err
	}
	fmt.Println("updates ")
	fmt.Println(string(Updates[0]))
	fmt.Println(len(Updates))

	result := LightClientUpdatesByRangeResponse{
		Updates: make([]*LightClientUpdateWithVersion, len(Updates)),
	}

	for i, u := range Updates {
		update, err := LightClientUpdateWithVersionFromJson(u)
		if err != nil {
			return nil, err
		}
		result.Updates[i] = update
	}

	return &result, nil
}
