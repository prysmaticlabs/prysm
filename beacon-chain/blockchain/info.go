package blockchain

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"

	"github.com/emicklei/dot"
)

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
	if s.headState() == nil {
		if _, err := w.Write([]byte("Unavailable during initial syncing")); err != nil {
			log.WithError(err).Error("Failed to render p2p info page")
		}
	}

	nodes := s.forkChoiceStore.Nodes()

	graph := dot.NewGraph(dot.Directed)
	graph.Attr("rankdir", "RL")
	graph.Attr("labeljust", "l")

	dotNodes := make([]*dot.Node, len(nodes))
	avgBalance := uint64(averageBalance(s.headState().Balances()))

	for i := len(nodes) - 1; i >= 0; i-- {
		// Construct label for each node.
		slot := strconv.Itoa(int(nodes[i].Slot))
		weight := strconv.Itoa(int(nodes[i].Weight / 1e9)) // Convert unit Gwei to unit ETH.
		votes := strconv.Itoa(int(nodes[i].Weight / 1e9 / avgBalance))
		index := strconv.Itoa(i)
		g := nodes[i].Graffiti[:]
		graffiti := hex.EncodeToString(g[:8])
		label := "slot: " + slot + "\n votes: " + votes + "\n weight: " + weight + "\n graffiti: " + graffiti
		var dotN dot.Node
		if nodes[i].Parent != ^uint64(0) {
			dotN = graph.Node(index).Box().Attr("label", label)
		}

		if nodes[i].Slot == s.headSlot() &&
			nodes[i].BestDescendant == ^uint64(0) {
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
