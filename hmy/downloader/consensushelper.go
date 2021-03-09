package downloader

import (
	"fmt"
	"math/big"

	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"

	bls_core "github.com/harmony-one/bls/ffi/go/bls"
	"github.com/harmony-one/harmony/consensus/quorum"
	"github.com/harmony-one/harmony/consensus/signature"
	"github.com/harmony-one/harmony/core/types"
	bls_cosi "github.com/harmony-one/harmony/crypto/bls"
	"github.com/harmony-one/harmony/internal/chain"
	"github.com/harmony-one/harmony/multibls"
	"github.com/harmony-one/harmony/shard"
)

// consensusHelperImpl helps to verify and write the block signature of itself.
type consensusHelperImpl struct {
	bc blockChain

	deciderCache    *lru.Cache
	shardStateCache *lru.Cache
}

func newConsensusHelper(bc blockChain) consensusHelper {
	deciderCache, _ := lru.New(5)
	shardStateCache, _ := lru.New(5)
	return &consensusHelperImpl{
		bc:              bc,
		deciderCache:    deciderCache,
		shardStateCache: shardStateCache,
	}
}

func (ch *consensusHelperImpl) verifyBlockSignature(block *types.Block) error {
	// TODO: This is the duplicate logic to the implementation of consensus. Better refactor
	//  to the blockchain structure
	commitSigBytes := signature.ConstructCommitPayload(ch.bc, block.Epoch(), block.Hash(),
		block.NumberU64(), block.Header().ViewID().Uint64())

	decider, err := ch.readDeciderByEpoch(block.Epoch())
	if err != nil {
		return err
	}
	sig, mask, err := decodeCommitSig(block.GetCurrentCommitSig(), decider.Participants())
	if err != nil {
		return err
	}
	if !decider.IsQuorumAchievedByMask(mask) {
		return errors.New("quorum not achieved")
	}
	if !sig.VerifyHash(mask.AggregatePublic, commitSigBytes) {
		return errors.New("aggregate signature failed verification")
	}
	return nil
}

func (ch *consensusHelperImpl) writeBlockSignature(block *types.Block) error {
	return ch.bc.WriteCommitSig(block.NumberU64(), block.GetCurrentCommitSig())
}

func (ch *consensusHelperImpl) getDeciderByEpoch(epoch *big.Int) (quorum.Decider, error) {
	epochUint := epoch.Uint64()
	if decider, ok := ch.deciderCache.Get(epochUint); ok && decider != nil {
		return decider.(quorum.Decider), nil
	}
	decider, err := ch.getDeciderByEpoch(epoch)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read quorum of epoch %v", epoch.Uint64())
	}
	ch.deciderCache.Add(epochUint, decider)
	return decider, nil
}

func (ch *consensusHelperImpl) readDeciderByEpoch(epoch *big.Int) (quorum.Decider, error) {
	isStaking := ch.bc.Config().IsStaking(epoch)
	decider := ch.getNewDecider(isStaking)
	ss, err := ch.getShardState(epoch)
	if err != nil {
		return nil, err
	}
	subComm, err := ss.FindCommitteeByID(ch.shardID())
	if err != nil {
		return nil, err
	}
	pubKeys, err := subComm.BLSPublicKeys()
	if err != nil {
		return nil, err
	}
	decider.UpdateParticipants(pubKeys)
	if _, err := decider.SetVoters(subComm, epoch); err != nil {
		return nil, err
	}
	return decider, nil
}

func (ch *consensusHelperImpl) getNewDecider(isStaking bool) quorum.Decider {
	if isStaking {
		return quorum.NewDecider(quorum.SuperMajorityVote, ch.bc.ShardID())
	} else {
		return quorum.NewDecider(quorum.SuperMajorityStake, ch.bc.ShardID())
	}
}

func (ch *consensusHelperImpl) getShardState(epoch *big.Int) (*shard.State, error) {
	if ss, ok := ch.shardStateCache.Get(epoch.Uint64()); ok && ss != nil {
		return ss.(*shard.State), nil
	}
	ss, err := ch.bc.ReadShardState(epoch)
	if err != nil {
		return nil, err
	}
	ch.shardStateCache.Add(epoch.Uint64(), ss)
	return ss, nil
}

func (ch *consensusHelperImpl) shardID() uint32 {
	return ch.bc.ShardID()
}

func decodeCommitSig(commitBytes []byte, publicKeys multibls.PublicKeys) (*bls_core.Sign, *bls_cosi.Mask, error) {
	if len(commitBytes) < bls_cosi.BLSSignatureSizeInBytes {
		return nil, nil, fmt.Errorf("unexpected signature bytes size: %v / %v", len(commitBytes),
			bls_cosi.BLSSignatureSizeInBytes)
	}
	return chain.ReadSignatureBitmapByPublicKeys(commitBytes, publicKeys)
}
