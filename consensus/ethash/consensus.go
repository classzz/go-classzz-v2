// Copyright 2017 The go-classzz-v2 Authors
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

package ethash

import (
	"errors"
	"fmt"
	"github.com/classzz/go-classzz-v2/core/vm"
	"math/big"
	"runtime"
	"time"

	"github.com/classzz/go-classzz-v2/common"
	"github.com/classzz/go-classzz-v2/consensus"
	"github.com/classzz/go-classzz-v2/core/state"
	"github.com/classzz/go-classzz-v2/core/types"
	"github.com/classzz/go-classzz-v2/params"
	"github.com/classzz/go-classzz-v2/rlp"
	"github.com/classzz/go-classzz-v2/trie"
	"golang.org/x/crypto/sha3"
)

// Ethash proof-of-work protocol constants.
var (
	BlockReward                   = new(big.Int).Mul(big.NewInt(100), big.NewInt(1e+18)) // Block reward in wei for successfully mining a block
	allowedFutureBlockTimeSeconds = big.NewInt(int64(15))                                // Max seconds from current time allowed for blocks, before they're considered future blocks

	SubsidyReductionInterval = big.NewInt(1000000)
	// calcDifficultyEip2384 is the difficulty adjustment algorithm as specified by EIP 2384.
	// It offsets the bomb 4M blocks from Constantinople, so in total 9M blocks.
	// Specification EIP-2384: https://eips.classzz.org/EIPS/eip-2384
	calcDifficulty = makeDifficultyCalculator()
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	errOlderBlockTime    = errors.New("timestamp older than parent")
	errInvalidDifficulty = errors.New("non-positive difficulty")
	errInvalidMixDigest  = errors.New("invalid mix digest")
	errInvalidPoW        = errors.New("invalid proof-of-work")
)

// Author implements consensus.Engine, returning the header's coinbase as the
// proof-of-work verified author of the block.
func (ethash *Ethash) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules of the
// stock Classzz ethash engine.
func (ethash *Ethash) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool, factor *big.Int) error {
	// If we're running a full engine faking, accept any input as valid
	if ethash.config.PowMode == ModeFullFake {
		return nil
	}
	// Short circuit if the header is known, or its parent not
	number := header.Number.Uint64()
	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// Sanity checks passed, do a proper verification
	return ethash.verifyHeader(chain, header, parent, seal, time.Now().Unix(), factor)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications.
func (ethash *Ethash) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header,
	seals []bool, factors []*big.Int) (chan<- struct{}, <-chan error) {
	// If we're running a full engine faking, accept any input as valid
	if ethash.config.PowMode == ModeFullFake || len(headers) == 0 {
		abort, results := make(chan struct{}), make(chan error, len(headers))
		for i := 0; i < len(headers); i++ {
			results <- nil
		}
		return abort, results
	}
	if len(headers) != len(factors) {
		abort, results := make(chan struct{}), make(chan error, len(headers))
		results <- consensus.ErrInvalidFactors
		return abort, results
	}
	// Spawn as many workers as allowed threads
	workers := runtime.GOMAXPROCS(0)
	if len(headers) < workers {
		workers = len(headers)
	}

	// Create a task channel and spawn the verifiers
	var (
		inputs  = make(chan int)
		done    = make(chan int, workers)
		errors  = make([]error, len(headers))
		abort   = make(chan struct{})
		unixNow = time.Now().Unix()
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				factor := factors[index]
				errors[index] = ethash.verifyHeaderWorker(chain, headers, seals, index, unixNow, factor)
				done <- index
			}
		}()
	}

	errorsOut := make(chan error, len(headers))
	go func() {
		defer close(inputs)
		var (
			in, out = 0, 0
			checked = make([]bool, len(headers))
			inputs  = inputs
		)
		for {
			select {
			case inputs <- in:
				if in++; in == len(headers) {
					// Reached end of headers. Stop sending to workers.
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errors[out]
					if out == len(headers)-1 {
						return
					}
				}
			case <-abort:
				return
			}
		}
	}()
	return abort, errorsOut
}

func (ethash *Ethash) verifyHeaderWorker(chain consensus.ChainHeaderReader, headers []*types.Header,
	seals []bool, index int, unixNow int64, factor *big.Int) error {
	var parent *types.Header
	if index == 0 {
		parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
	} else if headers[index-1].Hash() == headers[index].ParentHash {
		parent = headers[index-1]
	}
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	return ethash.verifyHeader(chain, headers[index], parent, seals[index], unixNow, factor)
}

// verifyHeader checks whether a header conforms to the consensus rules of the
// stock Classzz ethash engine.
// See YP section 4.3.4. "Block Header Validity"
func (ethash *Ethash) verifyHeader(chain consensus.ChainHeaderReader, header, parent *types.Header,
	seal bool, unixNow int64, factor *big.Int) error {
	// Ensure that the header's extra-data section is of a reasonable size
	if uint64(len(header.Extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("extra-data too long: %d > %d", len(header.Extra), params.MaximumExtraDataSize)
	}
	// Verify the header's timestamp
	if header.Time > uint64(unixNow+allowedFutureBlockTimeSeconds.Int64()) {
		return consensus.ErrFutureBlock
	}
	if header.Time <= parent.Time {
		return errOlderBlockTime
	}
	// Verify the block's difficulty based on its timestamp and parent's difficulty
	expected := ethash.CalcDifficulty(chain, header.Time, parent)

	if expected.Cmp(header.Difficulty) != 0 {
		return fmt.Errorf("invalid difficulty: have %v, want %v", header.Difficulty, expected)
	}
	// Verify that the gas limit is <= 2^63-1
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, cap)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}

	// Verify that the gas limit remains within allowed bounds
	diff := int64(parent.GasLimit) - int64(header.GasLimit)
	if diff < 0 {
		diff *= -1
	}
	limit := parent.GasLimit / params.GasLimitBoundDivisor

	if uint64(diff) >= limit || header.GasLimit < params.MinGasLimit {
		return fmt.Errorf("invalid gas limit: have %d, want %d += %d", header.GasLimit, parent.GasLimit, limit)
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(big.NewInt(1)) != 0 {
		return consensus.ErrInvalidNumber
	}
	// Verify the engine specific seal securing the block
	if seal {
		if err := ethash.verifySeal(chain, header, factor); err != nil {
			return err
		}
	}
	return nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
func (ethash *Ethash) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return CalcDifficulty(chain.Config(), time, parent)
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
func CalcDifficulty(config *params.ChainConfig, time uint64, parent *types.Header) *big.Int {
	// next := new(big.Int).Add(parent.Number, big1)
	return calcDifficulty(time, parent)
}

// Some weird constants to avoid constant memory allocs for them.
var (
	big1       = big.NewInt(1)
	bigMinus99 = big.NewInt(-99)
)

// makeDifficultyCalculator creates a difficultyCalculator with the given bomb-delay.
// the difficulty is calculated with Byzantium rules, which differs from Homestead in
// how uncles affect the calculation
func makeDifficultyCalculator() func(time uint64, parent *types.Header) *big.Int {
	// Note, the calculations below looks at the parent number, which is 1 below
	// the block number. Thus we remove one from the delay given
	return func(time uint64, parent *types.Header) *big.Int {
		// https://github.com/classzz/EIPs/issues/100.
		// algorithm:
		bigTime := new(big.Int).SetUint64(time)
		bigParentTime := new(big.Int).SetUint64(parent.Time)

		// holds intermediate values to make the algo easier to read & audit
		x := new(big.Int)
		y := new(big.Int)

		// 1 - ((timestamp - parent.timestamp) // 15
		x.Sub(bigTime, bigParentTime)
		x.Div(x, allowedFutureBlockTimeSeconds)
		x.Sub(big1, x)

		// max((1 - (block_timestamp - parent_timestamp) // 15, -99)
		if x.Cmp(bigMinus99) < 0 {
			x.Set(bigMinus99)
		}

		// parent_diff + (parent_diff * max( 1 - ((timestamp - parent.timestamp) // 15), -99) // 1024 )
		y.Mul(parent.Difficulty, x)
		x.Div(y, params.DifficultyBoundDivisor)
		newDifficulty := new(big.Int).Add(parent.Difficulty, x)

		// minimum difficulty can ever be (before exponential factor)
		if newDifficulty.Cmp(params.MinimumDifficulty) < 0 {
			newDifficulty.Set(params.MinimumDifficulty)
		}

		return newDifficulty
	}
}

// Exported for fuzzing
var DynamicDifficultyCalculator = makeDifficultyCalculator

// verifySeal checks whether a block satisfies the PoW difficulty requirements,
// either using the usual ethash cache for it, or alternatively using a full DAG
// to make remote mining fast.
func (ethash *Ethash) verifySeal(chain consensus.ChainHeaderReader, header *types.Header, factor *big.Int) error {
	// If we're running a fake PoW, accept any seal as valid
	if ethash.config.PowMode == ModeFake || ethash.config.PowMode == ModeFullFake {
		time.Sleep(ethash.fakeDelay)
		if ethash.fakeFail == header.Number.Uint64() {
			return errInvalidPoW
		}
		return nil
	}
	// If we're running a shared PoW, delegate verification to it
	if ethash.shared != nil {
		return ethash.shared.verifySeal(chain, header, factor)
	}
	// Ensure that we have a valid difficulty for the block
	if header.Difficulty.Sign() <= 0 {
		return errInvalidDifficulty
	}
	var (
		result []byte
	)
	// If fast-but-heavy PoW verification was requested, use an ethash dataset
	result = HashCZZ(ethash.SealHash(header).Bytes(), header.Nonce.Uint64())

	difficulty := header.Difficulty
	if factor != nil && factor.Sign() > 0 {
		difficulty = new(big.Int).Div(difficulty, factor)
		if difficulty.Cmp(big.NewInt(0)) == 0 {
			difficulty = params.MinimumDifficulty
		}
	}
	target := new(big.Int).Div(two256, difficulty)
	if new(big.Int).SetBytes(result).Cmp(target) > 0 {
		return errInvalidPoW
	}
	return nil
}

// Prepare implements consensus.Engine, initializing the difficulty field of a
// header to conform to the ethash protocol. The changes are done inline.
func (ethash *Ethash) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	header.Difficulty = ethash.CalcDifficulty(chain, header.Time, parent)
	return nil
}

// Finalize implements consensus.Engine, accumulating the block and uncle rewards,
// setting the final state on the header
func (ethash *Ethash) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction) {
	// Accumulate any block and uncle rewards and commit the final state root
	accumulateRewards(chain.Config(), state, header)
	consensus.OnceInitImpawnState(chain.Config(), state)

	if chain.Config().IsCIP4(header.Number) {
		pool1 := state.GetBalance(common.BytesToAddress([]byte{101}))
		state.SubBalance(common.BytesToAddress([]byte{101}), pool1)
		pool2 := state.GetBalance(common.BytesToAddress([]byte{102}))
		state.SubBalance(common.BytesToAddress([]byte{102}), pool2)
		pool3 := state.GetBalance(common.BytesToAddress([]byte{103}))
		state.SubBalance(common.BytesToAddress([]byte{103}), pool3)
		pool4 := state.GetBalance(common.BytesToAddress([]byte{104}))
		state.SubBalance(common.BytesToAddress([]byte{104}), pool4)
		pool5 := state.GetBalance(common.BytesToAddress([]byte{105}))
		state.SubBalance(common.BytesToAddress([]byte{105}), pool5)
		pool6 := state.GetBalance(common.BytesToAddress([]byte{106}))
		state.SubBalance(common.BytesToAddress([]byte{106}), pool6)
		pool7 := state.GetBalance(common.BytesToAddress([]byte{107}))
		state.SubBalance(common.BytesToAddress([]byte{107}), pool7)

		pool_count := big.NewInt(0).Add(pool1, pool2)
		pool_count = big.NewInt(0).Add(pool_count, pool3)
		pool_count = big.NewInt(0).Add(pool_count, pool4)
		pool_count = big.NewInt(0).Add(pool_count, pool5)
		pool_count = big.NewInt(0).Add(pool_count, pool6)
		pool_count = big.NewInt(0).Add(pool_count, pool7)

		state.AddBalance(common.HexToAddress("0x1111111111111111111111111111"), pool_count)

	}

	vm.ShiftItems(state, header.Number.Uint64())
	header.Root = state.IntermediateRoot(true)
}

// FinalizeAndAssemble implements consensus.Engine, accumulating the block and
// uncle rewards, setting the final state and assembling the block.
func (ethash *Ethash) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, receipts []*types.Receipt) (*types.Block, error) {
	// Finalize block
	ethash.Finalize(chain, header, state, txs)

	// Header seems complete, assemble into a block and return
	return types.NewBlock(header, txs, receipts, trie.NewStackTrie(nil)), nil
}

// SealHash returns the hash of a block prior to it being sealed.
func (ethash *Ethash) SealHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()

	rlp.Encode(hasher, []interface{}{
		header.ParentHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra,
	})
	hasher.Sum(hash[:0])
	return hash
}

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
func accumulateRewards(config *params.ChainConfig, state *state.StateDB, header *types.Header) {
	// Skip block reward in catalyst mode
	if config.IsNoReward(header.Number) {
		return
	}
	// Select the correct block reward based on chain progression
	// Equivalent to: baseSubsidy / 2^(height/subsidyHalvingInterval)
	interval := header.Number.Uint64() / SubsidyReductionInterval.Uint64()
	reward := new(big.Int).Rsh(BlockReward, uint(interval))
	// Accumulate the rewards for the miner
	state.AddBalance(header.Coinbase, reward)
}
