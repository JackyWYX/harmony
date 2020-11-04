package sttypes

import (
	"io/ioutil"

	protobuf "github.com/golang/protobuf/proto"
	p2ptypes "github.com/harmony-one/harmony/p2p/types"
	libp2p_network "github.com/libp2p/go-libp2p-core/network"
)

// Stream is the interface for streams implemented in each service.
// The stream interface is used for stream management as well as rate limiters
type Stream interface {
	ID() StreamID
	PeerID() p2ptypes.PeerID
	ProtoID() ProtoID
	Close() error
}

// BaseStream is the wrapper around
type BaseStream struct {
	id StreamID
	st libp2p_network.Stream
}

// StreamID contains the necessary information for identifyign a stream.
// Currently, it consist of peer ID and proto ID
type StreamID struct {
	PeerID  p2ptypes.PeerID
	ProtoID ProtoID
}

// Meta return the StreamID of the stream
func (st *BaseStream) ID() StreamID {
	return st.id
}

// PeerID return the peer id of the stream
func (st *BaseStream) PeerID() p2ptypes.PeerID {
	return st.id.PeerID
}

// ProtoID return the remote protocol ID of the stream
func (st *BaseStream) ProtoID() ProtoID {
	return st.id.ProtoID
}

// Close close the stream on both sides.
func (st *BaseStream) Close() error {
	return st.st.Reset()
}

// WriteMsg write the protobuf message to the stream
func (st *BaseStream) WriteMsg(msg protobuf.Message) error {
	b, err := protobuf.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = st.st.Write(b)
	return err
}

// ReadMsg read the protobuf message from the stream
func (st *BaseStream) ReadMsg() (protobuf.Message, error) {
	b, err := ioutil.ReadAll(st.st)
	if err != nil {
		return nil, err
	}
	var msg protobuf.Message
	if err := protobuf.Unmarshal(b, msg); err != nil {
		return nil, err
	}
	return msg, nil
}
