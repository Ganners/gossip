package main

import (
	"time"

	"github.com/ganners/gossip"
	"github.com/ganners/gossip/example/pb/login"
	"github.com/gogo/protobuf/proto"
)

func main() {

	logger := gossip.NewStdoutLogger()

	server, err := gossip.NewServer(
		"user",
		"Handles authentication",
		"0.0.0.0", "8002",
		logger,
	)

	// Add node manually
	server.AddNode(&gossip.GossipNode{
		Host: "0.0.0.0",
		Port: "8001",
	})

	if err != nil {
		logger.Errorf("Could not start server: %s", err.Error())
		return
	}

	// Make a login request every X seconds
	go func() {
		for {
			time.Sleep(time.Second * 5)

			// Make a client request:
			err, response, timeout := server.BroadcastAndWaitForResponse(
				"login",
				&login.LoginRequest{
					Username: "mark",
					Password: "12345",
				})

			if err != nil {
				server.Logger.Errorf("Error with login request: %s", err)
				continue
			}

			select {
			case rsp := <-response:
				loginResponse := &login.LoginResponse{}
				err := proto.Unmarshal(rsp.EncodedMessage, loginResponse)
				if err != nil {
					server.Logger.Errorf("Error unmarshaling response: %s", err)
				}

				// Otherwise lets print the details
				server.Logger.Debugf("Successfully logged in: %+v", *loginResponse)
			case <-timeout:
				server.Logger.Debugf("Timed out waiting for response to login")
			}
		}
	}()

	// Run a server until it is signalled to stop
	<-server.Start()

	logger.Debugf("Shutting down")
}
