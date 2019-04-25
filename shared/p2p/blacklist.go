package p2p

import  ( 
	ps "github.com/libp2p/go-libp2p-peerstore" 
)


type BlacklistedPeer struct {
	PeerMap map[ps.PeerInfo.ID]bool
}


func AddNewBlacklistedPeer(peerID ps.PeerInfo.ID)  {
	bp := new(BlacklistedPeer)
	bp.PeerMap = make(map[ps.PeerInfo.ID]bool)
	bp.PeerMap[peerID] = true
}


func IsPeerBlacklisted(peerID ps.PeerInfo.ID) bool {
	bps := new(BlacklistedPeer)
	bp.PeerMap = make(map[ps.PeerInfo.ID]bool)
	if v, found := bp.PeerMap[peerID]; found {
		if v {
		 	return true
		}
	} 
	return false
}
	

