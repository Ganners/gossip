// Code generated by protoc-gen-go.
// source: envelope.proto
// DO NOT EDIT!

/*
Package envelope is a generated protocol buffer package.

It is generated from these files:
	envelope.proto

It has these top-level messages:
	Envelope
*/
package envelope

import proto "github.com/hailocab/protobuf/proto"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal

// Gives us some understand as to whether this is a request or a
// response
type Envelope_Type int32

const (
	Envelope_REQUEST  Envelope_Type = 0
	Envelope_RESPONSE Envelope_Type = 1
)

var Envelope_Type_name = map[int32]string{
	0: "REQUEST",
	1: "RESPONSE",
}
var Envelope_Type_value = map[string]int32{
	"REQUEST":  0,
	"RESPONSE": 1,
}

func (x Envelope_Type) String() string {
	return proto.EnumName(Envelope_Type_name, int32(x))
}

// An envelope is our internal type which will wrap the protobuf object.
// It contains common fields which will be used by the framework.
type Envelope struct {
	Type    Envelope_Type    `protobuf:"varint,1,opt,name=type,enum=Envelope_Type" json:"type,omitempty"`
	Headers *Envelope_Header `protobuf:"bytes,2,opt,name=headers" json:"headers,omitempty"`
	// The encodedMessage contains the embedded proto bytes
	EncodedMessage []byte `protobuf:"bytes,3,opt,name=encodedMessage,proto3" json:"encodedMessage,omitempty"`
}

func (m *Envelope) Reset()         { *m = Envelope{} }
func (m *Envelope) String() string { return proto.CompactTextString(m) }
func (*Envelope) ProtoMessage()    {}

func (m *Envelope) GetHeaders() *Envelope_Header {
	if m != nil {
		return m.Headers
	}
	return nil
}

// Contains some header meta-data about the request/response
type Envelope_Header struct {
	// The key will be what services can filter on
	Key string `protobuf:"bytes,1,opt,name=key" json:"key,omitempty"`
	// The receipt is how a request can have a response returned
	Receipt string `protobuf:"bytes,2,opt,name=receipt" json:"receipt,omitempty"`
}

func (m *Envelope_Header) Reset()         { *m = Envelope_Header{} }
func (m *Envelope_Header) String() string { return proto.CompactTextString(m) }
func (*Envelope_Header) ProtoMessage()    {}

func init() {
	proto.RegisterEnum("Envelope_Type", Envelope_Type_name, Envelope_Type_value)
}
