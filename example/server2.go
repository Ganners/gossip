package main

import "github.com/ganners/gossip"

func main() {

	logger := gossip.NewStdoutLogger()

	server, err := gossip.NewServer(
		"auth",
		"Handles authentication",
		"0.0.0.0", "8002",
		logger,
	)

	// Add node manually
	server.Nodes = append(server.Nodes, &gossip.GossipNode{
		Host: "0.0.0.0",
		Port: "8001",
	})

	if err != nil {
		logger.Errorf("Could not start server: %s", err.Error())
		return
	}

	// Run a server until it is signalled to stop
	<-server.Start()

	logger.Debugf("Shutting down")
}
