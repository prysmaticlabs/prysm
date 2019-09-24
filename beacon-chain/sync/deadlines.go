package sync

import (
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

const defaultReadDuration = ttfbTimeout
const defaultWriteDuration = 10 * time.Second // RESP_TIMEOUT

func setRPCStreamDeadlines(stream network.Stream) {
	setStreamReadDeadline(stream, defaultReadDuration)
	setStreamWriteDeadline(stream, defaultWriteDuration)
}

func setStreamReadDeadline(stream network.Stream, duration time.Duration) {
	stream.SetReadDeadline(roughtime.Now().Add(duration))
}

func setStreamWriteDeadline(stream network.Stream, duration time.Duration) {
	stream.SetWriteDeadline(roughtime.Now().Add(duration))
}
