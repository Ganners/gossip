package gossip

import (
	"os"
	"os/signal"
	"syscall"
)

// Server is the root structure for a gossip node
type Server struct {
	Name        string
	Description string
	Nodes       []GossipNode
	Logger      Logger

	signals   chan os.Signal
	terminate chan struct{}
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
	return s.listenForTermination()
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
