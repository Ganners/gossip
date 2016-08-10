package main

import (
	"fmt"

	"github.com/ganners/gossip"
	"github.com/ganners/gossip/example/pb/login"
	"github.com/ganners/gossip/pb/envelope"
	"github.com/gogo/protobuf/proto"
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

	server.Handle("login", func(server *gossip.Server, request envelope.Envelope) error {
		req := &login.LoginRequest{}
		err := proto.Unmarshal(request.EncodedMessage, req)
		if err != nil {
			return fmt.Errorf("Could not unmarshal login request: %s", err)
		}

		// Some logging
		server.Logger.Debugf("Login called for %s with password %s", req.Username, req.Password)

		// Assume it's always successful. If there was a receipt in the
		// envelope then lets return some information to them
		headers := request.GetHeaders()
		if headers != nil && len(headers.Receipt) > 0 {

			server.Logger.Debugf("Receipt found, responding to login request")

			// Broadcast our response
			server.Broadcast(
				request.Headers.Receipt,
				&login.LoginResponse{
					Id:           "12345678",
					SessionToken: "ABCD-1234-1234",
					RefreshToken: "ABCD-1234-1234",
				},
				// Broadcast our response
				int32(envelope.Envelope_RESPONSE),
			)
		}

		return nil
	})

	// Run a server until it is signalled to stop
	<-server.Start()

	logger.Debugf("Shutting down")
}
