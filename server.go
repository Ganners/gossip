package gossip

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ganners/gossip/pb/envelope"
	"github.com/ganners/gossip/pb/subscribe"
	"github.com/gogo/protobuf/proto"
)

const (
	MessageMemory = time.Second * 30
)

// Server is the root structure for a gossip node
type Server struct {
	Name        string
	Description string

	Host string
	Port string

	nodes  map[string]*GossipNode
	Logger Logger

	workersPerHandler int

	signals   chan os.Signal
	terminate chan struct{}

	handlers   []RequestHandler
	schedulers map[time.Duration]HandlerFunc

	// The protobuf enveloped message
	incomingMessage chan *envelope.Envelope
	messagesSeen    map[string]struct{}
}

// NewServer creates a new idle server ready to be started
func NewServer(
	name string,
	description string,
	host string,
	port string,
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
		Logger:      logger,
		Host:        host,
		Port:        port,
		handlers:    defaultHandlers,

		// @TODO(mark): Make flag/config based or modifable
		workersPerHandler: 5,

		nodes:           bootstrapNodes,
		messagesSeen:    make(map[string]struct{}, 1000),
		terminate:       make(chan struct{}, 1),
		incomingMessage: make(chan *envelope.Envelope, 5),
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

	s.Logger.Debugf("Commencing server")

	// Start server
	go s.startNetworkListener()

	s.Logger.Debugf("Starting request handlers")

	// Commence all handlers
	go s.startRequestHandlers()

	s.Logger.Debugf("Sending subscriptions")

	// Send out subscription request
	go s.subscribeToNodes()

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

		s.Logger.Debugf("Accepted new connection")

		go s.startConnectionHandler(conn)
	}
}

// Starts a connection handler for a given connection
func (s *Server) startConnectionHandler(conn net.Conn) {
	defer conn.Close()
	request := make([]byte, 1024*2) // 2 MB is the max request size
	for {
		// Read the connection bytes
		n, err := conn.Read(request)
		b := request[:n]
		if err != nil {
			s.Logger.Errorf("[ReadLine] Unable to read line")
			return
		}

		s.Logger.Debugf("Reading # bytes %d", len(b))

		// Convert and downstream
		envelope := &envelope.Envelope{}
		err = proto.Unmarshal(b, envelope)
		if err != nil {
			s.Logger.Errorf("[Unmarshal Envelope] Unable to unmarshal envelope: %s", err.Error())
			return
		}

		// Send it downstream
		go func() {
			s.Logger.Debugf("Sending message downstream")
			s.incomingMessage <- envelope
		}()
	}
}

// When a message comes in, we forward that message to all handlers as
// many might want to deal with it
func (s *Server) forwardHandlers(message *envelope.Envelope) {
	// Send to the handlers which require it to be unmarshaled
	for _, handler := range s.handlers {
		if handler.IsMatch(message.GetHeaders().Key) {
			// The message to forward to the respective handler (dereference first)
			err := handler.HandlerFunc(s, *message)
			if err != nil {
				s.Logger.Errorf("[Handler] Error processing request: %s", err.Error())
			}
		}
	}
}

// Starts a handler for a single request, will be triggered when a
// server starts, and also when a new request gets added during runtime
func (s *Server) startRequestHandlers() {
	for i := 0; i < s.workersPerHandler; i++ {
		go func() {
			for {
				select {
				case <-s.terminate:
					return
				case message := <-s.incomingMessage:
					s.Logger.Debugf("Incoming message found: %s", message.Headers.Key)
					if alreadySeen := s.setMessageSeen(message); !alreadySeen {
						s.forwardHandlers(message)
					}
				}
			}
		}()
	}
}

// Sets the message as seen, adds a cleanup after a period of time. If the
// message has already been seen then this will return true, else false.
func (s *Server) setMessageSeen(envelope *envelope.Envelope) bool {
	// Return true if we've seen this before
	if _, seen := s.messagesSeen[envelope.Uid]; seen {
		return true
	}

	// Add it to set
	s.messagesSeen[envelope.Uid] = struct{}{}

	// Start a short lived goroutine which will fire after some seconds and
	// clean up
	go func(uid string) {
		<-time.After(MessageMemory)
		delete(s.messagesSeen, envelope.Uid)
	}(envelope.Uid)

	return false
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

// Spreads the raw bytes of a gossip, this needs to be an envelope to
// avoid breaking everything. Therefore considered unsafe to be public.
func (s *Server) spreadGossipRaw(b []byte) {
	// Select a sample
	// @TODO(mark): This should select a random distribution of nodes,
	//              not all nodes
	sample := s.nodes

	// Loop nodes and send to all
	for _, node := range sample {
		err := node.Connect()
		if err != nil {
			s.Logger.Errorf("Unable to connect to node: %s", err)
		}

		s.Logger.Debugf("Broadcasting # bytes: %d", len(b))

		err = node.SendMessage(b)
		if err != nil {
			s.Logger.Errorf("Unable to send to node: %s", err)
		}
	}
}

// Broadcasts a subscription notice to all nodes
func (s *Server) subscribeToNodes() {
	// No one to subscribe to if there are no nodes available
	if len(s.nodes) == 0 {
		return
	}

	subscription := &subscribe.Subscribe{
		Name:        s.Name,
		Description: s.Description,
		Host:        s.Host,
		Port:        s.Port,
	}

	// And send!
	s.Broadcast(
		"node.subscribe",
		subscription,
		int32(envelope.Envelope_ASYNC_REQUEST),
	)
}

// Public gossip sender
//
// The message type should be:
//  0 -> Async request
//  1 -> Sync request (will return a receipt)
//  3 -> Response (someone should be listening for the receipt)
//
// Will marshal to bytes, place inside an envelope which will then also be
// marshaled into bytes.
func (s *Server) Broadcast(
	key string,
	message proto.Message,
	messageType int32,
) (string, error) {
	// Check if the message type exists
	_, ok := envelope.Envelope_Type_name[messageType]
	if !ok {
		return "", fmt.Errorf("message type %d does not exist", messageType)
	}

	envelopeType := envelope.Envelope_Type(messageType)

	// Marshal the message
	b, err := proto.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("unable to marshal proto: %s", err)
	}

	// Generate a UID and, if necessary, a receipt
	uid := s.genUid(key)
	receipt := ""
	if envelopeType == envelope.Envelope_SYNC_REQUEST {
		receipt = s.genReceipt(uid)
	}

	// Put the envelope together and send
	envelope := &envelope.Envelope{
		Uid:  uid,
		Type: envelopeType,
		Headers: &envelope.Envelope_Header{
			Key:     key,
			Receipt: receipt,
		},
		PassedThrough:  0,
		EncodedMessage: b,
	}

	b, err = proto.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("unable to marshal envelope proto: %s", err)
	}

	s.spreadGossipRaw(b)

	return receipt, nil
}
