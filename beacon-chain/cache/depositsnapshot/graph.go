package depositsnapshot

import (
	"bufio"
	"fmt"
	"os"
)

func printTree(tree MerkleTreeNode) error {
	nodes, _ := printNode(tree, 0)
	file, err := os.Create("/home/sammy/prysmaticLabs/prysm/beacon-chain/cache/depositsnapshot/graph.gv")
	if err != nil {
		return err
	}
	defer file.Close()
	dataWriter := bufio.NewWriter(file)
	_, err = dataWriter.WriteString("digraph G {")
	if err != nil {
		return err
	}
	for _, node := range nodes {
		_, err = dataWriter.WriteString("\n" + node)
		if err != nil {
			return err
		}
	}
	_, err = dataWriter.WriteString("}\n")
	if err != nil {
		return err
	}
	dataWriter.Flush()
	return nil
}

func printNode(node MerkleTreeNode, count int) ([]string, int) {
	var out []string
	switch node.(type) {
	case *FinalizedNode:
		out = []string{fmt.Sprintf("N%d [shape=circle, color=violet, style=bold, label=F%d];", count, count)}
	case *LeafNode:
		out = []string{fmt.Sprintf("N%d [shape=circle, color=cyan, style=dashed, label=L%d];", count, count)}
	case *ZeroNode:
		out = []string{fmt.Sprintf("N%d [shape=circle, color=black, style=dashed, label=Z%d];", count, count)}
	case *InnerNode:
		out = []string{fmt.Sprintf("N%d [shape=circle, color=cyan, style=bold, label=N%d];", count, count)}
		parent := count
		left, nextcount := printNode(node.Left(), count+1)
		out = append(out, left...)
		out = append(out, fmt.Sprintf("N%d -> N%d", parent, parent+1))
		var right []string
		right, count = printNode(node.Right(), nextcount)
		out = append(out, right...)
		out = append(out, fmt.Sprintf("N%d -> N%d", parent, nextcount))
		count -= 1
	default:
		out = []string{fmt.Sprintf("N%d [shape=box, color=red, style=solid, label=\"?%d-%T\"];", count, count, node)}
	}
	return out, count + 1
}
