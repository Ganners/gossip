package gossip

import (
	"errors"
	"fmt"

	"github.com/ganners/gossip/pb/envelope"
	"github.com/ganners/gossip/pb/subscribe"
	"github.com/gogo/protobuf/proto"
)

var defaultHandlers = []RequestHandler{
	// Generic forwarder of gossip, spread the word!
	{
		RequestMatcher: RequestMatcher{"*"},
		UnmarshalType:  nil,
		HandlerFunc: func(server *Server, request proto.Message) error {
			envelope, ok := request.(*envelope.Envelope)
			if !ok {
				return errors.New("could not convert message to envelope")
			}

			server.Logger.Debugf("Forwarding gossip: %+v", envelope)

			// Increment passthrough and resend
			envelope.PassedThrough++
			b, err := proto.Marshal(envelope)
			if err != nil {
				return fmt.Errorf("Could not marshal proto: %s", err)
			}

			// Spread the word
			server.spreadGossipRaw(b)

			// Else forward to nodes
			return nil
		},
	},
	{
		RequestMatcher: RequestMatcher{"node.subscribe"},
		UnmarshalType:  &subscribe.Subscribe{},
		HandlerFunc: func(server *Server, request proto.Message) error {
			subscription, ok := request.(*subscribe.Subscribe)
			if !ok {
				return errors.New("could not convert message to envelope")
			}

			server.Logger.Debugf("Adding node: %+v", subscription)

			// Add to nodes
			server.Nodes = append(server.Nodes, &GossipNode{
				Name: subscription.Name,
				Host: subscription.Host,
				Port: subscription.Port,
			})

			// Else forward to nodes
			return nil
		},
	},
}
