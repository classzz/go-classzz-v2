// Copyright 2015 The go-classzz-v2 Authors
// This file is part of the go-classzz-v2 library.
//
// The go-classzz-v2 library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-classzz-v2 library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-classzz-v2 library. If not, see <http://www.gnu.org/licenses/>.

package czz

import (
	"context"
	"errors"
	"math/big"

	"github.com/classzz/go-classzz-v2/accounts"
	"github.com/classzz/go-classzz-v2/common"
	"github.com/classzz/go-classzz-v2/consensus"
	"github.com/classzz/go-classzz-v2/core"
	"github.com/classzz/go-classzz-v2/core/bloombits"
	"github.com/classzz/go-classzz-v2/core/rawdb"
	"github.com/classzz/go-classzz-v2/core/state"
	"github.com/classzz/go-classzz-v2/core/types"
	"github.com/classzz/go-classzz-v2/core/vm"
	"github.com/classzz/go-classzz-v2/czz/downloader"
	"github.com/classzz/go-classzz-v2/czz/gasprice"
	"github.com/classzz/go-classzz-v2/czzdb"
	"github.com/classzz/go-classzz-v2/event"
	"github.com/classzz/go-classzz-v2/miner"
	"github.com/classzz/go-classzz-v2/params"
	"github.com/classzz/go-classzz-v2/rpc"
)

// EthAPIBackend implements czzapi.Backend for full nodes
type EthAPIBackend struct {
	extRPCEnabled       bool
	allowUnprotectedTxs bool
	czz                 *Classzz
	gpo                 *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *EthAPIBackend) ChainConfig() *params.ChainConfig {
	return b.czz.blockchain.Config()
}

func (b *EthAPIBackend) CurrentBlock() *types.Block {
	return b.czz.blockchain.CurrentBlock()
}

func (b *EthAPIBackend) SetHead(number uint64) {
	b.czz.handler.downloader.Cancel()
	b.czz.blockchain.SetHead(number)
}

func (b *EthAPIBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.czz.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.czz.blockchain.CurrentBlock().Header(), nil
	}
	return b.czz.blockchain.GetHeaderByNumber(uint64(number)), nil
}

func (b *EthAPIBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.czz.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.czz.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		return header, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *EthAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.czz.blockchain.GetHeaderByHash(hash), nil
}

func (b *EthAPIBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.czz.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.czz.blockchain.CurrentBlock(), nil
	}
	return b.czz.blockchain.GetBlockByNumber(uint64(number)), nil
}

func (b *EthAPIBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.czz.blockchain.GetBlockByHash(hash), nil
}

func (b *EthAPIBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.czz.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.czz.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		block := b.czz.blockchain.GetBlock(hash, header.Number.Uint64())
		if block == nil {
			return nil, errors.New("header found, but block body is missing")
		}
		return block, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *EthAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, state := b.czz.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, err := b.czz.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *EthAPIBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header, err := b.HeaderByHash(ctx, hash)
		if err != nil {
			return nil, nil, err
		}
		if header == nil {
			return nil, nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.czz.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, nil, errors.New("hash is not currently canonical")
		}
		stateDb, err := b.czz.BlockChain().StateAt(header.Root)
		return stateDb, header, err
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *EthAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.czz.blockchain.GetReceiptsByHash(hash), nil
}

func (b *EthAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	receipts := b.czz.blockchain.GetReceiptsByHash(hash)
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *EthAPIBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
	return b.czz.blockchain.GetTdByHash(hash)
}

func (b *EthAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config) (*vm.EVM, func() error, error) {
	vmError := func() error { return nil }
	if vmConfig == nil {
		vmConfig = b.czz.blockchain.GetVMConfig()
	}
	txContext := core.NewEVMTxContext(msg)
	context := core.NewEVMBlockContext(header, b.czz.BlockChain(), nil)
	return vm.NewEVM(context, txContext, state, b.czz.blockchain.Config(), *vmConfig), vmError, nil
}

func (b *EthAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.czz.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *EthAPIBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.czz.miner.SubscribePendingLogs(ch)
}

func (b *EthAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.czz.BlockChain().SubscribeChainEvent(ch)
}

func (b *EthAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.czz.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *EthAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.czz.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *EthAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.czz.BlockChain().SubscribeLogsEvent(ch)
}

func (b *EthAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.czz.txPool.AddLocal(signedTx)
}

func (b *EthAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.czz.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *EthAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.czz.txPool.Get(hash)
}

func (b *EthAPIBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(b.czz.ChainDb(), txHash)
	return tx, blockHash, blockNumber, index, nil
}

func (b *EthAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.czz.txPool.Nonce(addr), nil
}

func (b *EthAPIBackend) Stats() (pending int, queued int) {
	return b.czz.txPool.Stats()
}

func (b *EthAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.czz.TxPool().Content()
}

func (b *EthAPIBackend) TxPool() *core.TxPool {
	return b.czz.TxPool()
}

func (b *EthAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.czz.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *EthAPIBackend) Downloader() *downloader.Downloader {
	return b.czz.Downloader()
}

func (b *EthAPIBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *EthAPIBackend) ChainDb() czzdb.Database {
	return b.czz.ChainDb()
}

func (b *EthAPIBackend) EventMux() *event.TypeMux {
	return b.czz.EventMux()
}

func (b *EthAPIBackend) AccountManager() *accounts.Manager {
	return b.czz.AccountManager()
}

func (b *EthAPIBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *EthAPIBackend) UnprotectedAllowed() bool {
	return b.allowUnprotectedTxs
}

func (b *EthAPIBackend) RPCGasCap() uint64 {
	return b.czz.config.RPCGasCap
}

func (b *EthAPIBackend) RPCTxFeeCap() float64 {
	return b.czz.config.RPCTxFeeCap
}

func (b *EthAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.czz.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *EthAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.czz.bloomRequests)
	}
}

func (b *EthAPIBackend) Engine() consensus.Engine {
	return b.czz.engine
}

func (b *EthAPIBackend) CurrentHeader() *types.Header {
	return b.czz.blockchain.CurrentHeader()
}

func (b *EthAPIBackend) Miner() *miner.Miner {
	return b.czz.Miner()
}

func (b *EthAPIBackend) StartMining(threads int) error {
	return b.czz.StartMining(threads)
}

func (b *EthAPIBackend) StateAtBlock(ctx context.Context, block *types.Block, reexec uint64, base *state.StateDB, checkLive bool) (*state.StateDB, error) {
	return b.czz.stateAtBlock(block, reexec, base, checkLive)
}

func (b *EthAPIBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, error) {
	return b.czz.stateAtTransaction(block, txIndex, reexec)
}
