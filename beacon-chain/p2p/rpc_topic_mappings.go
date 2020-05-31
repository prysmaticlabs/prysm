package p2p

const (
	// RPCStatusTopic defines the topic for the status rpc method.
	RPCStatusTopic = "/eth2/beacon_chain/req/status/1"
	// RPCGoodByeTopic defines the topic for the goodbye rpc method.
	RPCGoodByeTopic = "/eth2/beacon_chain/req/goodbye/1"
	// RPCBlocksByRangeTopic defines the topic for the blocks by range rpc method.
	RPCBlocksByRangeTopic = "/eth2/beacon_chain/req/beacon_blocks_by_range/1"
	// RPCBlocksByRootTopic defines the topic for the blocks by root rpc method.
	RPCBlocksByRootTopic = "/eth2/beacon_chain/req/beacon_blocks_by_root/1"
	// RPCPingTopic defines the topic for the ping rpc method.
	RPCPingTopic = "/eth2/beacon_chain/req/ping/1"
	// RPCMetaDataTopic defines the topic for the metadata rpc method.
	RPCMetaDataTopic = "/eth2/beacon_chain/req/metadata/1"
)
