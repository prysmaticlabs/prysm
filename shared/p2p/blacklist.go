package p2p

import  ( 
	"sync"
	peer "github.com/libp2p/go-libp2p-peer"
)


type PeerBlackList struct {
	pb map[peer.ID]struct{}
	lk sync.RWMutex
	size int
}

func GetPeerBlackList() *PeerBlackList {
	pbl := new(PeerBlackList)
	pbl.pb = make(map[peer.ID]struct{})
	pbl.size = -1
	return pbl
}


func (pbl *PeerBlackList) Add(p peer.ID)  {
	pbl.lk.RLock()
	pbl.pb[p] = struct{}{}
	pbl.lk.RUnlock()
}

func (pbl *PeerBlackList) Contains(p peer.ID) bool {
	pbl.lk.RLock()
	_, ok := pbl.pb[p]
	pbl.lk.RUnlock()
	return ok
}

func (pbl *PeerBlackList) Size() int {
	pbl.lk.RLock()
	defer pbl.lk.RUnlock()
	return len(pbl.pb)
}

func (pbl *PeerBlackList) BlackListedPeers() []peer.ID {
	pbl.lk.Lock()
	out := make([]peer.ID, 0, len(pbl.pb))
	for p, _ := range ps.pb {
		out = append(out, p)
	}
	pbl.lk.Unlock()
	return out
}



	

