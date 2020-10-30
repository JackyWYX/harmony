package sync

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"

	protobuf "github.com/golang/protobuf/proto"

	"github.com/harmony-one/harmony/consensus/engine"
	"github.com/harmony-one/harmony/p2p/stream/message"
	sttypes "github.com/harmony-one/harmony/p2p/stream/types"
	p2ptypes "github.com/harmony-one/harmony/p2p/types"
	libp2p_network "github.com/libp2p/go-libp2p-core/network"
)

// Stream is the structure for a stream to a peer running a single protocol
type Stream struct {
	st    libp2p_network.Stream // underlying libp2p stream
	chain engine.ChainReader
	// middleware
	stManager   sttypes.StreamManager
	rateLimiter sttypes.RateLimiter
	// metadata
	cMeta sttypes.Metadata
	// cache
	myHandshakeMsg *message.HandShakeMessage
	// flow control
	stop chan struct{}
}

// PeerID returns the PeerID of the underlying stream connection
func (st *Stream) PeerID() p2ptypes.PeerID {
	return st.cMeta.PeerID
}

// ProtoID returns the ProtoID of the stream connection
func (st *Stream) ProtoID() sttypes.ProtoID {
	return st.cMeta.ProtoID
}

// ProtoSpec return ProtoSpec of the stream connection
func (st *Stream) ProtoSpec() sttypes.ProtoSpec {
	return st.cMeta.ProtoSpec
}

// Direction return the direction of the stream connection
func (st *Stream) Direction() sttypes.Direction {
	return st.cMeta.Direction
}

// Close closes the stream
func (st *Stream) Close() {
	st.stop <- struct{}{}
}

func (st *Stream) handshake() error {
	myHandshake := st.getHandshakeMessage()

	if err := st.writeMsg(myHandshake); err != nil {
		return errors.Wrap(err, "write my handshake")
	}
	other, err := st.readMsg()
	if err != nil {
		return errors.Wrap(err, "read handshake")
	}
	otherHandshake, ok := other.(*message.HandShakeMessage)
	if !ok {
		return errors.New("not hand shake message")
	}
	return st.checkHandshakeMessage(otherHandshake)
}

func (st *Stream) handleRequest()

func (st *Stream) writeMsg(msg protobuf.Message) error {
	b, err := protobuf.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = st.st.Write(b)
	return err
}

func (st *Stream) readMsg() (protobuf.Message, error) {
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

func (st *Stream) getHandshakeMessage() *message.HandShakeMessage {
	if st.myHandshakeMsg != nil {
		return st.myHandshakeMsg
	}
	genesisHash := st.chain.GetHeaderByNumber(0).Hash()
	msg := &message.HandShakeMessage{
		GenesisHash: genesisHash[:],
	}
	st.myHandshakeMsg = msg
	return msg
}

func (st *Stream) checkHandshakeMessage(target *message.HandShakeMessage) error {
	self := st.myHandshakeMsg
	if !bytes.Equal(target.GenesisHash, self.GenesisHash) {
		return fmt.Errorf("unexpected genesis hash: %x / %x", target.GenesisHash, self.GenesisHash)
	}
	return nil
}
