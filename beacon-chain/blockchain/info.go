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

	identifiers := s.latestHeadIdentifiers()
	for _, identifier := range identifiers {
		r := hex.EncodeToString(bytesutil.Trunc(identifier[8:]))
		if _, err := fmt.Fprintf(w, "\n %d\t\t%s\t", bytesutil.FromBytes8(identifier[:8]), r); err != nil {
			logrus.WithError(err).Error("Failed to render chain heads page")
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.WithError(err).Error("Failed to render chain heads page")
	}

}

const template = `<html>
<head>
    <script src="//cdnjs.cloudflare.com/ajax/libs/viz.js/2.1.2/viz.js"></script>
    <script src="//cdnjs.cloudflare.com/ajax/libs/viz.js/2.1.2/full.render.js"></script>
<body>
    <script type="application/javascript">
        var graph = ` + "`%s`;" + `
        var viz = new Viz();
        viz.renderSVGElement(graph) // reading the graph.
            .then(function(element) {
                document.body.appendChild(element); // appends to document.
            })
            .catch(error => {
                // Create a new Viz instance (@see Caveats page for more info)
                viz = new Viz();
                // Possibly display the error
                console.error(error);
            });
    </script>
</head>
</body>
</html>`

// TreeHandler is a handler to serve /tree page in metrics.
func (s *Service) TreeHandler(w http.ResponseWriter, _ *http.Request) {
	if s.headState == nil {
		if _, err := w.Write([]byte("Unavailable during initial syncing")); err != nil {
			log.WithError(err).Error("Failed to render p2p info page")
		}
	}

	nodes := s.forkChoiceStore.Nodes()

	graph := dot.NewGraph(dot.Directed)
	graph.Attr("rankdir", "RL")
	graph.Attr("labeljust", "l")

	dotNodes := make([]*dot.Node, len(nodes))
	avgBalance := uint64(averageBalance(s.headState.Balances()))

	for i := len(nodes) - 1; i >= 0; i-- {
		// Construct label for each node.
		slot := strconv.Itoa(int(nodes[i].Slot))
		weight := strconv.Itoa(int(nodes[i].Weight / 1e9)) // Convert unit Gwei to unit ETH.
		votes := strconv.Itoa(int(nodes[i].Weight / 1e9 / avgBalance))
		bestDescendent := strconv.Itoa(int(nodes[i].BestDescendent))
		index := strconv.Itoa(int(i))
		label := "slot: " + slot + "\n index: " + index + "\n bestDescendent: " + bestDescendent + "\n votes: " + votes + "\n weight: " + weight
		var dotN dot.Node
		if nodes[i].Parent != ^uint64(0) {
			dotN = graph.Node(index).Box().Attr("label", label)
		}

		if nodes[i].Slot == s.HeadSlot() &&
			nodes[i].BestDescendent == ^uint64(0) {
			dotN = dotN.Attr("color", "green")
		}

		dotNodes[i] = &dotN
	}

	for i := len(nodes) - 1; i >= 0; i-- {
		if nodes[i].Parent != ^uint64(0) && nodes[i].Parent < uint64(len(dotNodes)) {
			graph.Edge(*dotNodes[i], *dotNodes[nodes[i].Parent])
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html")
	if _, err := fmt.Fprintf(w, template, graph.String()); err != nil {
		log.WithError(err).Error("Failed to render p2p info page")
	}
}

// This returns the latest head identifiers in a slice and up to latestSlotCount
func (s *Service) latestHeadIdentifiers() [][40]byte {
	identifiers := make([][40]byte, 0, len(s.canonicalRoots))
	for k := range s.canonicalRoots {
		identifiers = append(identifiers, k)
	}
	sort.Slice(identifiers, func(i int, j int) bool {
		return bytesutil.FromBytes8(identifiers[i][:8]) < bytesutil.FromBytes8(identifiers[j][:8])
	})
	if (len(identifiers)) > latestSlotCount {
		return identifiers[len(identifiers)-latestSlotCount:]
	}
	return identifiers
}
