package gossip

import (
	"fmt"

	"github.com/ganners/gossip/pb/envelope"
	"github.com/ganners/gossip/pb/subscribe"
	"github.com/gogo/protobuf/proto"
)

var defaultHandlers = []RequestHandler{
	{
		RequestMatcher: RequestMatcher{"node.subscribe"},
		HandlerFunc: func(server *Server, request envelope.Envelope) error {
			// Unmarshal and forward on
			subscription := &subscribe.Subscribe{}
			err := proto.Unmarshal(request.EncodedMessage, subscription)
			if err != nil {
				return fmt.Errorf("could not convert message to subscribe.Subscribe: %s", err)
			}

			// Add to nodes
			server.nodes[subscription.Host+subscription.Port] = &GossipNode{
				Name: subscription.Name,
				Host: subscription.Host,
				Port: subscription.Port,
			}

			// Else forward to nodes
			return nil
		},
	},
	// Generic forwarder of gossip, spread the word!
	{
		RequestMatcher: RequestMatcher{"*"},
		HandlerFunc: func(server *Server, request envelope.Envelope) error {

			envelope := request
			server.Logger.Debugf("Forwarding gossip: %+v", envelope)

			// Increment passthrough and resend
			envelope.PassedThrough++
			b, err := proto.Marshal(&envelope)
			if err != nil {
				return fmt.Errorf("Could not marshal proto: %s", err)
			}

			// Spread the word
			server.spreadGossipRaw(b)

			// Else forward to nodes
			return nil
		},
	},
}
