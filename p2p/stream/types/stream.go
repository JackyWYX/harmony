package sttypes

import (
	"fmt"
	"sync"

	libp2p_network "github.com/libp2p/go-libp2p-core/network"
)

// Stream is the interface for streams implemented in each service.
// The stream interface is used for stream management as well as rate limiters
type Stream interface {
	ID() StreamID
	ProtoID() ProtoID
	ProtoSpec() (ProtoSpec, error)
	WriteBytes([]byte) error
	ReadBytes() ([]byte, error)
	Close() error // Make sure streams can handle multiple calls of Close
}

// BaseStream is the wrapper around
type BaseStream struct {
	raw libp2p_network.Stream

	// parse protocol spec fields
	spec     ProtoSpec
	specErr  error
	specOnce sync.Once
}

// NewBaseStream creates BaseStream as the wrapper of libp2p Stream
func NewBaseStream(st libp2p_network.Stream) *BaseStream {
	return &BaseStream{
		raw: st,
	}
}

// StreamID is the unique identifier for the stream. It has the value of
// libp2p_network.Stream.ID()
type StreamID string

// Meta return the StreamID of the stream
func (st *BaseStream) ID() StreamID {
	return StreamID(st.raw.Conn().ID())
}

// ProtoID return the remote protocol ID of the stream
func (st *BaseStream) ProtoID() ProtoID {
	return ProtoID(st.raw.Protocol())
}

// ProtoSpec get the parsed protocol Specifier of the stream
func (st *BaseStream) ProtoSpec() (ProtoSpec, error) {
	st.specOnce.Do(func() {
		st.spec, st.specErr = ProtoIDToProtoSpec(st.ProtoID())
	})
	return st.spec, st.specErr
}

// Close close the stream on both sides.
func (st *BaseStream) Close() error {
	return st.raw.Reset()
}

// WriteBytes write the bytes to the stream
func (st *BaseStream) WriteBytes(b []byte) error {
	n, err := st.raw.Write(b)
	fmt.Println(n)
	return err
}

// ReadMsg read the bytes from the stream
func (st *BaseStream) ReadBytes() ([]byte, error) {
	//b, err := ioutil.ReadAll(st.raw)
	//if err != nil {
	//	return nil, err
	//}
	b := make([]byte, 128)
	n, err := st.raw.Read(b)
	if err != nil {
		fmt.Println("read error")
		return nil, err
	}
	fmt.Println("read success", n)
	return b[n:], nil
}
