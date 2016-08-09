package gossip

import (
	"fmt"
	"time"
)

// Generates a unique ID based on the server name, request key and nanosecond
// timestamp. Doesn't guarantee absolute uniqueness but it's close for testing
// purposes
func (s *Server) genUid(key string) string {
	return fmt.Sprintf("%s-%s-%d", s.Name, key, time.Now().UnixNano())
}

// Generates a receipt. This will just append to the unique ID to make it
// simpler
func (s *Server) genReceipt(key string) string {
	return fmt.Sprintf("%s--receipt", key)
}
