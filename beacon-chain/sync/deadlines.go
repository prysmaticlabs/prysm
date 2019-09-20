package sync

import (
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

const defaultReadDuration time.Duration = 5
const defaultWriteDuration time.Duration = 10

func setRPCStreamDeadlines(stream network.Stream) {
	setStreamReadDeadline(stream, defaultReadDuration)
	setStreamWriteDeadline(stream, defaultWriteDuration)
}

func setStreamReadDeadline(stream network.Stream, duration time.Duration) {
	stream.SetReadDeadline(roughtime.Now().Add(duration * time.Second)) // TTFB_TIMEOUT
}

func setStreamWriteDeadline(stream network.Stream, duration time.Duration) {
	stream.SetWriteDeadline(roughtime.Now().Add(duration * time.Second)) // RESP_TIMEOUT
}
