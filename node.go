package gossip

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"strings"
)

var (
	NoBootstrapNodes          = errors.New("No Bootstrap Nodes Provided")
	NodeDescriptionIncomplete = errors.New("Node Description is Incomplete")
)

// A GossipNode specifies the required information for nodes to establish a
// connection with
type GossipNode struct {
	Name string
	Conn net.Conn
	Host string
	Port string
}

// Sends a message to a node, can be used to send a request or forward
// gossip
func (n *GossipNode) SendMessage(b []byte) error {
	if n.Conn == nil {
		return errors.New("gossip node has not established a connection")
	}

	_, err := n.Conn.Write(b)
	return err
}

// @TODO(mark): This will start a TCP connection to each node, it isn't
//              particularly scalable but simplifies the initial implementation
func (n *GossipNode) Connect() error {
	// If it's already connected then return nil
	if n.Conn != nil {
		return nil
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", n.Host, n.Port))
	if err != nil {
		return fmt.Errorf("Unable to dial up to bootstrap node: %s", err)
	}

	// Set the connection
	n.Conn = conn

	return nil
}

// This will populate the bootstrap (start up) nodes, a node will need to be
// told about at least one connection to start gossiping
func bootstrapGossipNodes() ([]*GossipNode, error) {
	nodesStr := flag.String("bootstrap-nodes", "", "Comma separated list of nodes")
	return gossipNodesFromFlag(*nodesStr)
}

// This will perform the string operations to convert the input flag into a
// series of nodes.
func gossipNodesFromFlag(str string) ([]*GossipNode, error) {
	nodes := make([]*GossipNode, 0, 5)
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

		nodes = append(nodes, &GossipNode{
			Host: host,
			Port: port,
		})
	}

	return nodes, nil
}
