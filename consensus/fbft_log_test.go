package consensus

import (
	"testing"

	msg_pb "github.com/harmony-one/harmony/api/proto/message"
	"github.com/harmony-one/harmony/crypto/bls"
)

func TestFBFTLog_id(t *testing.T) {
	tests := []FBFTMessage{
		{
			MessageType: msg_pb.MessageType_ANNOUNCE,
			BlockHash:   [32]byte{01, 02},
			SenderPubkeys: []*bls.PublicKeyWrapper{
				{
					Bytes: bls.SerializedPublicKey{0x01, 0x02},
				},
				{
					Bytes: bls.SerializedPublicKey{0x02, 0x04},
				},
			},
		},
		{
			MessageType: msg_pb.MessageType_COMMIT,
			BlockHash:   [32]byte{02, 03},
		},
	}
	for _, msg := range tests {
		id := msg.id()

		if idDup := msg.id(); idDup != id {
			t.Errorf("id not replicable")
		}
		msg.SenderPubkeys = append(msg.SenderPubkeys, &bls.PublicKeyWrapper{
			Bytes: bls.SerializedPublicKey{0x10, 0x11},
		})
		if idDiff := msg.id(); idDiff == id {
			t.Errorf("id not unique")
		}
	}
}

func BenchmarkFBFGLog_id(b *testing.B) {
	msg := FBFTMessage{
		MessageType: msg_pb.MessageType_ANNOUNCE,
		BlockHash:   [32]byte{01, 02},
		SenderPubkeys: []*bls.PublicKeyWrapper{
			{
				Bytes: bls.SerializedPublicKey{0x01, 0x02},
			},
			{
				Bytes: bls.SerializedPublicKey{0x02, 0x04},
			},
		},
	}

	for i := 0; i < b.N; i++ {
		msg.id()
	}
}

func TestGetMessagesByTypeSeqViewHash(t *testing.T) {
	pbftMsg := FBFTMessage{
		MessageType: msg_pb.MessageType_ANNOUNCE,
		BlockNum:    2,
		ViewID:      3,
		BlockHash:   [32]byte{01, 02},
	}
	log := NewFBFTLog()
	log.AddMessage(&pbftMsg)

	found := log.GetMessagesByTypeSeqViewHash(
		msg_pb.MessageType_ANNOUNCE, 2, 3, [32]byte{01, 02},
	)
	if len(found) != 1 {
		t.Error("cannot find existing message")
	}

	notFound := log.GetMessagesByTypeSeqViewHash(
		msg_pb.MessageType_ANNOUNCE, 2, 3, [32]byte{01, 03},
	)
	if len(notFound) > 0 {
		t.Error("find message that not exist")
	}
}

func TestHasMatchingAnnounce(t *testing.T) {
	pbftMsg := FBFTMessage{
		MessageType: msg_pb.MessageType_ANNOUNCE,
		BlockNum:    2,
		ViewID:      3,
		BlockHash:   [32]byte{01, 02},
	}
	log := NewFBFTLog()
	log.AddMessage(&pbftMsg)
	found := log.HasMatchingViewAnnounce(2, 3, [32]byte{01, 02})
	if !found {
		t.Error("found should be true")
	}

	notFound := log.HasMatchingViewAnnounce(2, 3, [32]byte{02, 02})
	if notFound {
		t.Error("notFound should be false")
	}
}
