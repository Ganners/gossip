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
type RequestHandlerFunc func(request proto.Message) error

// A handler doesn't take a request, it will be used to do scheduled
// things
type HandlerFunc func() error

// The request handler wrapper, allows us to understand the key we're
// hunting for and also specify the type so we know how to unmarshal the
// protobuf
type RequestHandler struct {
	Key           string
	UnmarshalType proto.Message
	HandlerFunc   RequestHandlerFunc
}

// IsMatch will test whether the key matches another given key which is
// a string
//
// '*' will mean everything matches
// 'customer*' will mean anything that starts with 'customer' matches
// 'customer*' will mean anything that ends with 'customer' matches
func (req *RequestHandler) IsMatch(key string) bool {
	// Everything is a match
	if req.Key == "*" {
		return true
	}

	// Suffix match
	if req.Key[0] == '*' {
		if strings.HasSuffix(key, req.Key[1:]) {
			return true
		}
	}

	// Prefix match
	if req.Key[len(req.Key)-1] == '*' {
		if strings.HasPrefix(key, req.Key[:len(req.Key)-2]) {
			return true
		}
	}

	// Exact match
	if req.Key == key {
		return true
	}

	return false
}
