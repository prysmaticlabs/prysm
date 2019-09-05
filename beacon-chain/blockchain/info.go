package blockchain

import (
	"bytes"
	"net/http"
	"sort"
)

const latestSlotCount = 10

// InfoHandler is a handler to serve /p2p page in metrics.
func (c *ChainService) InfoHandler(w http.ResponseWriter, _ *http.Request) {
	buf := new(bytes.Buffer)

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.WithError(err).Error("Failed to render p2p info page")
	}
}

// This returns the latest head slots in a slice and up to latestSlotCount
func (c *ChainService) latestHeadSlots() []int {
	s := make([]int, 0, len(c.canonicalRoots))
	for k := range c.canonicalRoots {
		s = append(s, int(k))
	}
	sort.Ints(s)
	return s[latestSlotCount:]
}
