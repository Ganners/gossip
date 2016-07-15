package gossip

import (
	"errors"
	"flag"
	"strings"
)

var (
	NoBootstrapNodes          = errors.New("No Bootstrap Nodes Provided")
	NodeDescriptionIncomplete = errors.New("Node Description is Incomplete")
)

// A GossipNode specifies the required information for nodes to establish a
// connection with
type GossipNode struct {
	Host string
	Port string
}

// This will populate the bootstrap (start up) nodes, a node will need to be
// told about at least one connection to start gossiping
func bootstrapGossipNodes() ([]GossipNode, error) {
	nodesStr := flag.String("bootstrap-nodes", "", "Comma separated list of nodes")
	return gossipNodesFromFlag(*nodesStr)
}

// This will perform the string operations to convert the input flag into a
// series of nodes.
func gossipNodesFromFlag(str string) ([]GossipNode, error) {
	nodes := make([]GossipNode, 0, 5)
	nodeStrings := strings.Split(str, ",")

	if len(nodeStrings) == 0 {
		return nodes, NoBootstrapNodes
	}

	for _, nodeStr := range nodeStrings {

		// Some cleanup
		nodeStr = strings.TrimSpace(nodeStr)

		// We don't need to error for this
		if len(nodeStr) == 0 {
			continue
		}

		components := strings.Split(nodeStr, ":")

		if len(components) < 2 {
			return nodes, NodeDescriptionIncomplete
		}

		host := components[0]
		port := components[1]

		nodes = append(nodes, GossipNode{
			Host: host,
			Port: port,
		})
	}

	return nodes, nil
}
