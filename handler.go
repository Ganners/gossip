package gossip

import (
	"strings"

	"github.com/gogo/protobuf/proto"
)

// A request handler will expect a given key enveloped with a
// proto.Message, and will execute a request based on that
//
// It doesn't need to send any response if it does not want, though it
// can trigger another message to a receipt if it wants
type RequestHandlerFunc func(server *Server, request proto.Message) error

// A handler doesn't take a request, it will be used to do scheduled
// things
type HandlerFunc func(server *Server) error

// Embeddable type which allows a key to be matched
type RequestMatcher struct {
	Matcher string
}

// The request handler wrapper, allows us to understand the key we're
// hunting for and also specify the type so we know how to unmarshal
// the protobuf
type RequestHandler struct {
	RequestMatcher
	UnmarshalType proto.Message
	HandlerFunc   RequestHandlerFunc
}

// IsMatch will test whether the key matches another given key which is
// a string
//
// '*' will mean everything matches
// 'customer*' will mean anything that starts with 'customer' matches
// 'customer*' will mean anything that ends with 'customer' matches
func (req *RequestMatcher) IsMatch(key string) bool {
	// Everything is a match
	if req.Matcher == "*" {
		return true
	}

	// Suffix match
	if req.Matcher[0] == '*' {
		if strings.HasSuffix(key, req.Matcher[1:]) {
			return true
		}
	}

	// Prefix match
	if req.Matcher[len(req.Matcher)-1] == '*' {
		if strings.HasPrefix(key, req.Matcher[:len(req.Matcher)-2]) {
			return true
		}
	}

	// Exact match
	if req.Matcher == key {
		return true
	}

	return false
}
