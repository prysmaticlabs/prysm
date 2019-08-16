package sync

import (
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

func setRPCStreamDeadlines(stream network.Stream) {
	stream.SetReadDeadline(roughtime.Now().Add(5 * time.Second))   // TTFB_TIMEOUT
	stream.SetWriteDeadline(roughtime.Now().Add(10 * time.Second)) // RESP_TIMEOUT
}
