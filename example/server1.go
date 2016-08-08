package main

import (
	"time"

	"github.com/ganners/gossip"
)

func main() {

	logger := gossip.NewStdoutLogger()

	server, err := gossip.NewServer(
		"auth",
		"Handles authentication",
		"0.0.0.0", "8001",
		logger,
	)

	if err != nil {
		logger.Errorf("Could not start server: %s", err.Error())
		return
	}

	go func() {
		time.Sleep(time.Second * 3)
		server.Terminate()
	}()

	// Run a server until it is signalled to stop
	<-server.Start()

	logger.Debugf("Shutting down")
}
