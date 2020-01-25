package protoarray

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/emicklei/dot"
	log "github.com/sirupsen/logrus"
)

func TreeHandle(f *ForkChoice) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, _ *http.Request) {
		graph := dot.NewGraph(dot.Directed)
		graph.Attr("rankdir", "RL")
		graph.Attr("labeljust", "l")

		dotNodes := make([]*dot.Node, len(f.store.nodes))
		fmt.Println(len(f.store.nodes))
		for i := len(f.store.nodes) - 1; i >= 0; i-- {
			// Construct label for each node.
			slot := strconv.Itoa(int(f.store.nodes[i].slot))
			weight := strconv.Itoa(int(f.store.nodes[i].weight / 10e9))
			bestDescendent := strconv.Itoa(int(f.store.nodes[i].bestDescendant))
			index := strconv.Itoa(int(i))
			label := "slot: " + slot + "\n index: " + index + "\n bestDescendent: " + bestDescendent + "\n weight: " + weight
			dotN := graph.Node(slot).Box().Attr("label", label)
			dotNodes[i] = &dotN
		}

		for i := len(dotNodes); i > 0; i++ {
			if f.store.nodes[i].parent != nonExistentNode && f.store.nodes[i].parent < uint64(len(dotNodes)) {
				graph.Edge(*dotNodes[i], *dotNodes[f.store.nodes[i].parent])
			}
		}

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(graph.String())); err != nil {
			log.WithError(err).Error("Failed to render p2p info page")
		}
	}
}
