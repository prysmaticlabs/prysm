package structs

import (
	"encoding/json"
)

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

func LightClientUpdateWithVersionFromJson(data []byte) (*LightClientUpdateWithVersion, error) {
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
		err = json.Unmarshal(aux.Data.AttestedHeader, &x)
		if err != nil {
			return nil, err
		}
		result.Data.AttestedHeader = &x
		var y LightClientHeaderCapella
		err = json.Unmarshal(aux.Data.FinalizedHeader, &y)
		if err != nil {
			return nil, err
		}
		result.Data.FinalizedHeader = &y
	case "deneb", "electra":
		var x LightClientHeaderDeneb
		err = json.Unmarshal(aux.Data.AttestedHeader, &x)
		if err != nil {
			return nil, err
		}
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

func LightClientUpdatesByRangeResponseFromJson(data []byte) (*LightClientUpdatesByRangeResponse, error) {
	var Updates []json.RawMessage

	err := json.Unmarshal(data, &Updates)
	if err != nil {
		return nil, err
	}

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
