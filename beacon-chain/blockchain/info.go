package blockchain

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/emicklei/dot"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

const latestSlotCount = 10
const treeSize = 64

// For treehandler, each node is a representation of a node in the graph
type node struct {
	parentRoot [32]byte
	dothNode   *dot.Node
}

// TreeHandler is a handler to serve /tree page in metrics.
func (s *Service) TreeHandler(w http.ResponseWriter, r *http.Request) {
	graph := dot.NewGraph(dot.Directed)
	graph.Attr("rankdir", "RL")
	graph.Attr("label", "Canonical block = green")
	graph.Attr("labeljust", "l")

	// Determine block tree range. Current slot to epoch number of slots back.
	currentSlot := s.currentSlot()
	startSlot := uint64(1)
	if currentSlot-treeSize > startSlot {
		startSlot = currentSlot - treeSize
	}

	// Retrieve range blocks for the tree.
	filter := filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(currentSlot)
	blks, err := s.beaconDB.Blocks(context.Background(), filter)

	// Construct tree nodes for visualizations.
	m := make(map[[32]byte]*node)
	for i := 0; i < len(blks); i++ {
		b := blks[i]
		r, err := ssz.SigningRoot(b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Construct label of each node.
		rStr := hex.EncodeToString(r[:2])
		label := "slot: " + strconv.Itoa(int(b.Slot)) + "\n root: " + rStr
		dotN := graph.Node(rStr).Box().Attr("label", label)
		// Set the node box to green if the block is canonical.
		if bytes.Equal(r[:], s.CanonicalRoot(b.Slot)) {
			dotN = dotN.Attr("color", "green")
		}
		n := &node{
			parentRoot: bytesutil.ToBytes32(b.ParentRoot),
			dothNode:   &dotN,
		}
		m[r] = n
	}

	// Construct an edge only if block's parent exist in the tree.
	for _, n := range m {
		if _, ok := m[n.parentRoot]; ok {
			graph.Edge(*n.dothNode, *m[n.parentRoot].dothNode)
		}
	}

	svg, err := dotToSvg([]byte(graph.String()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, svg)
}

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

// This returns the current slot of the chain
func (s *Service) currentSlot() uint64 {
	diff := time.Now().Unix() - s.GenesisTime().Unix()
	return uint64(diff) / params.BeaconConfig().SecondsPerSlot
}

// This converts a raw dot data to svg
func dotToSvg(dot []byte) (string, error) {
	format := "svg"
	dotExe, err := exec.LookPath("dot")
	if err != nil {
		return "", errors.New("unable to find program 'dot', please install it or check your PATH")
	}

	img := filepath.Join(os.TempDir(), fmt.Sprintf("tree-vis.%s", format))

	cmd := exec.Command(dotExe, fmt.Sprintf("-T%s", format), "-o", img)
	cmd.Stdin = bytes.NewReader(dot)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return img, nil
}
