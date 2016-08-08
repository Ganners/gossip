package gossip

import (
	"reflect"
	"testing"
)

func TestGossipNodesFromFlag(t *testing.T) {

	testCases := []struct {
		input       string
		outputNodes []*GossipNode
		outputError error
	}{
		{
			input:       "192.168.0.1:3030",
			outputError: nil,
			outputNodes: []*GossipNode{
				{
					Host: "192.168.0.1",
					Port: "3030",
				},
			},
		},
		{
			input:       "  192.168.0.1:3030  ",
			outputError: nil,
			outputNodes: []*GossipNode{
				{
					Host: "192.168.0.1",
					Port: "3030",
				},
			},
		},
		{
			input:       "192.168.0.1:3030,test.com:12345",
			outputError: nil,
			outputNodes: []*GossipNode{
				{
					Host: "192.168.0.1",
					Port: "3030",
				},
				{
					Host: "test.com",
					Port: "12345",
				},
			},
		},
		{
			input:       "192.168.0.1:3030, test.com:12345 ,",
			outputError: nil,
			outputNodes: []*GossipNode{
				{
					Host: "192.168.0.1",
					Port: "3030",
				},
				{
					Host: "test.com",
					Port: "12345",
				},
			},
		},
	}

	for _, test := range testCases {
		nodes, err := gossipNodesFromFlag(test.input)
		if err != test.outputError {
			t.Errorf("Expected error of %s, got %s", test.outputError, err)
		}
		if !reflect.DeepEqual(nodes, test.outputNodes) {
			t.Errorf("Expected output of %+v, got %+v", test.outputNodes, nodes)
		}
	}
}
