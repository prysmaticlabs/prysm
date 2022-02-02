package blockchain

import (
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/emicklei/dot"
	"github.com/prysmaticlabs/prysm/config/params"
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
func (s *Service) TreeHandler(w http.ResponseWriter, r *http.Request) {
	headState, err := s.HeadState(r.Context())
	if err != nil {
		log.WithError(err).Error("Could not get head state")
		return
	}
	if headState == nil || headState.IsNil() {
		if _, err := w.Write([]byte("Unavailable during initial syncing")); err != nil {
			log.WithError(err).Error("Failed to render p2p info page")
		}
	}

	nodes := s.cfg.ForkChoiceStore.Nodes()

	graph := dot.NewGraph(dot.Directed)
	graph.Attr("rankdir", "RL")
	graph.Attr("labeljust", "l")

	dotNodes := make([]*dot.Node, len(nodes))
	avgBalance := uint64(averageBalance(headState.Balances()))

	for i := len(nodes) - 1; i >= 0; i-- {
		// Construct label for each node.
		slot := fmt.Sprintf("%d", nodes[i].Slot())
		weight := fmt.Sprintf("%d", nodes[i].Weight()/1e9) // Convert unit Gwei to unit ETH.
		votes := fmt.Sprintf("%d", nodes[i].Weight()/1e9/avgBalance)
		index := fmt.Sprintf("%d", i)
		g := nodes[i].Graffiti()
		graffiti := hex.EncodeToString(g[:8])
		label := "slot: " + slot + "\n votes: " + votes + "\n weight: " + weight + "\n graffiti: " + graffiti
		var dotN dot.Node
		if nodes[i].Parent() != ^uint64(0) {
			dotN = graph.Node(index).Box().Attr("label", label)
		}

		if nodes[i].Slot() == s.HeadSlot() &&
			nodes[i].BestDescendant() == ^uint64(0) &&
			nodes[i].Parent() != ^uint64(0) {
			dotN = dotN.Attr("color", "green")
		}

		dotNodes[i] = &dotN
	}

	for i := len(nodes) - 1; i >= 0; i-- {
		if nodes[i].Parent() != ^uint64(0) && nodes[i].Parent() < uint64(len(dotNodes)) {
			graph.Edge(*dotNodes[i], *dotNodes[nodes[i].Parent()])
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html")
	if _, err := fmt.Fprintf(w, template, graph.String()); err != nil {
		log.WithError(err).Error("Failed to render p2p info page")
	}
}

func averageBalance(balances []uint64) float64 {
	total := uint64(0)
	for i := 0; i < len(balances); i++ {
		total += balances[i]
	}
	return float64(total) / float64(len(balances)) / float64(params.BeaconConfig().GweiPerEth)
}
