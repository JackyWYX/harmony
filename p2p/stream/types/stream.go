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
	PeerID() p2ptypes.PeerID
	ProtoID() ProtoID
	ProtoSpec() ProtoSpec
	Close() error
}

// BaseStream is the wrapper around
type BaseStream struct {
	meta Metadata
	st   libp2p_network.Stream
}

// Metadata contains the necessary information for stream management
type Metadata struct {
	PeerID    p2ptypes.PeerID
	ProtoID   ProtoID
	ProtoSpec ProtoSpec
}

// PeerID return the peer ID of the stream
func (st *BaseStream) PeerID() p2ptypes.PeerID {
	return st.meta.PeerID
}

// ProtoID return the protocol ID of the stream
func (st *BaseStream) ProtoID() ProtoID {
	return st.meta.ProtoID
}

// ProtoSpec return the ProtoSpec of the stream
func (st *BaseStream) ProtoSpec() ProtoSpec {
	return st.meta.ProtoSpec
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
