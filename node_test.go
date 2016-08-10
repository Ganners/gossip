package gossip

import (
	"reflect"
	"strings"
	"testing"
)

func TestGossipNodesFromFlag(t *testing.T) {

	testCases := []struct {
		input       string
		outputNodes map[string]*GossipNode
		outputError error
	}{
		{
			input:       "192.168.0.1:3030",
			outputError: nil,
			outputNodes: map[string]*GossipNode{
				"192.168.0.13030": {
					Host: "192.168.0.1",
					Port: "3030",
				},
			},
		},
		{
			input:       "  192.168.0.1:3030  ",
			outputError: nil,
			outputNodes: map[string]*GossipNode{
				"192.168.0.13030": {
					Host: "192.168.0.1",
					Port: "3030",
				},
			},
		},
		{
			input:       "192.168.0.1:3030,test.com:12345",
			outputError: nil,
			outputNodes: map[string]*GossipNode{
				"192.168.0.13030": {
					Host: "192.168.0.1",
					Port: "3030",
				},
				"test.com12345": {
					Host: "test.com",
					Port: "12345",
				},
			},
		},
		{
			input:       "192.168.0.1:3030, test.com:12345 ,",
			outputError: nil,
			outputNodes: map[string]*GossipNode{
				"192.168.0.13030": {
					Host: "192.168.0.1",
					Port: "3030",
				},
				"test.com12345": {
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

func TestGossipNodesString(t *testing.T) {

	gossipNodes := make(GossipNodes)
	gossipNodes["192.168.0.10001"] = &GossipNode{
		Name: "",
		Host: "192.168.0.1",
		Port: "0001",
	}
	gossipNodes["192.168.0.10002"] = &GossipNode{
		Name: "Some Service 1",
		Host: "192.168.0.1",
		Port: "0002",
	}
	gossipNodes["192.168.0.10003"] = &GossipNode{
		Name: "Some Other Service",
		Host: "192.168.0.1",
		Port: "0003",
	}
	gossipNodes["192.168.0.10004"] = &GossipNode{
		Name: "Login Service",
		Host: "192.168.0.1",
		Port: "0004",
	}

	expected := strings.Join([]string{
		"+---------------------+--------------+-------+",
		"| Service Name        | Host         | Port  |",
		"+---------------------+--------------+-------+",
		"|                     | 192.168.0.1  | 0001  |",
		"| Some Service 1      | 192.168.0.1  | 0002  |",
		"| Some Other Service  | 192.168.0.1  | 0003  |",
		"| Login Service       | 192.168.0.1  | 0004  |",
		"+---------------------+--------------+-------+\n",
	}, "\n")

	str := gossipNodes.String()
	if str != expected {
		t.Errorf("Expected table \n%s, got \n%s", expected, str)
	}
}
