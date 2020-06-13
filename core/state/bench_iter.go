package state

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

func (db *DB) BenchIter() {
	timeStart := time.Now()
	it := trie.NewIterator(db.trie.NodeIterator(nil))

	i := 0
	for it.Next() {
		i++
		enc := it.Value
		var acc Account
		rlp.DecodeBytes(enc, &acc)
		db.db.ContractCode(crypto.Keccak256Hash(it.Key[:]), common.BytesToHash(acc.CodeHash))
	}
	elapsed := time.Since(timeStart)
	fmt.Printf("elapsed time: %v %v\n", elapsed, i)
}
