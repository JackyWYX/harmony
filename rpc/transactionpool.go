package rpc

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/harmony-one/harmony/core"
	"github.com/harmony-one/harmony/core/rawdb"
	"github.com/harmony-one/harmony/core/types"
	"github.com/harmony-one/harmony/hmy"
	internal_common "github.com/harmony-one/harmony/internal/common"
	staking "github.com/harmony-one/harmony/staking/types"
	"github.com/pkg/errors"
)

// TxHistoryArgs is struct to make GetTransactionsHistory request
type TxHistoryArgs struct {
	Address   string `json:"address"`
	PageIndex uint32 `json:"pageIndex"`
	PageSize  uint32 `json:"pageSize"`
	FullTx    bool   `json:"fullTx"`
	TxType    string `json:"txType"`
	Order     string `json:"order"`
}

// PublicTransactionPoolService exposes methods for the RPC interface
type PublicTransactionPoolService struct {
	hmy       *hmy.Harmony
	nonceLock *AddrLocker
	version   Version
}

// NewPublicTransactionPoolAPI creates a new API for the RPC interface
func NewPublicTransactionPoolAPI(hmy *hmy.Harmony, version Version) rpc.API {
	return rpc.API{
		Namespace: version.Namespace(),
		Version:   APIVersion,
		Service:   &PublicTransactionPoolService{hmy, new(AddrLocker), version},
		Public:    true,
	}
}

// GetTransactionsHistory returns the list of transactions hashes that involve a particular address.
func (s *PublicTransactionPoolService) GetTransactionsHistory(ctx context.Context, args TxHistoryArgs) (map[string]interface{}, error) {
	var address string
	var result []common.Hash
	var err error
	if strings.HasPrefix(args.Address, "one1") {
		address = args.Address
	} else {
		addr := internal_common.ParseAddr(args.Address)
		address, err = internal_common.AddressToBech32(addr)
		if err != nil {
			return nil, err
		}
	}
	hashes, err := s.hmy.GetTransactionsHistory(address, args.TxType, args.Order)
	if err != nil {
		return nil, err
	}
	result = ReturnWithPagination(hashes, args.PageIndex, args.PageSize)
	if !args.FullTx {
		return map[string]interface{}{"transactions": result}, nil
	}
	txs := []*RPCTransaction{}
	for _, hash := range result {
		tx := s.GetTransactionByHash(ctx, hash)
		txs = append(txs, tx)
	}
	return map[string]interface{}{"transactions": txs}, nil
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block with the given block number.
func (s *PublicTransactionPoolService) GetBlockTransactionCountByNumber(ctx context.Context, blockNum rpc.BlockNumber) *hexutil.Uint {
	if block, _ := s.hmy.BlockByNumber(ctx, blockNum); block != nil {
		n := hexutil.Uint(len(block.Transactions()))
		return &n
	}
	return nil
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
func (s *PublicTransactionPoolService) GetBlockTransactionCountByHash(ctx context.Context, blockHash common.Hash) *hexutil.Uint {
	if block, _ := s.hmy.GetBlock(ctx, blockHash); block != nil {
		n := hexutil.Uint(len(block.Transactions()))
		return &n
	}
	return nil
}

// GetTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func (s *PublicTransactionPoolService) GetTransactionByBlockNumberAndIndex(ctx context.Context, blockNum rpc.BlockNumber, index hexutil.Uint) *RPCTransaction {
	if block, _ := s.hmy.BlockByNumber(ctx, blockNum); block != nil {
		return newRPCTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (s *PublicTransactionPoolService) GetTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) *RPCTransaction {
	if block, _ := s.hmy.GetBlock(ctx, blockHash); block != nil {
		return newRPCTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetTransactionByHash returns the plain transaction for the given hash
func (s *PublicTransactionPoolService) GetTransactionByHash(ctx context.Context, hash common.Hash) *RPCTransaction {
	// Try to return an already finalized transaction
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(s.hmy.ChainDb(), hash)
	block, _ := s.hmy.GetBlock(ctx, blockHash)
	if block == nil {
		return nil
	}
	if tx != nil {
		return newRPCTransaction(tx, blockHash, blockNumber, block.Time().Uint64(), index)
	}
	// Transaction unknown, return as such
	return nil
}

// GetStakingTransactionByHash returns the staking transaction for the given hash
func (s *PublicTransactionPoolService) GetStakingTransactionByHash(ctx context.Context, hash common.Hash) *RPCStakingTransaction {
	// Try to return an already finalized transaction
	stx, blockHash, blockNumber, index := rawdb.ReadStakingTransaction(s.hmy.ChainDb(), hash)
	block, _ := s.hmy.GetBlock(ctx, blockHash)
	if block == nil {
		return nil
	}
	if stx != nil {
		return newRPCStakingTransaction(stx, blockHash, blockNumber, block.Time().Uint64(), index)
	}
	// Transaction unknown, return as such
	return nil
}

// GetStakingTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func (s *PublicTransactionPoolService) GetStakingTransactionByBlockNumberAndIndex(ctx context.Context, blockNum rpc.BlockNumber, index hexutil.Uint) *RPCStakingTransaction {
	if block, _ := s.hmy.BlockByNumber(ctx, blockNum); block != nil {
		return newRPCStakingTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetStakingTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (s *PublicTransactionPoolService) GetStakingTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) *RPCStakingTransaction {
	if block, _ := s.hmy.GetBlock(ctx, blockHash); block != nil {
		return newRPCStakingTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number.
// Legacy for apiv1. For apiv2, please use getAccountNonce/getPoolNonce/getTransactionsCount/getStakingTransactionsCount apis for
// more granular transaction counts queries
func (s *PublicTransactionPoolService) GetTransactionCount(ctx context.Context, addr string, blockNum rpc.BlockNumber) (*hexutil.Uint64, error) {
	address := internal_common.ParseAddr(addr)
	// Ask transaction pool for the nonce which includes pending transactions
	if blockNum == rpc.PendingBlockNumber {
		nonce, err := s.hmy.GetPoolNonce(ctx, address)
		if err != nil {
			return nil, err
		}
		return (*hexutil.Uint64)(&nonce), nil
	}
	// Resolve block number and use its state to ask for the nonce
	state, _, err := s.hmy.StateAndHeaderByNumber(ctx, blockNum)
	if state == nil || err != nil {
		return nil, err
	}
	nonce := state.GetNonce(address)
	return (*hexutil.Uint64)(&nonce), state.Error()
}

// GetTransactionsCount returns the number of regular transactions from genesis of input type ("SENT", "RECEIVED", "ALL")
func (s *PublicTransactionPoolService) GetTransactionsCount(ctx context.Context, address, txType string) (uint64, error) {
	var err error
	if !strings.HasPrefix(address, "one1") {
		addr := internal_common.ParseAddr(address)
		address, err = internal_common.AddressToBech32(addr)
		if err != nil {
			return 0, err
		}
	}
	return s.hmy.GetTransactionsCount(address, txType)
}

// GetStakingTransactionsCount returns the number of staking transactions from genesis of input type ("SENT", "RECEIVED", "ALL")
func (s *PublicTransactionPoolService) GetStakingTransactionsCount(ctx context.Context, address, txType string) (uint64, error) {
	var err error
	if !strings.HasPrefix(address, "one1") {
		addr := internal_common.ParseAddr(address)
		address, err = internal_common.AddressToBech32(addr)
		if err != nil {
			return 0, err
		}
	}
	return s.hmy.GetStakingTransactionsCount(address, txType)
}

// SendRawStakingTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *PublicTransactionPoolService) SendRawStakingTransaction(
	ctx context.Context, encodedTx hexutil.Bytes,
) (common.Hash, error) {
	if len(encodedTx) >= types.MaxEncodedPoolTransactionSize {
		err := errors.Wrapf(core.ErrOversizedData, "encoded tx size: %d", len(encodedTx))
		return common.Hash{}, err
	}
	tx := new(staking.StakingTransaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return common.Hash{}, err
	}
	c := s.hmy.ChainConfig().ChainID
	if id := tx.ChainID(); id.Cmp(c) != 0 {
		return common.Hash{}, errors.Wrapf(
			ErrInvalidChainID, "blockchain chain id:%s, given %s", c.String(), id.String(),
		)
	}
	return SubmitStakingTransaction(ctx, s.hmy, tx)
}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *PublicTransactionPoolService) SendRawTransaction(
	ctx context.Context, encodedTx hexutil.Bytes,
) (common.Hash, error) {
	if len(encodedTx) >= types.MaxEncodedPoolTransactionSize {
		err := errors.Wrapf(core.ErrOversizedData, "encoded tx size: %d", len(encodedTx))
		return common.Hash{}, err
	}
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return common.Hash{}, err
	}
	c := s.hmy.ChainConfig().ChainID
	if id := tx.ChainID(); id.Cmp(c) != 0 {
		return common.Hash{}, errors.Wrapf(
			ErrInvalidChainID, "blockchain chain id:%s, given %s", c.String(), id.String(),
		)
	}
	return SubmitTransaction(ctx, s.hmy, tx)
}

func (s *PublicTransactionPoolService) fillTransactionFields(tx *types.Transaction, fields map[string]interface{}) error {
	var err error
	fields["shardID"] = tx.ShardID()
	var signer types.Signer = types.FrontierSigner{}
	if tx.Protected() {
		signer = types.NewEIP155Signer(tx.ChainID())
	}
	from, _ := types.Sender(signer, tx)
	fields["from"] = from
	fields["to"] = ""
	if tx.To() != nil {
		fields["to"], err = internal_common.AddressToBech32(*tx.To())
		if err != nil {
			return err
		}
		fields["from"], err = internal_common.AddressToBech32(from)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *PublicTransactionPoolService) fillStakingTransactionFields(stx *staking.StakingTransaction, fields map[string]interface{}) error {
	from, err := stx.SenderAddress()
	if err != nil {
		return err
	}
	fields["sender"], err = internal_common.AddressToBech32(from)
	if err != nil {
		return err
	}
	fields["type"] = stx.StakingType()
	return nil
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (s *PublicTransactionPoolService) GetTransactionReceipt(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	var tx *types.Transaction
	var stx *staking.StakingTransaction
	var blockHash common.Hash
	var blockNumber, index uint64
	tx, blockHash, blockNumber, index = rawdb.ReadTransaction(s.hmy.ChainDb(), hash)
	if tx == nil {
		stx, blockHash, blockNumber, index = rawdb.ReadStakingTransaction(s.hmy.ChainDb(), hash)
		if stx == nil {
			return nil, nil
		}
	}
	receipts, err := s.hmy.GetReceipts(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	if len(receipts) <= int(index) {
		return nil, nil
	}
	receipt := receipts[index]
	fields := map[string]interface{}{
		"blockHash":         blockHash,
		"blockNumber":       hexutil.Uint64(blockNumber),
		"transactionHash":   hash,
		"transactionIndex":  hexutil.Uint64(index),
		"gasUsed":           hexutil.Uint64(receipt.GasUsed),
		"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
		"logsBloom":         receipt.Bloom,
	}
	if tx != nil {
		if err = s.fillTransactionFields(tx, fields); err != nil {
			return nil, err
		}
	} else { // stx not nil
		if err = s.fillStakingTransactionFields(stx, fields); err != nil {
			return nil, err
		}
	}
	// Assign receipt status or post state.
	if len(receipt.PostState) > 0 {
		fields["root"] = hexutil.Bytes(receipt.PostState)
	} else {
		fields["status"] = hexutil.Uint(receipt.Status)
	}
	if receipt.Logs == nil {
		fields["logs"] = [][]*types.Log{}
	}
	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if receipt.ContractAddress != (common.Address{}) {
		fields["contractAddress"] = receipt.ContractAddress
	}
	return fields, nil
}

// GetPoolStats returns stats for the tx-pool
func (s *PublicTransactionPoolService) GetPoolStats() map[string]interface{} {
	pendingCount, queuedCount := s.hmy.GetPoolStats()
	return map[string]interface{}{
		"executable-count":     pendingCount,
		"non-executable-count": queuedCount,
	}
}

// PendingTransactions returns the plain transactions that are in the transaction pool
func (s *PublicTransactionPoolService) PendingTransactions() ([]*RPCTransaction, error) {
	pending, err := s.hmy.GetPoolTransactions()
	if err != nil {
		return nil, err
	}
	transactions := make([]*RPCTransaction, 0)
	for i := range pending {
		if plainTx, ok := pending[i].(*types.Transaction); ok {
			if tx := newRPCTransaction(plainTx, common.Hash{}, 0, 0, 0); tx != nil {
				transactions = append(transactions, tx)
			}
		} else if _, ok := pending[i].(*staking.StakingTransaction); ok {
			continue // Do not return staking transactions here.
		} else {
			return nil, types.ErrUnknownPoolTxType
		}
	}
	return transactions, nil
}

// PendingStakingTransactions returns the staking transactions that are in the transaction pool
func (s *PublicTransactionPoolService) PendingStakingTransactions() ([]*RPCStakingTransaction, error) {
	pending, err := s.hmy.GetPoolTransactions()
	if err != nil {
		return nil, err
	}
	transactions := make([]*RPCStakingTransaction, 0)
	for i := range pending {
		if _, ok := pending[i].(*types.Transaction); ok {
			continue // Do not return plain transactions here
		} else if stakingTx, ok := pending[i].(*staking.StakingTransaction); ok {
			if tx := newRPCStakingTransaction(stakingTx, common.Hash{}, 0, 0, 0); tx != nil {
				transactions = append(transactions, tx)
			}
		} else {
			return nil, types.ErrUnknownPoolTxType
		}
	}
	return transactions, nil
}

// GetCurrentTransactionErrorSink ..
func (s *PublicTransactionPoolService) GetCurrentTransactionErrorSink() types.TransactionErrorReports {
	return s.hmy.GetCurrentTransactionErrorSink()
}

// GetCurrentStakingErrorSink ..
func (s *PublicTransactionPoolService) GetCurrentStakingErrorSink() types.TransactionErrorReports {
	return s.hmy.GetCurrentStakingErrorSink()
}

// GetCXReceiptByHash returns the transaction for the given hash
func (s *PublicTransactionPoolService) GetCXReceiptByHash(
	ctx context.Context, hash common.Hash,
) *RPCCXReceipt {
	if cx, blockHash, blockNumber, _ := rawdb.ReadCXReceipt(s.hmy.ChainDb(), hash); cx != nil {
		return newRPCCXReceipt(cx, blockHash, blockNumber)
	}
	return nil
}

// GetPendingCXReceipts ..
func (s *PublicTransactionPoolService) GetPendingCXReceipts(ctx context.Context) []*types.CXReceiptsProof {
	return s.hmy.GetPendingCXReceipts()
}