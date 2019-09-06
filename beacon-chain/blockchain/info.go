package blockchain

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
)

const latestSlotCount = 10

// HeadsHandler is a handler to serve /heads page in metrics.
func (c *ChainService) HeadsHandler(w http.ResponseWriter, _ *http.Request) {
	buf := new(bytes.Buffer)

	if _, err := fmt.Fprintf(w, "\n %s\t%s\t", "Head slot", "Head root"); err != nil {
		logrus.WithError(err).Error("Failed to render chain heads page")
		return
	}

	if _, err := fmt.Fprintf(w, "\n %s\t%s\t", "---------", "---------"); err != nil {
		logrus.WithError(err).Error("Failed to render chain heads page")
		return
	}

	slots := c.latestHeadSlots()
	for _, s := range slots {
		r := hex.EncodeToString(bytesutil.Trunc(c.canonicalRoots[uint64(s)]))
		if _, err := fmt.Fprintf(w, "\n %d\t\t%s\t", s, r); err != nil {
			logrus.WithError(err).Error("Failed to render chain heads page")
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.WithError(err).Error("Failed to render chain heads page")
	}

}

// This returns the latest head slots in a slice and up to latestSlotCount
func (c *ChainService) latestHeadSlots() []int {
	s := make([]int, 0, len(c.canonicalRoots))
	for k := range c.canonicalRoots {
		s = append(s, int(k))
	}
	sort.Ints(s)
	if (len(s)) > latestSlotCount {
		return s[len(s)-latestSlotCount:]
	}
	return s
}
