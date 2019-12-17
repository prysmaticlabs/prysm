package p2p

import (
	"encoding/base64"

	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// Content addressable ID function.
//
// Loosely defined as Base64(sha2(data)) until a formal specification is determined.
// Pending: https://github.com/ethereum/eth2.0-specs/issues/1528
func msgIDFunction(pmsg *pubsub_pb.Message) string {
	h := hashutil.FastSum256(pmsg.Data)
	return base64.URLEncoding.EncodeToString(h[:])
}
