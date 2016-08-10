package gossip

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
)

var (
	NoBootstrapNodes          = errors.New("No Bootstrap Nodes Provided")
	NodeDescriptionIncomplete = errors.New("Node Description is Incomplete")
)

// Sortable form for the gossip nodes, by host and port rather than
// name
type GossipNodesOrdered []*GossipNode

// Make GossipNodesOrdered sortable
func (g GossipNodesOrdered) Len() int {
	return len(g)
}
func (g GossipNodesOrdered) Less(i, j int) bool {
	strI := fmt.Sprintf(g[i].Host + g[i].Port)
	strJ := fmt.Sprintf(g[j].Host + g[j].Port)
	return strI < strJ
}
func (g GossipNodesOrdered) Swap(i, j int) {
	g[i], g[j] = g[j], g[i]
}

// GossipNodes wraps a map of nodes (where the key is host+port)
type GossipNodes map[string]*GossipNode

func (n GossipNodes) ToSortedList() []*GossipNode {
	ordered := make(GossipNodesOrdered, 0, len(n)) // To keep it ordered
	for _, node := range n {
		ordered = append(ordered, node)
	}
	sort.Sort(GossipNodesOrdered(ordered))
	return ordered
}

// Prints out the list of nodes as a table, pretty!
func (n GossipNodes) String() string {

	str := ""

	headerName := "Service Name"
	headerHost := "Host"
	headerPort := "Port"

	maxName := len(headerName)
	maxHost := len(headerHost)
	maxPort := len(headerPort)

	for _, node := range n {
		if len(node.Name) > maxName {
			maxName = len(node.Name)
		}
		if len(node.Host) > maxHost {
			maxHost = len(node.Host)
		}
		if len(node.Port) > maxPort {
			maxPort = len(node.Port)
		}
	}

	maxNameStr := strconv.Itoa(maxName + 2)
	maxHostStr := strconv.Itoa(maxHost + 2)
	maxPortStr := strconv.Itoa(maxPort + 2)

	line := "+" + strings.Repeat("-", maxName+3)
	line += "+" + strings.Repeat("-", maxHost+3)
	line += "+" + strings.Repeat("-", maxPort+3) + "+\n"

	// Start line
	str += line
	header := "| " + fmt.Sprintf("%-"+maxNameStr+"s", headerName)
	header += "| " + fmt.Sprintf("%-"+maxHostStr+"s", headerHost)
	header += "| " + fmt.Sprintf("%-"+maxPortStr+"s", headerPort) + "|\n"
	str += header
	str += line

	for _, node := range n.ToSortedList() {
		row := "| " + fmt.Sprintf("%-"+maxNameStr+"s", node.Name)
		row += "| " + fmt.Sprintf("%-"+maxHostStr+"s", node.Host)
		row += "| " + fmt.Sprintf("%-"+maxPortStr+"s", node.Port) + "|\n"
		str += row
	}

	str += line

	return str
}

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
func bootstrapGossipNodes() (map[string]*GossipNode, error) {
	nodesStr := flag.String("bootstrap-nodes", "", "Comma separated list of nodes")
	return gossipNodesFromFlag(*nodesStr)
}

// This will perform the string operations to convert the input flag into a
// series of nodes.
func gossipNodesFromFlag(str string) (map[string]*GossipNode, error) {

	nodes := make(map[string]*GossipNode, 10)
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

		// Add to the set of gossip nodes
		nodes[host+port] = &GossipNode{
			Host: host,
			Port: port,
		}
	}

	return nodes, nil
}
