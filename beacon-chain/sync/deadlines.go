package sync

import (
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var defaultReadDuration = ttfbTimeout
var defaultWriteDuration = params.BeaconNetworkConfig().RespTimeout // RESP_TIMEOUT

func setRPCStreamDeadlines(stream network.Stream) {
	setStreamReadDeadline(stream, defaultReadDuration)
	setStreamWriteDeadline(stream, defaultWriteDuration)
}

func setStreamReadDeadline(stream network.Stream, duration time.Duration) {
	// libp2p uses the system clock time for determining the deadline so we use
	// time.Now() instead of the synchronized roughtime.Now().
	if err := stream.SetReadDeadline(time.Now().Add(duration)); err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"peer":      stream.Conn().RemotePeer(),
			"protocol":  stream.Protocol(),
			"direction": stream.Stat().Direction,
		}).Debug("Failed to set stream deadline")
	}
}

func setStreamWriteDeadline(stream network.Stream, duration time.Duration) {
	// libp2p uses the system clock time for determining the deadline so we use
	// time.Now() instead of the synchronized roughtime.Now().
	if err := stream.SetWriteDeadline(time.Now().Add(duration)); err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"peer":      stream.Conn().RemotePeer(),
			"protocol":  stream.Protocol(),
			"direction": stream.Stat().Direction,
		}).Debug("Failed to set stream deadline")
	}
}
