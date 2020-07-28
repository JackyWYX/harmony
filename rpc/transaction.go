package rpc

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/harmony-one/harmony/core/rawdb"
	"github.com/harmony-one/harmony/core/types"
	"github.com/harmony-one/harmony/core/vm"
	"github.com/harmony-one/harmony/hmy"
	internal_common "github.com/harmony-one/harmony/internal/common"
	"github.com/harmony-one/harmony/internal/params"
	"github.com/harmony-one/harmony/internal/utils"
	v1 "github.com/harmony-one/harmony/rpc/v1"
	v2 "github.com/harmony-one/harmony/rpc/v2"
)

const (
	defaultPageSize = uint32(100)
)

// PublicTransactionService provides an API to access Harmony's transaction service.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicTransactionService struct {
	hmy     *hmy.Harmony
	version Version
}

// NewPublicTransactionAPI creates a new API for the RPC interface
func NewPublicTransactionAPI(hmy *hmy.Harmony, version Version) rpc.API {
	return rpc.API{
		Namespace: version.Namespace(),
		Version:   APIVersion,
		Service:   &PublicTransactionService{hmy, version},
		Public:    true,
	}
}

// GetAccountNonce returns the nonce value of the given address for the given block number
func (s *PublicTransactionService) GetAccountNonce(
	ctx context.Context, address string, blockNumber BlockNumber,
) (uint64, error) {
	// Process number based on version
	blockNum := blockNumber.EthBlockNumber()

	// Response output is the same for all versions
	addr := internal_common.ParseAddr(address)
	return s.hmy.GetAccountNonce(ctx, addr, blockNum)
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number.
// Legacy for apiv1. For apiv2, please use getAccountNonce/getPoolNonce/getTransactionsCount/getStakingTransactionsCount apis for
// more granular transaction counts queries
// Note that the return type is an interface to account for the different versions
func (s *PublicTransactionService) GetTransactionCount(
	ctx context.Context, addr string, blockNumber BlockNumber,
) (response interface{}, err error) {
	// Process arguments based on version
	blockNum := blockNumber.EthBlockNumber()
	address := internal_common.ParseAddr(addr)

	// Fetch transaction count
	var nonce uint64
	if blockNum == rpc.PendingBlockNumber {
		// Ask transaction pool for the nonce which includes pending transactions
		nonce, err = s.hmy.GetPoolNonce(ctx, address)
		if err != nil {
			return nil, err
		}
	} else {
		// Resolve block number and use its state to ask for the nonce
		state, _, err := s.hmy.StateAndHeaderByNumber(ctx, blockNum)
		if err != nil {
			return nil, err
		}
		if state == nil {
			return nil, fmt.Errorf("state not found")
		}
		if state.Error() != nil {
			return nil, state.Error()
		}
		nonce = state.GetNonce(address)
	}

	// Format response according to version
	switch s.version {
	case V1:
		return (hexutil.Uint64)(nonce), nil
	case V2:
		return nonce, nil
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// GetTransactionsCount returns the number of regular transactions from genesis of input type ("SENT", "RECEIVED", "ALL")
func (s *PublicTransactionService) GetTransactionsCount(
	ctx context.Context, address, txType string,
) (count uint64, err error) {
	if !strings.HasPrefix(address, "one1") {
		// Handle hex address
		addr := internal_common.ParseAddr(address)
		address, err = internal_common.AddressToBech32(addr)
		if err != nil {
			return 0, err
		}
	}

	// Response output is the same for all versions
	return s.hmy.GetTransactionsCount(address, txType)
}

// GetStakingTransactionsCount returns the number of staking transactions from genesis of input type ("SENT", "RECEIVED", "ALL")
func (s *PublicTransactionService) GetStakingTransactionsCount(
	ctx context.Context, address, txType string,
) (count uint64, err error) {
	if !strings.HasPrefix(address, "one1") {
		// Handle hex address
		addr := internal_common.ParseAddr(address)
		address, err = internal_common.AddressToBech32(addr)
		if err != nil {
			return 0, err
		}
	}

	// Response output is the same for all versions
	return s.hmy.GetStakingTransactionsCount(address, txType)
}

// EstimateGas returns an estimate of the amount of gas needed to execute the
// given transaction against the current pending block.
func (s *PublicTransactionService) EstimateGas(
	ctx context.Context, args CallArgs,
) (hexutil.Uint64, error) {
	gas, err := doEstimateGas(ctx, s.hmy, args, nil)
	if err != nil {
		return 0, err
	}

	// Response output is the same for all versions
	return (hexutil.Uint64)(gas), nil
}

// GetTransactionByHash returns the plain transaction for the given hash
func (s *PublicTransactionService) GetTransactionByHash(
	ctx context.Context, hash common.Hash,
) (StructuredResponse, error) {
	// Try to return an already finalized transaction
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(s.hmy.ChainDb(), hash)
	if tx == nil {
		utils.Logger().Debug().
			Err(errors.Wrapf(ErrTransactionNotFound, "hash %v", hash.String())).
			Msgf("%v error at %v", LogTag, "GetTransactionByHash")
		// Legacy behavior is to not return RPC errors
		return nil, nil
	}
	block, err := s.hmy.GetBlock(ctx, blockHash)
	if err != nil {
		utils.Logger().Debug().
			Err(err).
			Msgf("%v error at %v", LogTag, "GetTransactionByHash")
		// Legacy behavior is to not return RPC errors
		return nil, nil
	}

	// Format the response according to the version
	switch s.version {
	case V1:
		tx, err := v1.NewRPCTransaction(tx, blockHash, blockNumber, block.Time().Uint64(), index)
		if err != nil {
			return nil, err
		}
		return NewStructuredResponse(tx)
	case V2:
		tx, err := v2.NewRPCTransaction(tx, blockHash, blockNumber, block.Time().Uint64(), index)
		if err != nil {
			return nil, err
		}
		return NewStructuredResponse(tx)
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// GetStakingTransactionByHash returns the staking transaction for the given hash
func (s *PublicTransactionService) GetStakingTransactionByHash(
	ctx context.Context, hash common.Hash,
) (StructuredResponse, error) {
	// Try to return an already finalized transaction
	stx, blockHash, blockNumber, index := rawdb.ReadStakingTransaction(s.hmy.ChainDb(), hash)
	if stx == nil {
		utils.Logger().Debug().
			Err(errors.Wrapf(ErrTransactionNotFound, "hash %v", hash.String())).
			Msgf("%v error at %v", LogTag, "GetStakingTransactionByHash")
		// Legacy behavior is to not return RPC errors
		return nil, nil
	}
	block, err := s.hmy.GetBlock(ctx, blockHash)
	if err != nil {
		utils.Logger().Debug().
			Err(err).
			Msgf("%v error at %v", LogTag, "GetStakingTransactionByHash")
		// Legacy behavior is to not return RPC errors
		return nil, nil
	}

	switch s.version {
	case V1:
		tx, err := v1.NewRPCStakingTransaction(stx, blockHash, blockNumber, block.Time().Uint64(), index)
		if err != nil {
			return nil, err
		}
		return NewStructuredResponse(tx)
	case V2:
		tx, err := v2.NewRPCStakingTransaction(stx, blockHash, blockNumber, block.Time().Uint64(), index)
		if err != nil {
			return nil, err
		}
		return NewStructuredResponse(tx)
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// GetTransactionsHistory returns the list of transactions hashes that involve a particular address.
func (s *PublicTransactionService) GetTransactionsHistory(
	ctx context.Context, args TxHistoryArgs,
) (StructuredResponse, error) {
	// Fetch transaction history
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

	result = returnHashesWithPagination(hashes, args.PageIndex, args.PageSize)

	// Just hashes have same response format for all versions
	if !args.FullTx {
		return StructuredResponse{"transactions": result}, nil
	}

	// Full transactions have different response format
	txs := []StructuredResponse{}
	for _, hash := range result {
		tx, err := s.GetTransactionByHash(ctx, hash)
		if err == nil {
			// Legacy behavior is to not return RPC errors
			txs = append(txs, tx)
		} else {
			utils.Logger().Debug().
				Err(err).
				Msgf("%v error at %v", LogTag, "GetTransactionsHistory")
		}
	}
	return StructuredResponse{"transactions": txs}, nil
}

// GetStakingTransactionsHistory returns the list of transactions hashes that involve a particular address.
func (s *PublicTransactionService) GetStakingTransactionsHistory(
	ctx context.Context, args TxHistoryArgs,
) (StructuredResponse, error) {
	// Fetch transaction history
	var address string
	var result []common.Hash
	var err error
	if strings.HasPrefix(args.Address, "one1") {
		address = args.Address
	} else {
		addr := internal_common.ParseAddr(args.Address)
		address, err = internal_common.AddressToBech32(addr)
		if err != nil {
			utils.Logger().Debug().
				Err(err).
				Msgf("%v error at %v", LogTag, "GetStakingTransactionsHistory")
			// Legacy behavior is to not return RPC errors
			return nil, nil
		}
	}
	hashes, err := s.hmy.GetStakingTransactionsHistory(address, args.TxType, args.Order)
	if err != nil {
		utils.Logger().Debug().
			Err(err).
			Msgf("%v error at %v", LogTag, "GetStakingTransactionsHistory")
		// Legacy behavior is to not return RPC errors
		return nil, nil
	}

	result = returnHashesWithPagination(hashes, args.PageIndex, args.PageSize)

	// Just hashes have same response format for all versions
	if !args.FullTx {
		return StructuredResponse{"staking_transactions": result}, nil
	}

	// Full transactions have different response format
	txs := []StructuredResponse{}
	for _, hash := range result {
		tx, err := s.GetStakingTransactionByHash(ctx, hash)
		if err == nil {
			txs = append(txs, tx)
		} else {
			utils.Logger().Debug().
				Err(err).
				Msgf("%v error at %v", LogTag, "GetStakingTransactionsHistory")
			// Legacy behavior is to not return RPC errors
		}
	}
	return StructuredResponse{"staking_transactions": txs}, nil
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block with the given block number.
// Note that the return type is an interface to account for the different versions
func (s *PublicTransactionService) GetBlockTransactionCountByNumber(
	ctx context.Context, blockNumber BlockNumber,
) (interface{}, error) {
	// Process arguments based on version
	blockNum := blockNumber.EthBlockNumber()

	// Fetch block
	block, err := s.hmy.BlockByNumber(ctx, blockNum)
	if err != nil {
		utils.Logger().Debug().
			Err(err).
			Msgf("%v error at %v", LogTag, "GetBlockTransactionCountByNumber")
		// Legacy behavior is to not return RPC errors
		return nil, nil
	}

	// Format response according to version
	switch s.version {
	case V1:
		return hexutil.Uint(len(block.Transactions())), nil
	case V2:
		return len(block.Transactions()), nil
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
// Note that the return type is an interface to account for the different versions
func (s *PublicTransactionService) GetBlockTransactionCountByHash(
	ctx context.Context, blockHash common.Hash,
) (interface{}, error) {
	// Fetch block
	block, err := s.hmy.GetBlock(ctx, blockHash)
	if err != nil {
		utils.Logger().Debug().
			Err(err).
			Msgf("%v error at %v", LogTag, "GetBlockTransactionCountByHash")
		// Legacy behavior is to not return RPC errors
		return nil, nil
	}

	// Format response according to version
	switch s.version {
	case V1:
		return hexutil.Uint(len(block.Transactions())), nil
	case V2:
		return len(block.Transactions()), nil
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// GetTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func (s *PublicTransactionService) GetTransactionByBlockNumberAndIndex(
	ctx context.Context, blockNumber BlockNumber, index TransactionIndex,
) (StructuredResponse, error) {
	// Process arguments based on version
	blockNum := blockNumber.EthBlockNumber()

	// Fetch Block
	block, err := s.hmy.BlockByNumber(ctx, blockNum)
	if err != nil {
		utils.Logger().Debug().
			Err(err).
			Msgf("%v error at %v", LogTag, "GetTransactionByBlockNumberAndIndex")
		// Legacy behavior is to not return RPC errors
		return nil, nil
	}

	// Format response according to version
	switch s.version {
	case V1:
		tx, err := v1.NewRPCTransactionFromBlockIndex(block, uint64(index))
		if err != nil {
			return nil, err
		}
		return NewStructuredResponse(tx)
	case V2:
		tx, err := v2.NewRPCTransactionFromBlockIndex(block, uint64(index))
		if err != nil {
			return nil, err
		}
		return NewStructuredResponse(tx)
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// GetTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (s *PublicTransactionService) GetTransactionByBlockHashAndIndex(
	ctx context.Context, blockHash common.Hash, index TransactionIndex,
) (StructuredResponse, error) {
	// Fetch Block
	block, err := s.hmy.GetBlock(ctx, blockHash)
	if err != nil {
		utils.Logger().Debug().
			Err(err).
			Msgf("%v error at %v", LogTag, "GetTransactionByBlockHashAndIndex")
		// Legacy behavior is to not return RPC errors
		return nil, nil
	}

	// Format response according to version
	switch s.version {
	case V1:
		tx, err := v1.NewRPCTransactionFromBlockIndex(block, uint64(index))
		if err != nil {
			return nil, err
		}
		return NewStructuredResponse(tx)
	case V2:
		tx, err := v2.NewRPCTransactionFromBlockIndex(block, uint64(index))
		if err != nil {
			return nil, err
		}
		return NewStructuredResponse(tx)
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// GetBlockStakingTransactionCountByNumber returns the number of staking transactions in the block with the given block number.
// Note that the return type is an interface to account for the different versions
func (s *PublicTransactionService) GetBlockStakingTransactionCountByNumber(
	ctx context.Context, blockNumber BlockNumber,
) (interface{}, error) {
	// Process arguments based on version
	blockNum := blockNumber.EthBlockNumber()

	// Fetch block
	block, err := s.hmy.BlockByNumber(ctx, blockNum)
	if err != nil {
		utils.Logger().Debug().
			Err(err).
			Msgf("%v error at %v", LogTag, "GetBlockStakingTransactionCountByNumber")
		// Legacy behavior is to not return RPC errors
		return nil, nil
	}

	// Format response according to version
	switch s.version {
	case V1:
		return hexutil.Uint(len(block.StakingTransactions())), nil
	case V2:
		return len(block.StakingTransactions()), nil
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// GetBlockStakingTransactionCountByHash returns the number of staking transactions in the block with the given hash.
// Note that the return type is an interface to account for the different versions
func (s *PublicTransactionService) GetBlockStakingTransactionCountByHash(
	ctx context.Context, blockHash common.Hash,
) (interface{}, error) {
	// Fetch block
	block, err := s.hmy.GetBlock(ctx, blockHash)
	if err != nil {
		utils.Logger().Debug().
			Err(err).
			Msgf("%v error at %v", LogTag, "GetBlockStakingTransactionCountByHash")
		// Legacy behavior is to not return RPC errors
		return nil, nil
	}

	// Format response according to version
	switch s.version {
	case V1:
		return hexutil.Uint(len(block.StakingTransactions())), nil
	case V2:
		return len(block.StakingTransactions()), nil
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// GetStakingTransactionByBlockNumberAndIndex returns the staking transaction for the given block number and index.
func (s *PublicTransactionService) GetStakingTransactionByBlockNumberAndIndex(
	ctx context.Context, blockNumber BlockNumber, index TransactionIndex,
) (StructuredResponse, error) {
	// Process arguments based on version
	blockNum := blockNumber.EthBlockNumber()

	// Fetch Block
	block, err := s.hmy.BlockByNumber(ctx, blockNum)
	if err != nil {
		utils.Logger().Debug().
			Err(err).
			Msgf("%v error at %v", LogTag, "GetStakingTransactionByBlockNumberAndIndex")
		// Legacy behavior is to not return RPC errors
		return nil, nil
	}

	// Format response according to version
	switch s.version {
	case V1:
		tx, err := v1.NewRPCStakingTransactionFromBlockIndex(block, uint64(index))
		if err != nil {
			return nil, err
		}
		return NewStructuredResponse(tx)
	case V2:
		tx, err := v2.NewRPCStakingTransactionFromBlockIndex(block, uint64(index))
		if err != nil {
			return nil, err
		}
		return NewStructuredResponse(tx)
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// GetStakingTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (s *PublicTransactionService) GetStakingTransactionByBlockHashAndIndex(
	ctx context.Context, blockHash common.Hash, index TransactionIndex,
) (StructuredResponse, error) {
	// Fetch Block
	block, err := s.hmy.GetBlock(ctx, blockHash)
	if err != nil {
		utils.Logger().Debug().
			Err(err).
			Msgf("%v error at %v", LogTag, "GetStakingTransactionByBlockHashAndIndex")
		// Legacy behavior is to not return RPC errors
		return nil, nil
	}

	// Format response according to version
	switch s.version {
	case V1:
		tx, err := v1.NewRPCStakingTransactionFromBlockIndex(block, uint64(index))
		if err != nil {
			return nil, err
		}
		return NewStructuredResponse(tx)
	case V2:
		tx, err := v2.NewRPCStakingTransactionFromBlockIndex(block, uint64(index))
		if err != nil {
			return nil, err
		}
		return NewStructuredResponse(tx)
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (s *PublicTransactionService) GetTransactionReceipt(
	ctx context.Context, hash common.Hash,
) (StructuredResponse, error) {
	// Fetch receipt for plain & staking transaction
	var tx types.PoolTransaction
	var blockHash common.Hash
	var blockNumber, index uint64
	tx, blockHash, blockNumber, index = rawdb.ReadTransaction(s.hmy.ChainDb(), hash)
	if tx == nil {
		tx, blockHash, blockNumber, index = rawdb.ReadStakingTransaction(s.hmy.ChainDb(), hash)
		if tx == nil {
			// Legacy behavior is to not return error if transaction is not found
			utils.Logger().Debug().
				Err(fmt.Errorf("unable to find plain tx or staking tx with hash %v", hash.String())).
				Msgf("%v error at %v", LogTag, "GetTransactionReceipt")
			return nil, nil
		}
	}
	receipts, err := s.hmy.GetReceipts(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	if len(receipts) <= int(index) {
		return nil, fmt.Errorf("index of transaction greater than number of receipts")
	}
	receipt := receipts[index]

	// Format response according to version
	switch s.version {
	case V1:
		RPCReceipt, err := v1.NewRPCReceipt(tx, blockHash, blockNumber, index, receipt)
		if err != nil {
			return nil, err
		}
		return NewStructuredResponse(RPCReceipt)
	case V2:
		RPCReceipt, err := v2.NewRPCReceipt(tx, blockHash, blockNumber, index, receipt)
		if err != nil {
			return nil, err
		}
		return NewStructuredResponse(RPCReceipt)
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// GetCXReceiptByHash returns the transaction for the given hash
func (s *PublicTransactionService) GetCXReceiptByHash(
	ctx context.Context, hash common.Hash,
) (StructuredResponse, error) {
	if cx, blockHash, blockNumber, _ := rawdb.ReadCXReceipt(s.hmy.ChainDb(), hash); cx != nil {
		// Format response according to version
		switch s.version {
		case V1:
			tx, err := v1.NewRPCCXReceipt(cx, blockHash, blockNumber)
			if err != nil {
				return nil, err
			}
			return NewStructuredResponse(tx)
		case V2:
			tx, err := v2.NewRPCCXReceipt(cx, blockHash, blockNumber)
			if err != nil {
				return nil, err
			}
			return NewStructuredResponse(tx)
		default:
			return nil, ErrUnknownRPCVersion
		}
	}
	utils.Logger().Debug().
		Err(fmt.Errorf("unable to found CX receipt for tx %v", hash.String())).
		Msgf("%v error at %v", LogTag, "GetCXReceiptByHash")
	return nil, nil // Legacy behavior is to not return an error here
}

// ResendCx requests that the egress receipt for the given cross-shard
// transaction be sent to the destination shard for credit.  This is used for
// unblocking a half-complete cross-shard transaction whose fund has been
// withdrawn already from the source shard but not credited yet in the
// destination account due to transient failures.
func (s *PublicTransactionService) ResendCx(ctx context.Context, txID common.Hash) (bool, error) {
	_, success := s.hmy.ResendCx(ctx, txID)

	// Response output is the same for all versions
	return success, nil
}

// returnHashesWithPagination returns result with pagination (offset, page in TxHistoryArgs).
func returnHashesWithPagination(hashes []common.Hash, pageIndex uint32, pageSize uint32) []common.Hash {
	size := defaultPageSize
	if pageSize > 0 {
		size = pageSize
	}
	if uint64(size)*uint64(pageIndex) >= uint64(len(hashes)) {
		return make([]common.Hash, 0)
	}
	if uint64(size)*uint64(pageIndex)+uint64(size) > uint64(len(hashes)) {
		return hashes[size*pageIndex:]
	}
	return hashes[size*pageIndex : size*pageIndex+size]
}

// doEstimateGas ..
func doEstimateGas(
	ctx context.Context, hmy *hmy.Harmony, args CallArgs, gasCap *big.Int,
) (uint64, error) {
	// Binary search the gas requirement, as it may be higher than the amount used
	var (
		lo  = params.TxGas - 1
		hi  uint64
		max uint64
	)
	blockNum := rpc.LatestBlockNumber
	if args.Gas != nil && uint64(*args.Gas) >= params.TxGas {
		hi = uint64(*args.Gas)
	} else {
		// Retrieve the blk to act as the gas ceiling
		blk, err := hmy.BlockByNumber(ctx, blockNum)
		if err != nil {
			return 0, err
		}
		hi = blk.GasLimit()
	}
	if gasCap != nil && hi > gasCap.Uint64() {
		hi = gasCap.Uint64()
	}
	max = hi

	// Use zero-address if none other is available
	if args.From == nil {
		args.From = &common.Address{}
	}

	// Create a helper to check if a gas allowance results in an executable transaction
	executable := func(gas uint64) bool {
		args.Gas = (*hexutil.Uint64)(&gas)

		_, _, failed, err := doCall(ctx, hmy, args, blockNum, vm.Config{}, 0, big.NewInt(int64(max)))
		if err != nil || failed {
			return false
		}
		return true
	}

	// Execute the binary search and hone in on an executable gas limit
	for lo+1 < hi {
		mid := (hi + lo) / 2
		if !executable(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}

	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == max {
		if !executable(hi) {
			return 0, fmt.Errorf("gas required exceeds allowance (%d) or always failing transaction", max)
		}
	}
	return hi, nil
}
