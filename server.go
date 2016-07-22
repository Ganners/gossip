package gossip

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ganners/gossip/envelope"
	"github.com/gogo/protobuf/proto"
)

// Server is the root structure for a gossip node
type Server struct {
	Name        string
	Description string

	Host string
	Port string

	Nodes  []GossipNode
	Logger Logger

	WorkersPerListener int
	WorkersPerHandler  int

	signals   chan os.Signal
	terminate chan struct{}

	handlers    []RequestHandler
	rawHandlers []RawRequestHandler
	schedulers  map[time.Duration]HandlerFunc

	// The protobuf enveloped message
	incomingMessage chan envelope.Envelope
}

// NewServer creates a new idle server ready to be started
func NewServer(
	name string,
	description string,
	logger Logger,
) (*Server, error) {

	// Retrieve the bootstrap nodes
	bootstrapNodes, err := bootstrapGossipNodes()
	if err != nil {
		return nil, err
	}

	server := &Server{
		Name:        name,
		Description: description,
		Nodes:       bootstrapNodes,
		Logger:      logger,

		terminate: make(chan struct{}, 1),
	}

	server.setupSignals()

	return server, nil
}

// Listens to operating system signals so that we can SIGTERM (etc.)
// the running server
func (s *Server) setupSignals() {
	s.signals = make(chan os.Signal, 1)
	signal.Notify(s.signals, os.Interrupt)
	signal.Notify(s.signals, syscall.SIGTERM)
	signal.Notify(s.signals, syscall.SIGILL)

	go func() {
		for {
			select {
			case <-s.signals:
				s.Logger.Debugf("Signal triggered")
				s.Terminate()
				os.Exit(1)
				return
			}
		}
	}()
}

// Start gives you back a server with a receive only channel, this
// channel will receive upon termination of the server which can come
// from a number of different sources.
func (s *Server) Start() <-chan struct{} {

	// Commence all handlers
	s.startRequestHandlers()

	// Commence all schedulers
	// @TODO(mark)

	s.startNetworkListener()

	return s.listenForTermination()
}

// Commences the network listening, will read in the TCP connections and
// convert to the envelope
func (s *Server) startNetworkListener() {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%s", s.Host, s.Port))
	if err != nil {
		s.Logger.Errorf("[Network Listener] Unable to start network listener: %s", err.Error())

		// Retrying
		time.Sleep(time.Second * 5)
		s.startNetworkListener()
	}

	// Loop and accept connections which immediately forward to a connection
	// handler
	for {
		conn, err := ln.Accept()
		if err != nil {
			s.Logger.Errorf("[Network Listener] Unable to accept connection: %s", err.Error())
			continue
		}

		go s.startConnectionHandler(conn)
	}
}

// Starts a connection handler for a given connection
func (s *Server) startConnectionHandler(conn net.Conn) {
	// Read the connection bytes
	reader := bufio.NewReader(conn)
	b, err := reader.ReadBytes('\n')
	if err != nil {
		s.Logger.Errorf("[Connection Handler] Unable to read bytes: %s", err.Error())
		return
	}

	// Forward gossip to other nodes we're connected to
	for _, node := range s.Nodes {
		err := node.SendMessage(b)
		if err != nil {
			// @TODO(mark): Mark a failure at node
			s.Logger.Errorf("[Forward Gossip] Unable to contact node %+v: %s", node, err.Error())
			return
		}
	}

	// Convert and downstream
	envelope := envelope.Envelope{}
	err = proto.Unmarshal(b, &envelope)
	if err != nil {
		s.Logger.Errorf("[Unmarshal Envelope] Unable to unmarshal envelope: %s", err.Error())
		return
	}

	// Send it downstream
	s.incomingMessage <- envelope
}

// When a message comes in, we forward that message to all handlers as
// many might want to deal with it
func (s *Server) forwardHandlers(message envelope.Envelope) {

	// Send to the raw handlers first
	for _, rawHandler := range s.rawHandlers {
		err := rawHandler.HandlerFunc(s, message.EncodedMessage)
		if err != nil {
			s.Logger.Errorf("[Handler] Error processing raw request: %s", err.Error())
		}
	}

	// Send to the handlers which require it to be unmarshaled
	for _, handler := range s.handlers {
		if handler.IsMatch(message.GetHeaders().Key) {
			unmarshalType := handler.UnmarshalType
			err := proto.Unmarshal(message.EncodedMessage, unmarshalType)
			if err != nil {
				s.Logger.Errorf("[Handler] Error unmarshaling request: %s", err.Error())
			}
			err = handler.HandlerFunc(s, unmarshalType)
			if err != nil {
				s.Logger.Errorf("[Handler] Error processing request: %s", err.Error())
			}
		}
	}
}

// Starts a handler for a single request, will be triggered when a
// server starts, and also when a new request gets added during runtime
func (s *Server) startRequestHandlers() {
	for i := 0; i < s.WorkersPerHandler; i++ {
		go func() {
			for {
				select {
				case <-s.terminate:
					return
				case message := <-s.incomingMessage:
					s.forwardHandlers(message)
				}
			}
		}()
	}
}

// Listens for termination to return on a shutdown channel to the
// caller
func (s *Server) listenForTermination() <-chan struct{} {
	shutdown := make(chan struct{}, 1)
	go func(shutdown chan struct{}) {
		for {
			select {
			case <-s.terminate:
				s.Logger.Debugf("Termination triggered")
				shutdown <- struct{}{}
				return
			}
		}
	}(shutdown)
	return shutdown
}

// Triggers a termination to send to the server which will cleanly shut down
func (s *Server) Terminate() {
	close(s.terminate)
}
