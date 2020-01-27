package blockchain

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/emicklei/dot"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
)

const latestSlotCount = 10

// HeadsHandler is a handler to serve /heads page in metrics.
func (s *Service) HeadsHandler(w http.ResponseWriter, _ *http.Request) {
	buf := new(bytes.Buffer)

	if _, err := fmt.Fprintf(w, "\n %s\t%s\t", "Head slot", "Head root"); err != nil {
		logrus.WithError(err).Error("Failed to render chain heads page")
		return
	}

	if _, err := fmt.Fprintf(w, "\n %s\t%s\t", "---------", "---------"); err != nil {
		logrus.WithError(err).Error("Failed to render chain heads page")
		return
	}

	slots := s.latestHeadSlots()
	for _, slot := range slots {
		r := hex.EncodeToString(bytesutil.Trunc(s.canonicalRoots[uint64(slot)]))
		if _, err := fmt.Fprintf(w, "\n %d\t\t%s\t", slot, r); err != nil {
			logrus.WithError(err).Error("Failed to render chain heads page")
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.WithError(err).Error("Failed to render chain heads page")
	}

}

// TreeHandler is a handler to serve /tree page in metrics.
func (s *Service) TreeHandler(w http.ResponseWriter, _ *http.Request) {
	nodes := s.forkChoiceStore.Nodes()

	graph := dot.NewGraph(dot.Directed)
	graph.Attr("rankdir", "RL")
	graph.Attr("labeljust", "l")

	dotNodes := make([]*dot.Node, len(nodes))
	for i := len(nodes) - 1; i >= 0; i-- {
		// Construct label for each node.
		slot := strconv.Itoa(int(nodes[i].Slot))
		weight := strconv.Itoa(int(nodes[i].Weight / 10e9))
		bestDescendent := strconv.Itoa(int(nodes[i].BestDescendent))
		index := strconv.Itoa(int(i))
		label := "slot: " + slot + "\n index: " + index + "\n bestDescendent: " + bestDescendent + "\n weight: " + weight
		var dotN dot.Node
		if nodes[i].Parent != ^uint64(0) {
			dotN = graph.Node(index).Box().Attr("label", label)
		}
		dotNodes[i] = &dotN
	}

	for i := len(nodes) - 1; i >= 0; i-- {
		if nodes[i].Parent != ^uint64(0) && nodes[i].Parent < uint64(len(dotNodes)) {
			graph.Edge(*dotNodes[i], *dotNodes[nodes[i].Parent])
		}
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(graph.String())); err != nil {
		log.WithError(err).Error("Failed to render p2p info page")
	}
}

// This returns the latest head slots in a slice and up to latestSlotCount
func (s *Service) latestHeadSlots() []int {
	slots := make([]int, 0, len(s.canonicalRoots))
	for k := range s.canonicalRoots {
		slots = append(slots, int(k))
	}
	sort.Ints(slots)
	if (len(slots)) > latestSlotCount {
		return slots[len(slots)-latestSlotCount:]
	}
	return slots
}
