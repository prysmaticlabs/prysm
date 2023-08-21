package node

type SyncStatusResponse struct {
	Data *SyncStatusResponseData `json:"data"`
}

type SyncStatusResponseData struct {
	HeadSlot     string `json:"head_slot"`
	SyncDistance string `json:"sync_distance"`
	IsSyncing    bool   `json:"is_syncing"`
	IsOptimistic bool   `json:"is_optimistic"`
	ElOffline    bool   `json:"el_offline"`
}
