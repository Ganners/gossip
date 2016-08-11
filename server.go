package gossip

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ganners/gossip/pb/envelope"
	"github.com/ganners/gossip/pb/subscribe"
	"github.com/gogo/protobuf/proto"
)

const (
	MessageMemory   = time.Second * 30
	ResponseTimeout = time.Second * 2
)

// Server is the root structure for a gossip node
type Server struct {
	Name        string
	Description string

	Host string
	Port string

	nodes  GossipNodes
	Logger Logger

	workersPerHandler int

	signals   chan os.Signal
	terminate chan struct{}

	// Handlers are stored in a set, we need to be able to quickly add/remove
	// from it
	handlersLock sync.RWMutex
	handlers     map[*RequestHandler]struct{}

	// @TODO(mark): Schedulers to be used for healthchecking and various things
	schedulersLock sync.RWMutex
	schedulers     map[time.Duration]HandlerFunc

	messagesLock sync.RWMutex
	messagesSeen map[string]struct{}

	// The protobuf enveloped message
	incomingMessage chan *envelope.Envelope
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

		// @TODO(mark): Make flag/config based or modifable
		workersPerHandler: 5,

		handlers:        make(map[*RequestHandler]struct{}, 100),
		nodes:           bootstrapNodes,
		messagesSeen:    make(map[string]struct{}, 1000),
		terminate:       make(chan struct{}, 1),
		incomingMessage: make(chan *envelope.Envelope, 5),
	}

	for _, handler := range defaultHandlers {
		server.Handle(handler.Matcher, handler.HandlerFunc)
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
	s.handlersLock.RLock()
	defer s.handlersLock.RUnlock()

	for handler, _ := range s.handlers {
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
					if alreadySeen := s.setMessageSeen(message); !alreadySeen {
						s.Logger.Debugf("Incoming message found: %s", message.Headers.Key)
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

	s.messagesLock.RLock()

	// Return true if we've seen this before
	if _, seen := s.messagesSeen[envelope.Uid]; seen {
		s.messagesLock.RUnlock()
		return true
	}
	s.messagesLock.RUnlock()

	// Add it to set
	s.messagesLock.Lock()
	s.messagesSeen[envelope.Uid] = struct{}{}
	s.messagesLock.Unlock()

	// Start a short lived goroutine which will fire after some seconds and
	// clean up
	go func(uid string) {
		<-time.After(MessageMemory)
		s.messagesLock.Lock()
		delete(s.messagesSeen, envelope.Uid)
		s.messagesLock.Unlock()
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

			// @TODO(mark): Need some strategy to allow for X retries before
			// declaring a node dead and gossiping about the death of the node
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

// Wrapper for Broadcast & AddResponseHandler, the return values are primarily
// an error on the initial request, and then two channels for either the
// response or timeout which can be handled by the caller
func (s *Server) BroadcastAndWaitForResponse(
	key string,
	message proto.Message,
) (error, <-chan envelope.Envelope, <-chan struct{}) {

	// Generate a UID and, if necessary, a receipt
	uid := s.genUid(key)
	receipt := s.genReceipt(uid)
	response, timeout := s.AddResponseHandler(receipt)

	err := s.broadcast(key, uid, receipt, message, int32(envelope.Envelope_SYNC_REQUEST))
	if err != nil {
		return err, nil, nil
	}

	return nil, response, timeout
}

// The public version
func (s *Server) Broadcast(
	key string,
	message proto.Message,
	messageType int32,
) (string, error) {
	uid := s.genUid(key)

	// Returning an empty string due to an API change
	// @TODO(mark): Remove from all usage code
	return "", s.broadcast(key, uid, "", message, messageType)
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
func (s *Server) broadcast(
	key string,
	uid string,
	receipt string,
	message proto.Message,
	messageType int32,
) error {
	// Check if the message type exists
	_, ok := envelope.Envelope_Type_name[messageType]
	if !ok {
		return fmt.Errorf("message type %d does not exist", messageType)
	}

	envelopeType := envelope.Envelope_Type(messageType)

	// Marshal the message
	b, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("unable to marshal proto: %s", err)
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
		return fmt.Errorf("unable to marshal envelope proto: %s", err)
	}

	s.spreadGossipRaw(b)

	return nil
}

// Adds a node to the list of gossip nodes, can be used externally to
// add a node programatically
func (s *Server) AddNode(node *GossipNode) error {
	if len(node.Host) == 0 {
		return errors.New("No host specified on node")
	}
	if len(node.Port) == 0 {
		return errors.New("No port specified on node")
	}
	if node.Host == s.Host && node.Port == s.Port {
		return errors.New("Can not add self as node")
	}
	s.nodes[node.Host+node.Port] = node
	return nil
}

// Adds a new handler to the server for a given filter
func (s *Server) Handle(filter string, handler RequestHandlerFunc) {
	s.handlersLock.Lock()
	s.handlers[&RequestHandler{
		RequestMatcher: RequestMatcher{filter},
		HandlerFunc:    handler,
	}] = struct{}{}
	s.handlersLock.Unlock()
}

// Adds a handler which will live for a short time
func (s *Server) AddResponseHandler(
	receipt string,
) (<-chan envelope.Envelope, <-chan struct{}) {

	// Where the response will be returned (or timeout)
	responseChan := make(chan envelope.Envelope, 1)
	timeoutChan := make(chan struct{}, 1)

	go func() {
		reqResponse := make(chan envelope.Envelope, 1)

		reqHandler := &RequestHandler{
			RequestMatcher: RequestMatcher{receipt}, // Listen for the receipt
			HandlerFunc: func(server *Server, envelope envelope.Envelope) error {
				reqResponse <- envelope // Send the envelope back
				return nil
			},
		}

		s.handlersLock.Lock()
		s.handlers[reqHandler] = struct{}{} // Add the handler
		s.handlersLock.Unlock()

		// Listen for response (and forward on), or timeout
		select {
		case response := <-reqResponse:
			responseChan <- response
		case <-time.After(ResponseTimeout):
			timeoutChan <- struct{}{}
		}

		// Remove the handler, no longer need it around!
		s.handlersLock.Lock()
		delete(s.handlers, reqHandler)
		s.handlersLock.Unlock()
	}()

	return responseChan, timeoutChan
}
