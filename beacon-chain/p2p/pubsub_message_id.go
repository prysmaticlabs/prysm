package p2p

import (
	"encoding/base64"

	"github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// Content addressable ID function.
//
// Loosely defined as Base64(sha2(data))
//
// Pending: https://github.com/ethereum/eth2.0-specs/issues/1528
func msgIdFunction(pmsg *pb.Message) string {
	h := hashutil.FastSum256(pmsg.Data)
	return base64.URLEncoding.EncodeToString(h[:])
}
