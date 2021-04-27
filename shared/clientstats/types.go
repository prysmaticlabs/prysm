package clientstats

type CommonStats struct {
	CPUProcessSecondsTotal int64 `json:"cpu_process_seconds_total"`
	MemoryProcessBytes int64 `json:"memory_process_bytes"`
	ClientName string `json:"client_name"`
	ClientVersion string `json:"client_version"`
	ClientBuild int64 `json:"client_build"`
	// NA
	SyncEth2FallbackConfigured bool `json:"sync_eth2_fallback_configured"`
	// NA
	SyncEth2FallbackConnected bool `json:"sync_eth2_fallback_connected"`
}

type BeaconNodeStats struct {
	// TBD -- pending merge of slasher refactor
	SlasherActive bool `json:"slasher_active"`
	SyncEth1FallbackConfigured bool `json:"sync_eth1_fallback_configured"`
	SyncEth1FallbackConnected bool `json:"sync_eth1_fallback_connected"`
	SyncEth1Connected bool `json:"sync_eth1_connected"`
	SyncEth2Synced bool `json:"sync_eth2_synced"`
	DiskBeaconchainBytesTotal int64 `json:"disk_beaconchain_bytes_total"`
	// N/A -- would require significant network code changes at this time
	NetworkLibp2pBytesTotalReceive int64 `json:"network_libp2p_bytes_total_receive"`
	// N/A -- would require significant network code changes at this time
	NetworkLibp2pBytesTotalTransmit int64 `json:"network_libp2p_bytes_total_transmit"`
	// p2p_peer_count where label "state" == "Connected"
	NetworkPeersConnected int64 `json:"network_peers_connected"`
	// beacon_head_slot
	SyncBeaconHeadSlot int64 `json:"sync_beacon_head_slot"`
	CommonStats `json:",inline"`
}

type ValidatorStats struct {
	// use validator_count (sum across all labels)
	ValidatorTotal int64 `json:"validator_total"`
	// use validator_count{state="Active"} # 210265 -- just the Active label
	ValidatorActive int64 `json:"validator_active"`
	CommonStats `json:",inline"`
}