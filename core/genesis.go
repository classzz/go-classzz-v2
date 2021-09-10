// Copyright 2014 The go-classzz-v2 Authors
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

package core

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/classzz/go-classzz-v2/consensus"
	"github.com/classzz/go-classzz-v2/rpc"
	"math/big"
	"strings"

	"github.com/classzz/go-classzz-v2/common"
	"github.com/classzz/go-classzz-v2/common/hexutil"
	"github.com/classzz/go-classzz-v2/common/math"
	"github.com/classzz/go-classzz-v2/core/rawdb"
	"github.com/classzz/go-classzz-v2/core/state"
	"github.com/classzz/go-classzz-v2/core/types"
	"github.com/classzz/go-classzz-v2/core/vm"
	"github.com/classzz/go-classzz-v2/crypto"
	"github.com/classzz/go-classzz-v2/czzdb"
	"github.com/classzz/go-classzz-v2/log"
	"github.com/classzz/go-classzz-v2/params"
	"github.com/classzz/go-classzz-v2/rlp"
	"github.com/classzz/go-classzz-v2/trie"
)

//go:generate gencodec -type Genesis -field-override genesisSpecMarshaling -out gen_genesis.go
//go:generate gencodec -type GenesisAccount -field-override genesisAccountMarshaling -out gen_genesis_account.go

var errGenesisNoConfig = errors.New("genesis has no chain configuration")

type StakeMember struct {
	Coinbase        common.Address `json:"coinbase`
	StakeBase       common.Address `json:"stakebase`
	Pubkey          []byte
	Amount          *big.Int
	CoinBaseAddress []common.Address
}

// Genesis specifies the header fields, state of a genesis block. It also defines hard
// fork switch-over blocks through the chain configuration.
type Genesis struct {
	Config     *params.ChainConfig `json:"config"`
	Nonce      uint64              `json:"nonce"`
	Timestamp  uint64              `json:"timestamp"`
	ExtraData  []byte              `json:"extraData"`
	GasLimit   uint64              `json:"gasLimit"   gencodec:"required"`
	Difficulty *big.Int            `json:"difficulty" gencodec:"required"`
	Mixhash    common.Hash         `json:"mixHash"`
	Coinbase   common.Address      `json:"coinbase"`
	Alloc      GenesisAlloc        `json:"alloc"      gencodec:"required"`

	Committee []*StakeMember `json:"stake"      gencodec:"required"`
	// These fields are used for consensus tests. Please don't use them
	// in actual genesis blocks.
	Number     uint64      `json:"number"`
	GasUsed    uint64      `json:"gasUsed"`
	ParentHash common.Hash `json:"parentHash"`
	BaseFee    *big.Int    `json:"baseFeePerGas"`
}

// GenesisAlloc specifies the initial state that is part of the genesis block.
type GenesisAlloc map[common.Address]GenesisAccount

func (ga *GenesisAlloc) UnmarshalJSON(data []byte) error {
	m := make(map[common.UnprefixedAddress]GenesisAccount)
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*ga = make(GenesisAlloc)
	for addr, a := range m {
		(*ga)[common.Address(addr)] = a
	}
	return nil
}

// GenesisAccount is an account in the state of the genesis block.
type GenesisAccount struct {
	Code       []byte                      `json:"code,omitempty"`
	Storage    map[common.Hash]common.Hash `json:"storage,omitempty"`
	Balance    *big.Int                    `json:"balance" gencodec:"required"`
	Nonce      uint64                      `json:"nonce,omitempty"`
	PrivateKey []byte                      `json:"secretKey,omitempty"` // for tests
}

// field type overrides for gencodec
type genesisSpecMarshaling struct {
	Nonce      math.HexOrDecimal64
	Timestamp  math.HexOrDecimal64
	ExtraData  hexutil.Bytes
	GasLimit   math.HexOrDecimal64
	GasUsed    math.HexOrDecimal64
	Number     math.HexOrDecimal64
	Difficulty *math.HexOrDecimal256
	BaseFee    *math.HexOrDecimal256
	Alloc      map[common.UnprefixedAddress]GenesisAccount
}

type genesisAccountMarshaling struct {
	Code       hexutil.Bytes
	Balance    *math.HexOrDecimal256
	Nonce      math.HexOrDecimal64
	Storage    map[storageJSON]storageJSON
	PrivateKey hexutil.Bytes
}

// storageJSON represents a 256 bit byte array, but allows less than 256 bits when
// unmarshaling from hex.
type storageJSON common.Hash

func (h *storageJSON) UnmarshalText(text []byte) error {
	text = bytes.TrimPrefix(text, []byte("0x"))
	if len(text) > 64 {
		return fmt.Errorf("too many hex characters in storage key/value %q", text)
	}
	offset := len(h) - len(text)/2 // pad on the left
	if _, err := hex.Decode(h[offset:], text); err != nil {
		fmt.Println(err)
		return fmt.Errorf("invalid hex storage key/value %q", text)
	}
	return nil
}

func (h storageJSON) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// GenesisMismatchError is raised when trying to overwrite an existing
// genesis block with an incompatible one.
type GenesisMismatchError struct {
	Stored, New common.Hash
}

func (e *GenesisMismatchError) Error() string {
	return fmt.Sprintf("database contains incompatible genesis (have %x, new %x)", e.Stored, e.New)
}

// SetupGenesisBlock writes or updates the genesis block in db.
// The block that will be used is:
//
//                          genesis == nil       genesis != nil
//                       +------------------------------------------
//     db has no genesis |  main-net default  |  genesis
//     db has genesis    |  from DB           |  genesis (if compatible)
//
// The stored chain configuration will be updated if it is compatible (i.e. does not
// specify a fork block below the local head block). In case of a conflict, the
// error is a *params.ConfigCompatError and the new, unwritten config is returned.
//
// The returned chain configuration is never nil.
func SetupGenesisBlock(db czzdb.Database, genesis *Genesis) (*params.ChainConfig, common.Hash, error) {
	return SetupGenesisBlockWithOverride(db, genesis, nil)
}

func SetupGenesisBlockWithOverride(db czzdb.Database, genesis *Genesis, overrideLondon *big.Int) (*params.ChainConfig, common.Hash, error) {
	if genesis != nil && genesis.Config == nil {
		return params.AllEthashProtocolChanges, common.Hash{}, errGenesisNoConfig
	}
	// Just commit the new block if there is no stored genesis block.
	stored := rawdb.ReadCanonicalHash(db, 0)
	if (stored == common.Hash{}) {
		if genesis == nil {
			log.Info("Writing default main-net genesis block")
			genesis = DefaultGenesisBlock()
		} else {
			log.Info("Writing custom genesis block")
		}
		block, err := genesis.Commit(db)
		if err != nil {
			return genesis.Config, common.Hash{}, err
		}
		return genesis.Config, block.Hash(), nil
	}
	// We have the genesis block in database(perhaps in ancient database)
	// but the corresponding state is missing.
	header := rawdb.ReadHeader(db, stored, 0)
	if _, err := state.New(header.Root, state.NewDatabaseWithConfig(db, nil), nil); err != nil {
		if genesis == nil {
			genesis = DefaultGenesisBlock()
		}
		// Ensure the stored genesis matches with the given one.
		hash := genesis.ToBlock(nil).Hash()
		if hash != stored {
			return genesis.Config, hash, &GenesisMismatchError{stored, hash}
		}
		block, err := genesis.Commit(db)
		if err != nil {
			return genesis.Config, hash, err
		}
		return genesis.Config, block.Hash(), nil
	}
	// Check whether the genesis block is already written.
	if genesis != nil {
		hash := genesis.ToBlock(nil).Hash()
		if hash != stored {
			return genesis.Config, hash, &GenesisMismatchError{stored, hash}
		}
	}
	// Get the existing chain configuration.
	newcfg := genesis.configOrDefault(stored)
	//if overrideLondon != nil {
	//	newcfg.LondonBlock = overrideLondon
	//}
	//if err := newcfg.CheckConfigForkOrder(); err != nil {
	//	return newcfg, common.Hash{}, err
	//}
	storedcfg := rawdb.ReadChainConfig(db, stored)
	if storedcfg == nil {
		log.Warn("Found genesis block without chain config")
		rawdb.WriteChainConfig(db, stored, newcfg)
		return newcfg, stored, nil
	}
	// Special case: don't change the existing config of a non-mainnet chain if no new
	// config is supplied. These chains would get AllProtocolChanges (and a compat error)
	// if we just continued here.
	if genesis == nil && stored != params.MainnetGenesisHash {
		return storedcfg, stored, nil
	}
	// Check config compatibility and write the config. Compatibility errors
	// are returned to the caller unless we're already at block zero.
	height := rawdb.ReadHeaderNumber(db, rawdb.ReadHeadHeaderHash(db))
	if height == nil {
		return newcfg, stored, fmt.Errorf("missing block number for head header hash")
	}
	compatErr := storedcfg.CheckCompatible(newcfg, *height)
	if compatErr != nil && *height != 0 && compatErr.RewindTo != 0 {
		return newcfg, stored, compatErr
	}
	rawdb.WriteChainConfig(db, stored, newcfg)
	return newcfg, stored, nil
}

func (g *Genesis) configOrDefault(ghash common.Hash) *params.ChainConfig {
	switch {
	case g != nil:
		return g.Config
	case ghash == params.MainnetGenesisHash:
		return params.MainnetChainConfig
	case ghash == params.TestnetGenesisHash:
		return params.TestnetChainConfig
	default:
		return params.AllEthashProtocolChanges
	}
}

// ToBlock creates the genesis block and writes state of a genesis specification
// to the given database (or discards it if nil).
func (g *Genesis) ToBlock(db czzdb.Database) *types.Block {
	if db == nil {
		db = rawdb.NewMemoryDatabase()
	}
	statedb, err := state.New(common.Hash{}, state.NewDatabase(db), nil)
	if err != nil {
		panic(err)
	}
	for addr, account := range g.Alloc {
		statedb.AddBalance(addr, account.Balance)
		statedb.SetCode(addr, account.Code)
		statedb.SetNonce(addr, account.Nonce)
		for key, value := range account.Storage {
			statedb.SetState(addr, key, value)
		}
	}
	consensus.OnceInitImpawnState(g.Config, statedb)
	impl := vm.NewTeWakaImpl()
	for _, member := range g.Committee {
		impl.Mortgage(member.Coinbase, member.StakeBase, member.Pubkey, member.Amount, member.CoinBaseAddress)
		vm.GenesisLockedBalance(statedb, member.Coinbase, member.StakeBase, member.Amount)
	}
	err = impl.Save(statedb, vm.TeWaKaAddress)
	if err != nil {
		log.Error("ToFastBlock IMPL Save", "error", err)
	}
	root := statedb.IntermediateRoot(false)
	head := &types.Header{
		Number:     new(big.Int).SetUint64(g.Number),
		Nonce:      types.EncodeNonce(g.Nonce),
		Time:       g.Timestamp,
		ParentHash: g.ParentHash,
		Extra:      g.ExtraData,
		GasLimit:   g.GasLimit,
		GasUsed:    g.GasUsed,
		BaseFee:    g.BaseFee,
		Difficulty: g.Difficulty,
		Coinbase:   g.Coinbase,
		Root:       root,
	}
	if g.GasLimit == 0 {
		head.GasLimit = params.GenesisGasLimit
	}
	if g.Difficulty == nil {
		head.Difficulty = params.GenesisDifficulty
	}
	if g.BaseFee != nil {
		head.BaseFee = g.BaseFee
	} else {
		head.BaseFee = new(big.Int).SetUint64(params.InitialBaseFee)
	}
	statedb.Commit(false)
	statedb.Database().TrieDB().Commit(root, true, nil)

	return types.NewBlock(head, nil, nil, trie.NewStackTrie(nil))
}

// Commit writes the block and state of a genesis specification to the database.
// The block is committed as the canonical head block.
func (g *Genesis) Commit(db czzdb.Database) (*types.Block, error) {
	block := g.ToBlock(db)
	if block.Number().Sign() != 0 {
		return nil, fmt.Errorf("can't commit genesis block with number > 0")
	}
	config := g.Config
	if config == nil {
		config = params.AllEthashProtocolChanges
	}
	rawdb.WriteTd(db, block.Hash(), block.NumberU64(), g.Difficulty)
	rawdb.WriteBlock(db, block)
	rawdb.WriteReceipts(db, block.Hash(), block.NumberU64(), nil)
	rawdb.WriteCanonicalHash(db, block.Hash(), block.NumberU64())
	rawdb.WriteHeadBlockHash(db, block.Hash())
	rawdb.WriteHeadFastBlockHash(db, block.Hash())
	rawdb.WriteHeadHeaderHash(db, block.Hash())
	rawdb.WriteChainConfig(db, block.Hash(), config)
	return block, nil
}

// MustCommit writes the genesis block and state to db, panicking on error.
// The block is committed as the canonical head block.
func (g *Genesis) MustCommit(db czzdb.Database) *types.Block {
	block, err := g.Commit(db)
	if err != nil {
		panic(err)
	}
	return block
}

// GenesisBlockForTesting creates and writes a block in which addr has the given wei balance.
func GenesisBlockForTesting(db czzdb.Database, addr common.Address, balance *big.Int) *types.Block {
	g := Genesis{
		Alloc:   GenesisAlloc{addr: {Balance: balance}},
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
	return g.MustCommit(db)
}

// DefaultGenesisBlock returns the Classzz main net genesis block.
func DefaultGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.MainnetChainConfig,
		Nonce:      66,
		ExtraData:  hexutil.MustDecode("0x4275696c642061206272696467652c20796f7520636f6e6e65637420322069736c616e64733b206275696c6420612063616e6f6520796f7520636f6e6e65637420616c6c2069736c616e64732e2054652057616b6120697320746861742063616e6f652c207468617420636f6e6e6563747320616c6c20626c6f636b636861696e732e"),
		GasLimit:   5000,
		Difficulty: new(big.Int).SetUint64(1_000_000_000_000),
		Alloc:      decodePrealloc(mainnetAllocData),
		Committee: []*StakeMember{
			{
				Coinbase:  common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
				StakeBase: common.BytesToAddress([]byte{1, 1}),
				Pubkey:    common.Hex2Bytes("03fe89d1c232d3e1e39790803f3998ec9abb8af2175221347c12c151e5e7f932f0"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0x12684D85b7885DE85EBc6398bBcEfa08274134A0"),
				StakeBase: common.BytesToAddress([]byte{1, 2}),
				Pubkey:    common.Hex2Bytes("02c2552119fda685f2f6da437078225b3fef14bc61cc3ccf758702c23f4611c655"),
				Amount:    new(big.Int).Mul(big.NewInt(2000000), big.NewInt(1e18)),
			},
			{
				Coinbase:  common.HexToAddress("0xC11f364698f5eeB18Bc82b30Fc2b84847177fD23"),
				StakeBase: common.BytesToAddress([]byte{1, 3}),
				Pubkey:    common.Hex2Bytes("033e63e6ab0e433ed6227cc19b13fdbf9bf436268e8129cec96783c62e18c35415"),
				Amount:    new(big.Int).Mul(big.NewInt(6000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0x1568120C04367518B3e878085667625617383828"),
				StakeBase: common.BytesToAddress([]byte{1, 4}),
				Pubkey:    common.Hex2Bytes("03833f0837af9c04d00b67bfbf2fef55314d039c4d5ed1d189567b09dba28bcd0d"),
				Amount:    new(big.Int).Mul(big.NewInt(7000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0x1B46857eE2748e09a48F499001Cb16a9B0d7a61e"),
				StakeBase: common.BytesToAddress([]byte{1, 5}),
				Pubkey:    common.Hex2Bytes("036288fbb681212270ffb1f128f592a88f2ee90eae9103b8b847fa5e32bfbea2ee"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0x7D1C8B109737D12E49523baafF92462aE29FF833"),
				StakeBase: common.BytesToAddress([]byte{1, 6}),
				Pubkey:    common.Hex2Bytes("03273b69ad364c4647f19f5cb37f9c8300cd698596e0e9df52fd51d94db52975ea"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0x41dA56830548eE25601B488fdE310DB8A9436362"),
				StakeBase: common.BytesToAddress([]byte{1, 7}),
				Pubkey:    common.Hex2Bytes("03284e410b1a510b4bf55254d02d18a352941ebbd30fe8b6a6036399b5575c06d1"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0xedD509361ED3C4b1e789fEAd2a1052E7F675C2e8"),
				StakeBase: common.BytesToAddress([]byte{1, 8}),
				Pubkey:    common.Hex2Bytes("03ce17baa5176825fd7febe50ec1e6cd3a26526c6ddd2e7eed9c93c50ef9e11e2c"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0x168A4beFc9deac7744F3C8d4C19C341c6F141553"),
				StakeBase: common.BytesToAddress([]byte{1, 9}),
				Pubkey:    common.Hex2Bytes("02542050fb9b585795c88edc49e48000080457fd5ea077974b0b478b498d80d8c8"),
				Amount:    new(big.Int).Mul(big.NewInt(2000000), big.NewInt(1e18)),
			},
			{
				Coinbase:  common.HexToAddress("0xe50a511a60e40Fe14A77fcC44c586bA7b60C7d5E"),
				StakeBase: common.BytesToAddress([]byte{1, 10}),
				Pubkey:    common.Hex2Bytes("03cb35ea1bff93d40913062c9e2cdc9f1c444a702ef338571f47faed967d73fcc8"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0xD96f04cb4a49d465294d47A56d59a6B164C6A5E0"),
				StakeBase: common.BytesToAddress([]byte{1, 11}),
				Pubkey:    common.Hex2Bytes("02394fbbe279189bba851c6aba15dbbc15d375f58e7044a9081c7c6e7980d2f611"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0xb2286479525505Fcc5Ef3a1d1407EF96bc0Ac745"),
				StakeBase: common.BytesToAddress([]byte{1, 12}),
				Pubkey:    common.Hex2Bytes("02394fbbe279189bba851c6aba15dbbc15d375f58e7044a9081c7c6e7980d2f611"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0xB03E6687a5D0a27C7661C03aB0F88836be1b468B"),
				StakeBase: common.BytesToAddress([]byte{1, 13}),
				Pubkey:    common.Hex2Bytes("02239ffaa617f9d5f3ad8945b50740caa986928f5acd495ac9c4890e8df624236a"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0xf3237AC2BD73185b508CAA4B90c5febB6338170f"),
				StakeBase: common.BytesToAddress([]byte{1, 14}),
				Pubkey:    common.Hex2Bytes("03d4319f9db08461c17aa837fe320ec7910bce26f61d71f23777c44449cc13ee96"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0x2545C05095709505DE20dc6074BDCf90F7Fd5A95"),
				StakeBase: common.BytesToAddress([]byte{1, 15}),
				Pubkey:    common.Hex2Bytes("0285329a00e38cc07e0aecae13d173f404680868bcc3408f81077bb4822bd7f81f"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0x351659b14c6700aa5C0360E08c94Ba6395496aec"),
				StakeBase: common.BytesToAddress([]byte{1, 16}),
				Pubkey:    common.Hex2Bytes("0261cc730c8ffe3b9b21582f4581f172158d8351ec0186b20b22f1165350e286ec"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0x09e0874c6399d3cC739D879206Ee5D4140212EE5"),
				StakeBase: common.BytesToAddress([]byte{1, 17}),
				Pubkey:    common.Hex2Bytes("038f0dccb50af3a53f871de937e7a2cad88bd0fa41c9ddfd426254e354ddcdbc5a"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0x996667572e76849b0CA0A8912227E063F613E505"),
				StakeBase: common.BytesToAddress([]byte{1, 18}),
				Pubkey:    common.Hex2Bytes("021cacb67dcb5dcf9c26c89a9b2dfacebc406ca083a0b690838fbc6308f77fdf5a"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0x4BD8Ab37688589eC75cd39eBb7d52ec93CC59CCa"),
				StakeBase: common.BytesToAddress([]byte{1, 19}),
				Pubkey:    common.Hex2Bytes("02807798f6f9ac77611a2efff6f629d51219429e923375be34781f463525d56df2"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0x596Af95f4378AD3866e0FA1dF0C04C0290dc4F34"),
				StakeBase: common.BytesToAddress([]byte{1, 20}),
				Pubkey:    common.Hex2Bytes("03684cd38bf36922cd37489cb8e19864ef56de8e0175998fc09bebf0a640e89037"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0x1491F7883f4220Cb46DE3804F83A294e6af624F2"),
				StakeBase: common.BytesToAddress([]byte{1, 21}),
				Pubkey:    common.Hex2Bytes("0318e25c9d2b6301ab36391335b280d15e8ed310885e4f3bc9020db5a36e49081b"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
			{
				Coinbase:  common.HexToAddress("0xF59c92820a62c174419d66bc0Ec3cE528939dC9d"),
				StakeBase: common.BytesToAddress([]byte{1, 22}),
				Pubkey:    common.Hex2Bytes("02fe7bd7e666382dc9f8d243c3013ef29d5ddbd6ee453fe454b7aa82c1436416f9"),
				Amount:    new(big.Int).Mul(big.NewInt(5000000), big.NewInt(1e18)),
				CoinBaseAddress: []common.Address{
					common.HexToAddress("0x5CEED59b15E4566A01020989Ca848DdA69B92b9F"),
					common.HexToAddress("0x2faeb19FABC6aB7C174B2E698A2f21dCe5Bd595A"),
					common.HexToAddress("0x5884d78105EeB71F6b844b0393d816e0982672cc"),
					common.HexToAddress("0xFd4B92f7Eb6352e87290B9Dbe3d0DAfca1BFADd6"),
				},
			},
		},
	}
}

func DefaultTestnetGenesisBlock() *Genesis {
	i2 := new(big.Int).Mul(big.NewInt(100000), big.NewInt(1e18))
	// addr1: 0xF59039fdA7dBC14F050BFeF36C75F5fD3D3eb23B
	key1 := hexutil.MustDecode("0x04e76d4d749766a5682f2b88bd0c4633fd2afc1ae183cb21203b321210271de6c498197f32e873586c7a8c32fb5606279466002bb09e99bed225bbe231312ac8e2")
	// addr2: 0xCBbf6dA3b3809A3AD0140d9FBd3b91Eb7EafFC31
	key2 := hexutil.MustDecode("0x0470aaf7409e2ff3ef0b3c776f103eb1761ac2949ad8750de10ce9f1b4b497666552542601f0e160b98d2f9057eb3b3e4755c0ed2872b0ad6e6f36b3685953eb1f")
	// addr3: 0xC85eF13F14f807954cA22bdA4919e06c838A079e
	key3 := hexutil.MustDecode("0x04bfd74dc8e5a30c1352827a1ac5f2aa528940ef24e3ff91b9ca9932ae0a63dad094ab8c833aeda5a306324cd0b6bfda298ca9b4ec568a8817724bcb8189ffd75c")

	return &Genesis{
		Config:     params.TestnetChainConfig,
		Timestamp:  0x61318edd,
		GasLimit:   0x47b760,
		Difficulty: big.NewInt(10240),
		Alloc:      decodePrealloc(mainnetAllocData),
		Committee: []*StakeMember{
			{
				Coinbase:  common.HexToAddress("0xF59039fdA7dBC14F050BFeF36C75F5fD3D3eb23B"),
				StakeBase: common.HexToAddress("0xF59039fdA7dBC14F050BFeF36C75F5fD3D3eb23B"),
				Pubkey:    key1,
				Amount:    new(big.Int).Set(i2),
			},
			{
				Coinbase:  common.HexToAddress("0xCBbf6dA3b3809A3AD0140d9FBd3b91Eb7EafFC31"),
				StakeBase: common.HexToAddress("0xCBbf6dA3b3809A3AD0140d9FBd3b91Eb7EafFC31"),
				Pubkey:    key2,
				Amount:    new(big.Int).Set(i2),
			},
			{
				Coinbase:  common.HexToAddress("0xC85eF13F14f807954cA22bdA4919e06c838A079e"),
				StakeBase: common.HexToAddress("0xC85eF13F14f807954cA22bdA4919e06c838A079e"),
				Pubkey:    key3,
				Amount:    new(big.Int).Set(i2),
			},
		},
	}
}

// DeveloperGenesisBlock returns the 'gczz --dev' genesis block.
func DeveloperGenesisBlock(period uint64, faucet common.Address) *Genesis {
	// Override the default period to the user requested one
	config := *params.AllCliqueProtocolChanges
	config.Clique = &params.CliqueConfig{
		Period: period,
		Epoch:  config.Clique.Epoch,
	}

	// Assemble and return the genesis with the precompiles and faucet pre-funded
	return &Genesis{
		Config:     &config,
		ExtraData:  append(append(make([]byte, 32), faucet[:]...), make([]byte, crypto.SignatureLength)...),
		GasLimit:   11500000,
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Difficulty: big.NewInt(1),
		Alloc: map[common.Address]GenesisAccount{
			common.BytesToAddress([]byte{1}): {Balance: big.NewInt(1)}, // ECRecover
			common.BytesToAddress([]byte{2}): {Balance: big.NewInt(1)}, // SHA256
			common.BytesToAddress([]byte{3}): {Balance: big.NewInt(1)}, // RIPEMD
			common.BytesToAddress([]byte{4}): {Balance: big.NewInt(1)}, // Identity
			common.BytesToAddress([]byte{5}): {Balance: big.NewInt(1)}, // ModExp
			common.BytesToAddress([]byte{6}): {Balance: big.NewInt(1)}, // ECAdd
			common.BytesToAddress([]byte{7}): {Balance: big.NewInt(1)}, // ECScalarMul
			common.BytesToAddress([]byte{8}): {Balance: big.NewInt(1)}, // ECPairing
			common.BytesToAddress([]byte{9}): {Balance: big.NewInt(1)}, // BLAKE2b
			faucet:                           {Balance: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9))},
		},
	}
}

func decodePrealloc(data string) GenesisAlloc {
	var p []struct{ Addr, Balance *big.Int }
	if err := rlp.NewStream(strings.NewReader(data), 0).Decode(&p); err != nil {
		panic(err)
	}
	ga := make(GenesisAlloc, len(p))
	for _, account := range p {
		ga[common.BigToAddress(account.Addr)] = GenesisAccount{Balance: account.Balance}
	}
	return ga
}

func CommitClient(EthClient, HecoClient, BscClient, OkexClient []string, param *params.ChainConfig) (*params.ChainConfig, error) {

	for _, v := range EthClient {

		if v[:4] != "http" {
			v = "http://" + v
		}
		client, err := rpc.Dial(v)
		if err != nil {
			log.Warn("rpc failed", "url", v, "err", err)
			return nil, err
		}

		var number hexutil.Uint64
		if err := client.Call(&number, "eth_blockNumber"); err != nil {
			log.Warn("rpc failed", "url", v, "err", err)
			return nil, err
		}
		log.Info("eth rpc successed", "url", v, "block", number)
		param.EthClient = append(param.EthClient, client)
	}

	for _, v := range HecoClient {

		if v[:4] != "http" {
			v = "http://" + v
		}
		client, err := rpc.Dial(v)
		if err != nil {
			log.Warn("rpc failed", "url", v, "err", err)
			return nil, err
		}

		var number hexutil.Uint64
		if err := client.Call(&number, "eth_blockNumber"); err != nil {
			log.Warn("rpc failed", "url", v, "err", err)
			return nil, err
		}
		log.Info("heco rpc successed", "url", v, "block", number)
		param.HecoClient = append(param.HecoClient, client)
	}

	for _, v := range BscClient {

		if v[:4] != "http" {
			v = "http://" + v
		}
		client, err := rpc.Dial(v)
		if err != nil {
			log.Warn("rpc failed", "url", v, "err", err)
			return nil, err
		}

		var number hexutil.Uint64
		if err := client.Call(&number, "eth_blockNumber"); err != nil {
			log.Warn("rpc failed", "url", v, "err", err)
			return nil, err
		}
		log.Info("bsc rpc successed", "url", v, "block", number)
		param.BscClient = append(param.BscClient, client)
	}

	for _, v := range OkexClient {

		if v[:4] != "http" {
			v = "http://" + v
		}
		client, err := rpc.Dial(v)
		if err != nil {
			log.Warn("rpc failed", "url", v, "err", err)
			return nil, err
		}

		var number hexutil.Uint64
		if err := client.Call(&number, "eth_blockNumber"); err != nil {
			log.Warn("rpc failed", "url", v, "err", err)
			return nil, err
		}
		log.Info("okex rpc successed", "url", v, "block", number)
		param.OkexClient = append(param.OkexClient, client)
	}

	return param, nil
}
