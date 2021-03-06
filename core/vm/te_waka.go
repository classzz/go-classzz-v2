// Copyright 2016 The go-classzz-v2 Authors
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

package vm

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/classzz/go-classzz-v2/crypto"
	"math/big"
	"strings"
	"time"

	"github.com/classzz/go-classzz-v2/accounts/abi"
	"github.com/classzz/go-classzz-v2/common"
	"github.com/classzz/go-classzz-v2/core/types"
	"github.com/classzz/go-classzz-v2/log"
	"github.com/classzz/oectx-ethtx/utils"
)

const (
	// Entangle Transcation type
	ExpandedTxConvert_Czz uint8 = iota
	ExpandedTxConvert_ECzz
	ExpandedTxConvert_HCzz
	ExpandedTxConvert_BCzz
	ExpandedTxConvert_OCzz
	ExpandedTxConvert_PCzz
	ExpandedTxConvert_MCzz
	ExpandedTxConvert_GCzz
)

var (
	baseUnit      = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	Int10         = new(big.Int).Exp(big.NewInt(10), big.NewInt(10), nil)
	Int1000       = big.NewInt(1000)
	MortgageToMin = new(big.Int).SetUint64(257)
	MortgageToMax = new(big.Int).SetUint64(356)

	mimStakingAmount = new(big.Int).Mul(big.NewInt(1000000), baseUnit)
	Address0         = common.BytesToAddress([]byte{0})

	// i.e. contractAddress = 0x0000000000000000000000000000746577616b61
	TeWaKaAddress = common.BytesToAddress([]byte("tewaka"))
	CoinPools     = map[uint8]common.Address{
		ExpandedTxConvert_ECzz: common.BytesToAddress([]byte{101}),
		ExpandedTxConvert_HCzz: common.BytesToAddress([]byte{102}),
		ExpandedTxConvert_BCzz: common.BytesToAddress([]byte{103}),
		ExpandedTxConvert_OCzz: common.BytesToAddress([]byte{104}),
		ExpandedTxConvert_PCzz: common.BytesToAddress([]byte{105}),
		ExpandedTxConvert_MCzz: common.BytesToAddress([]byte{106}),
		ExpandedTxConvert_GCzz: common.BytesToAddress([]byte{107}),
	}

	receiptMap     = map[common.Hash]*types.Receipt{}
	transactionMap = map[common.Hash]*types.Transaction{}

	ethPoolAddr     = "0xa9bDC85F01Aa9E7167E26189596f9a9E2cE67215|"
	hecoPoolAddr    = "0x6a1C9835B7b0943908B25C46D8810bCC9Ab57426|0x15e83ebd7F74dDAb5F9baE06f93dF848862B458E"
	bscPoolAddr     = "0xABe6ED40D861ee39Aa8B21a6f8A554fECb0D32a5|0x8c2eE44fC151B89DD4465be9e59627aCcC12A51F"
	oecPoolAddr     = "0x007c98F9f2c70746a64572E67FBCc41a2b8bba18|0x3b8afE66C371E959A33a975384Fb0572c981c812"
	polygonPoolAddr = "0xdf10e0Caa2BBe67f7a1E91A3e6660cC1e34e81B9|"
	metisPoolAddr   = "0x007c98F9f2c70746a64572E67FBCc41a2b8bba18|"
	gatePoolAddr    = "0x503C5C292CD5300E4006c447A46DEab216a54fb2|"

	burnTopics = "0xa4bd93d5396d36bd742684adb6dbe69f45c14792170e66134569c1adf91d1fb9"
	mintTopics = "0xd4b70e0d50bcb13e7654961d68ed7b96f84a2fcc32edde496c210382dc025708"

	crossTopics = "0xa7b2921d83c1ae7d5d671011b33435909f492547e4d69136a3c02820dfcb2b3f"

	ErrRpcErr = errors.New("rpc err")
)

// TeWaKaGas defines all method gas
var TeWaKaGas = map[string]uint64{
	"mortgage": 360000,
	"update":   360000,
	"convert":  2400000,
	"confirm":  2400000,
	"casting":  2400000,
}

// Staking contract ABI
var AbiTeWaKa abi.ABI
var AbiTeWakaV3 abi.ABI
var AbiCzzRouter abi.ABI

type StakeContract struct{}

func init() {
	AbiTeWaKa, _ = abi.JSON(strings.NewReader(TeWakaABI))
	AbiTeWakaV3, _ = abi.JSON(strings.NewReader(TeWakaV3ABI))
	AbiCzzRouter, _ = abi.JSON(strings.NewReader(CzzRouterABI))

	var p types.Receipts
	json.Unmarshal([]byte(receipts), &p)
	for _, v := range p {
		receiptMap[v.TxHash] = v
	}
	var p2 types.Transactions
	json.Unmarshal([]byte(transactions), &p2)
	for _, v := range p2 {
		transactionMap[v.Hash()] = v
		oecH, _ := utils.Hash(v)
		transactionMap[oecH] = v
	}
}

// RunStaking execute staking contract
func RunStaking(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {

	method, err := AbiTeWaKa.MethodById(input)
	if err != nil {
		log.Error("No method found")
		return nil, ErrExecutionReverted
	}

	data := input[4:]
	isCip4 := !evm.chainConfig.IsCIP4(evm.Context.BlockNumber)

	switch method.Name {
	case "mortgage":
		ret, err = mortgage(evm, contract, data)
	case "update":
		ret, err = update(evm, contract, data)
	case "convert":
		if isCip4 {
			ret, err = convert(evm, contract, data)
		} else {
			err = errors.New("isCip4 close func")
		}
	case "confirm":
		if isCip4 {
			ret, err = confirm(evm, contract, data)
		} else {
			err = errors.New("isCip4 close func")
		}
	case "casting":
		if isCip4 {
			ret, err = casting(evm, contract, data)
		} else {
			err = errors.New("isCip4 close func")
		}
	case "crossToMainChainMap":
		ret, err = crossToMainChainMap(evm, contract, data)
	case "betweenSideChainCrossMap":
		ret, err = betweenSideChainCrossMap(evm, contract, data)
	default:
		log.Debug("Staking call fallback function")
		err = ErrStakingInvalidInput
	}

	if err != nil {
		log.Debug("Staking error code", "method.Name", method.Name, "err", err)
		err = ErrExecutionReverted
	}

	return ret, err
}

// logN add event log to receipt with topics up to 4
func logN(evm *EVM, contract *Contract, topics []common.Hash, data []byte) ([]byte, error) {
	evm.StateDB.AddLog(&types.Log{
		Address: contract.Address(),
		Topics:  topics,
		Data:    data,
		// This is a non-consensus field, but assigned here because
		// core/state doesn't know the current block number.
		BlockNumber: evm.Context.BlockNumber.Uint64(),
	})
	return nil, nil
}

func GenesisLockedBalance(db StateDB, from, to common.Address, amount *big.Int) {
	db.SubBalance(from, amount)
	db.AddBalance(to, amount)
}

// mortgage
func mortgage(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	t0 := time.Now()
	args := struct {
		PubKey          []byte
		ToAddress       common.Address
		StakingAmount   *big.Int
		CoinBaseAddress []common.Address
	}{}
	method, _ := AbiTeWaKa.Methods["mortgage"]

	err = method.Inputs.UnpackAtomic(&args, input)
	if err != nil {
		log.Error("Unpack deposit pubkey error", "err", err)
		return nil, ErrStakingInvalidInput
	}

	from := contract.caller.Address()

	t1 := time.Now()
	tewaka := NewTeWakaImpl()
	err = tewaka.Load(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking load error", "error", err)
		return nil, err
	}

	//
	if args.StakingAmount.Cmp(mimStakingAmount) < 0 {
		return nil, fmt.Errorf("mortgage StakingAmount %s", "StakingAmount <  emimState")
	}

	//
	if ValidPubkey(args.PubKey) != nil {
		return nil, fmt.Errorf("mortgage PubKey %s", "PubKey err")
	}

	//
	ToAddressNum := new(big.Int).SetBytes(args.ToAddress.Bytes())
	if ToAddressNum.Cmp(MortgageToMin) < 0 && ToAddressNum.Cmp(MortgageToMax) > 1 {
		return nil, fmt.Errorf("mortgage ToAddressNum %s", "MortgageToMax > ToAddressNum > MortgageToMin")
	}

	//
	if tewaka.GetStakeUser(from) != nil {
		return nil, fmt.Errorf("mortgage HasStakeUser %s", "from already exist")
	}

	//
	if tewaka.GetStakeToAddress(args.ToAddress) != nil {
		return nil, fmt.Errorf("mortgage HasStakeToAddress %s", "ToAddress already exist")
	}

	//
	if len(args.CoinBaseAddress) > 16 {
		return nil, fmt.Errorf("mortgage CoinBaseAddress %s", "len(CoinBaseAddress) > 16")
	}

	t2 := time.Now()

	tewaka.Mortgage(from, args.ToAddress, args.PubKey, args.StakingAmount, args.CoinBaseAddress)

	t3 := time.Now()
	err = tewaka.Save(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking save state error", "error", err)
		return nil, err
	}

	if have, want := evm.StateDB.GetBalance(from), args.StakingAmount; have.Cmp(want) < 0 {
		return nil, fmt.Errorf("%w: address %v have %v want %v", errors.New("insufficient funds for gas * price + value"), from, have, want)
	}

	evm.StateDB.SubBalance(from, args.StakingAmount)
	evm.StateDB.AddBalance(args.ToAddress, args.StakingAmount)

	t4 := time.Now()
	event := AbiTeWaKa.Events["mortgage"]
	logData, err := event.Inputs.Pack(args.PubKey, args.ToAddress, args.StakingAmount, args.CoinBaseAddress)
	if err != nil {
		log.Error("Pack staking log error", "error", err)
		return nil, err
	}
	topics := []common.Hash{
		event.ID,
		common.BytesToHash(from[:]),
	}
	logN(evm, contract, topics, logData)
	context := []interface{}{
		"number", evm.Context.BlockNumber.Uint64(), "address", from, "StakingAmount", args.StakingAmount,
		"input", common.PrettyDuration(t1.Sub(t0)), "load", common.PrettyDuration(t2.Sub(t1)),
		"insert", common.PrettyDuration(t3.Sub(t2)), "save", common.PrettyDuration(t4.Sub(t3)),
		"log", common.PrettyDuration(time.Since(t4)), "elapsed", common.PrettyDuration(time.Since(t0)),
	}
	log.Debug("mortgage", context...)
	return nil, nil
}

// Update
func update(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	t0 := time.Now()
	args := struct {
		StakingAmount   *big.Int
		CoinBaseAddress []common.Address
	}{}

	method, _ := AbiTeWaKa.Methods["update"]
	err = method.Inputs.UnpackAtomic(&args, input)
	if err != nil {
		log.Error("Unpack deposit pubkey error", "err", err)
		return nil, ErrStakingInvalidInput
	}

	from := contract.caller.Address()
	t1 := time.Now()

	tewaka := NewTeWakaImpl()
	err = tewaka.Load(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking load error", "error", err)
		return nil, err
	}

	var item *types.Pledge
	if item = tewaka.GetStakeUser(from); item == nil {
		return nil, fmt.Errorf("update GetStakeUser %s", "from is nil")
	}

	if args.StakingAmount.Cmp(big.NewInt(0)) > 0 {
		//
		if args.StakingAmount.Cmp(mimStakingAmount) < 0 {
			return nil, fmt.Errorf("update StakingAmount %s", "StakingAmount <  emimState")
		}

		if have, want := evm.StateDB.GetBalance(from), args.StakingAmount; have.Cmp(want) < 0 {
			return nil, fmt.Errorf("%w: address %v have %v want %v", errors.New("insufficient funds for gas * price + value"), from, have, want)
		}

		evm.StateDB.SubBalance(from, args.StakingAmount)
		evm.StateDB.AddBalance(item.ToAddress, args.StakingAmount)
	}

	//
	if len(args.CoinBaseAddress) > 16 {
		log.Error("CoinBaseAddress > 16 ", "error", err)
		return nil, err
	}

	t2 := time.Now()
	temp := tewaka.Update(from, args.CoinBaseAddress)

	if !temp {
		log.Error("from err", "error", err)
		return nil, err
	}

	t3 := time.Now()
	err = tewaka.Save(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking save state error", "error", err)
		return nil, err
	}

	t4 := time.Now()
	event := AbiTeWaKa.Events["update"]
	logData, err := event.Inputs.Pack(args.StakingAmount, args.CoinBaseAddress)
	if err != nil {
		log.Error("Pack staking log error", "error", err)
		return nil, err
	}
	topics := []common.Hash{
		event.ID,
		common.BytesToHash(from[:]),
	}
	logN(evm, contract, topics, logData)
	context := []interface{}{
		"number", evm.Context.BlockNumber.Uint64(), "address", from, "CoinBaseAddress", args.CoinBaseAddress,
		"input", common.PrettyDuration(t1.Sub(t0)), "load", common.PrettyDuration(t2.Sub(t1)),
		"insert", common.PrettyDuration(t3.Sub(t2)), "save", common.PrettyDuration(t4.Sub(t3)),
		"log", common.PrettyDuration(time.Since(t4)), "elapsed", common.PrettyDuration(time.Since(t0)),
	}
	log.Debug("update", context...)
	return nil, nil
}

// Convert
func convert(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	t0 := time.Now()
	args := struct {
		AssetType *big.Int
		TxHash    string
	}{}

	method, _ := AbiTeWaKa.Methods["convert"]
	err = method.Inputs.UnpackAtomic(&args, input)
	if err != nil {
		log.Error("Unpack convert pubkey error", "err", err)
		return nil, ErrStakingInvalidInput
	}

	TxHash := common.HexToHash(args.TxHash)
	from := contract.caller.Address()
	t1 := time.Now()

	tewaka := NewTeWakaImpl()
	err = tewaka.Load(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking load error", "error", err)
		return nil, err
	}

	var item *types.ConvertItem
	AssetType := uint8(args.AssetType.Uint64())

	if exit := tewaka.HasItem(&types.UsedItem{AssetType, TxHash}, evm.StateDB); exit {
		return nil, ErrTxhashAlreadyInput
	}

	if item, err = verifyConvertEthereumTypeTx("Side", evm, AssetType, TxHash); err == ErrRpcErr {
		return nil, err
	}

	Amount := new(big.Int).Mul(item.Amount, Int10)
	FeeAmount := big.NewInt(0).Div(Amount, big.NewInt(1000))
	item.FeeAmount = big.NewInt(0).Div(item.Amount, big.NewInt(1000))
	IDHash := item.Hash()
	item.ID = new(big.Int).SetBytes(IDHash[:10])
	t2 := time.Now()

	if item.ConvertType == ExpandedTxConvert_Czz {

		toaddresspuk, err := crypto.UnmarshalPubkey(item.PubKey)
		if err != nil || toaddresspuk == nil {
			return nil, err
		}
		toaddress := crypto.PubkeyToAddress(*toaddresspuk)

		evm.StateDB.SubBalance(CoinPools[item.AssetType], Amount)
		evm.StateDB.AddBalance(toaddress, new(big.Int).Sub(Amount, FeeAmount))
		evm.StateDB.AddBalance(Address0, FeeAmount)
	} else {
		evm.StateDB.SubBalance(CoinPools[item.AssetType], Amount)
		evm.StateDB.AddBalance(CoinPools[item.ConvertType], new(big.Int).Sub(Amount, FeeAmount))
		evm.StateDB.AddBalance(Address0, FeeAmount)
		tewaka.Convert(item)
	}

	tewaka.SetItem(&types.UsedItem{AssetType, TxHash})

	t3 := time.Now()
	err = tewaka.Save(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking save state error", "error", err)
		return nil, err
	}

	t4 := time.Now()
	event := AbiTeWaKa.Events["convert"]
	logData, err := event.Inputs.Pack(item.ID, args.AssetType, big.NewInt(int64(item.ConvertType)), item.TxHash.String(), item.Path, item.RouterAddr, item.PubKey, item.Amount, item.FeeAmount, item.Slippage, item.IsInsurance, item.Extra)
	if err != nil {
		log.Error("Pack staking log error", "error", err)
		return nil, err
	}
	topics := []common.Hash{
		event.ID,
		common.BytesToHash(from[:]),
	}
	logN(evm, contract, topics, logData)
	context := []interface{}{
		"number", evm.Context.BlockNumber.Uint64(), "address", from, "Amount", item.Amount,
		"AssetType", args.AssetType, "ConvertType", item.ConvertType, "TxHash", args.TxHash,
		"input", common.PrettyDuration(t1.Sub(t0)), "load", common.PrettyDuration(t2.Sub(t1)),
		"insert", common.PrettyDuration(t3.Sub(t2)), "save", common.PrettyDuration(t4.Sub(t3)),
		"log", common.PrettyDuration(time.Since(t4)), "elapsed", common.PrettyDuration(time.Since(t0)),
	}
	log.Debug("convert", context...)

	return nil, nil
}

// Confirm
func confirm(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	t0 := time.Now()
	args := struct {
		ConvertType *big.Int
		TxHash      string
	}{}

	method, _ := AbiTeWaKa.Methods["confirm"]
	err = method.Inputs.UnpackAtomic(&args, input)
	if err != nil {
		log.Error("Unpack convert pubkey error", "err", err)
		return nil, ErrStakingInvalidInput
	}

	TxHash := common.HexToHash(args.TxHash)
	from := contract.caller.Address()
	t1 := time.Now()

	tewaka := NewTeWakaImpl()
	err = tewaka.Load(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking load error", "error", err)
		return nil, err
	}
	var item *types.ConvertItem
	ConvertType := uint8(args.ConvertType.Uint64())

	if exit := tewaka.HasItem(&types.UsedItem{ConvertType, TxHash}, evm.StateDB); exit {
		return nil, ErrTxhashAlreadyInput
	}

	isCip2 := evm.chainConfig.IsCIP2(evm.Context.BlockNumber)

	if item, err = verifyConfirmEthereumTypeTx("Side", tewaka, ConvertType, TxHash, isCip2); err != nil {
		return nil, err
	}

	t2 := time.Now()

	tewaka.Confirm(item)
	tewaka.SetItem(&types.UsedItem{ConvertType, TxHash})

	t3 := time.Now()
	err = tewaka.Save(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking save state error", "error", err)
		return nil, err
	}

	t4 := time.Now()
	event := AbiTeWaKa.Events["confirm"]
	logData, err := event.Inputs.Pack(item.ID, big.NewInt(int64(item.AssetType)), args.ConvertType, args.TxHash)
	if err != nil {
		log.Error("Pack staking log error", "error", err)
		return nil, err
	}
	topics := []common.Hash{
		event.ID,
		common.BytesToHash(from[:]),
	}
	logN(evm, contract, topics, logData)
	context := []interface{}{
		"number", evm.Context.BlockNumber.Uint64(), "address", from, "Amount", item.Amount,
		"AssetType", item.AssetType, "ConvertType", args.ConvertType, "TxHash", args.TxHash,
		"input", common.PrettyDuration(t1.Sub(t0)), "load", common.PrettyDuration(t2.Sub(t1)),
		"insert", common.PrettyDuration(t3.Sub(t2)), "save", common.PrettyDuration(t4.Sub(t3)),
		"log", common.PrettyDuration(time.Since(t4)), "elapsed", common.PrettyDuration(time.Since(t0)),
	}
	log.Debug("confirm", context...)

	return nil, nil
}

// Casting
func casting(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	t0 := time.Now()
	args := struct {
		ConvertType *big.Int
		Amount      *big.Int
		Path        []common.Address
		PubKey      []byte
		RouterAddr  common.Address
		Slippage    *big.Int
		IsInsurance bool
	}{}

	method, _ := AbiTeWaKa.Methods["casting"]
	err = method.Inputs.UnpackAtomic(&args, input)
	if err != nil {
		log.Error("Unpack convert pubkey error", "err", err)
		return nil, ErrStakingInvalidInput
	}

	from := contract.caller.Address()
	t1 := time.Now()

	tewaka := NewTeWakaImpl()
	err = tewaka.Load(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking load error", "error", err)
		return nil, err
	}

	if evm.chainConfig.IsCIP2(evm.Context.BlockNumber) && len(args.PubKey) > 0 {
		toaddresspuk, err := crypto.DecompressPubkey(args.PubKey)
		if err != nil || toaddresspuk == nil {
			toaddresspuk, err = crypto.UnmarshalPubkey(args.PubKey)
			if err != nil || toaddresspuk == nil {
				return nil, fmt.Errorf("toaddresspuk [puk:%s] is err: %s", hex.EncodeToString(args.PubKey), err)
			}
		}
	}

	item := &types.ConvertItem{}
	ConvertType := uint8(args.ConvertType.Uint64())

	item = &types.ConvertItem{
		ConvertType: ConvertType,
		Path:        args.Path,
		Amount:      args.Amount,
		PubKey:      args.PubKey,
		RouterAddr:  args.RouterAddr,
		Slippage:    args.Slippage,
		IsInsurance: args.IsInsurance,
	}

	if evm.chainConfig.IsCIP2(evm.Context.BlockNumber) {
		item.Extra = from.Bytes()
	}

	item.FeeAmount = new(big.Int).Div(item.Amount, Int1000)
	IDHash := item.Hash()
	Nonce := evm.StateDB.GetNonce(from)
	isCip3 := evm.chainConfig.IsCIP3(evm.Context.BlockNumber)
	if isCip3 {
		item.ID = new(big.Int).Add(new(big.Int).SetBytes(IDHash[:10]), big.NewInt(int64(Nonce)))
	} else {
		item.ID = new(big.Int).SetBytes(IDHash[:10])
	}

	t2 := time.Now()

	if have, want := evm.StateDB.GetBalance(from), item.Amount; have.Cmp(want) < 0 {
		return nil, fmt.Errorf("%w: address %v have %v want %v", errors.New("insufficient funds for gas * price + value"), from, have, want)
	}

	evm.StateDB.SubBalance(from, args.Amount)
	evm.StateDB.AddBalance(CoinPools[ConvertType], new(big.Int).Sub(item.Amount, item.FeeAmount))
	evm.StateDB.AddBalance(Address0, item.FeeAmount)

	tewaka.Convert(item)

	t3 := time.Now()
	err = tewaka.Save(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking save state error", "error", err)
		return nil, err
	}

	t4 := time.Now()
	event := AbiTeWaKa.Events["casting"]
	logData, err := event.Inputs.Pack(item.ID, args.ConvertType, item.Path, item.PubKey, item.Amount, item.FeeAmount, item.RouterAddr, item.Slippage, item.IsInsurance, item.Extra)
	if err != nil {
		log.Error("Pack staking log error", "error", err)
		return nil, err
	}
	topics := []common.Hash{
		event.ID,
		common.BytesToHash(from[:]),
	}
	logN(evm, contract, topics, logData)
	context := []interface{}{
		"number", evm.Context.BlockNumber.Uint64(), "address", from, "Amount", item.Amount,
		"input", common.PrettyDuration(t1.Sub(t0)), "load", common.PrettyDuration(t2.Sub(t1)),
		"insert", common.PrettyDuration(t3.Sub(t2)), "save", common.PrettyDuration(t4.Sub(t3)),
		"log", common.PrettyDuration(time.Since(t4)), "elapsed", common.PrettyDuration(time.Since(t0)),
	}
	log.Debug("casting", context...)

	return nil, nil
}

func verifyConvertEthereumTypeTx(netName string, evm *EVM, AssetType uint8, TxHash common.Hash) (*types.ConvertItem, error) {

	receipt := receiptMap[TxHash]

	if receipt == nil {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) [txid:%s] not find", netName, TxHash)
	}

	if receipt.Status != 1 {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) [txid:%s] Status [%d]", netName, TxHash, receipt.Status)
	}

	if len(receipt.Logs) < 1 {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s)  receipt Logs length is 0 ", netName)
	}

	var txLog *types.Log
	for _, log := range receipt.Logs {
		if log.Topics[0].String() == burnTopics {
			txLog = log
			break
		}
	}

	if txLog == nil {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) txLog is nil ", netName)
	}

	logs := struct {
		From         common.Address
		AmountIn     *big.Int
		AmountOut    *big.Int
		ConvertType  *big.Int
		ToPath       []common.Address
		ToRouterAddr common.Address
		Slippage     *big.Int
		IsInsurance  bool
		Extra        []byte
	}{}

	if err := AbiCzzRouter.UnpackIntoInterface(&logs, "BurnToken", txLog.Data); err != nil {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s)  UnpackIntoInterface err (%s)", netName, err)
	}

	amountPool := evm.StateDB.GetBalance(CoinPools[AssetType])

	Amount := logs.AmountOut
	if logs.AmountOut.Cmp(big.NewInt(0)) == 0 {
		Amount = logs.AmountIn
	}

	TxAmount := new(big.Int).Mul(Amount, Int10)
	if TxAmount.Cmp(amountPool) > 0 {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) tx amount [%d] > pool [%d]", netName, TxAmount.Uint64(), amountPool)
	}

	if _, ok := CoinPools[uint8(logs.ConvertType.Uint64())]; !ok && uint8(logs.ConvertType.Uint64()) != 0 {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) ConvertType is [%d] CoinPools not find", netName, logs.ConvertType.Uint64())
	}

	if AssetType == uint8(logs.ConvertType.Uint64()) {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) AssetType = ConvertType = [%d]", netName, logs.ConvertType.Uint64())
	}

	extTx := transactionMap[TxHash]
	if err := CheckToAddress(AssetType, netName, extTx); err != nil {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) %s", netName, err)
	}

	Vb, R, S := extTx.RawSignatureValues()

	var plainV byte
	if isProtectedV(Vb) {
		chainID := deriveChainId(Vb).Uint64()
		plainV = byte(Vb.Uint64() - 35 - 2*chainID)
	} else {
		// If the signature is not optionally protected, we assume it
		// must already be equal to the recovery id.
		plainV = byte(Vb.Uint64())
	}

	if !crypto.ValidateSignatureValues(plainV, R, S, false) {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) ValidateSignatureValues invalid transaction v, r, s values", netName)
	}

	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = plainV

	var pk []byte
	var err error
	if extTx.Type() == types.LegacyTxType {
		a := types.NewEIP155Signer(extTx.ChainId())
		pk, err = crypto.Ecrecover(a.Hash(extTx).Bytes(), sig)
	} else if extTx.Type() == types.DynamicFeeTxType {
		a := types.NewLondonSigner(extTx.ChainId())
		pk, err = crypto.Ecrecover(a.Hash(extTx).Bytes(), sig)
	} else {
		a := types.NewEIP2930Signer(extTx.ChainId())
		pk, err = crypto.Ecrecover(a.Hash(extTx).Bytes(), sig)
	}

	if err != nil {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) Ecrecover err: %s", netName, err)
	}

	item := &types.ConvertItem{
		AssetType:   AssetType,
		ConvertType: uint8(logs.ConvertType.Uint64()),
		TxHash:      TxHash,
		PubKey:      pk,
		Amount:      Amount,
		Path:        logs.ToPath,
		RouterAddr:  logs.ToRouterAddr,
		IsInsurance: logs.IsInsurance,
		Slippage:    logs.Slippage,
		Extra:       logs.Extra,
	}

	return item, nil
}

func verifyConfirmEthereumTypeTx(netName string, tewaka *TeWakaImpl, ConvertType uint8, TxHash common.Hash, isCip2 bool) (*types.ConvertItem, error) {

	receipt := receiptMap[TxHash]

	if TxHash == common.HexToHash("0xcdde8c184a958fde12c07dadbcd90ca29831b633a371c40a01fdab0511372641") {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) [txid:%s] not find", netName, TxHash)
	}

	if receipt == nil {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) [txid:%s] not find", netName, TxHash)
	}

	if receipt.Status != 1 {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) [txid:%s] Status [%d]", netName, TxHash, receipt.Status)
	}

	if len(receipt.Logs) < 1 {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s)  receipt Logs length is 0 ", netName)
	}

	var txLog *types.Log
	for _, log := range receipt.Logs {
		if log.Topics[0].String() == mintTopics {
			txLog = log
			break
		}
	}

	if txLog == nil {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) txLog is nil ", netName)
	}

	logs := struct {
		To        common.Address
		Mid       *big.Int
		Gas       *big.Int
		AmountIn  *big.Int
		AmountOut *big.Int
	}{}

	if err := AbiCzzRouter.UnpackIntoInterface(&logs, "MintToken", txLog.Data); err != nil {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s)  UnpackIntoInterface err (%s)", netName, err)
	}

	var item *types.ConvertItem
	for _, v := range tewaka.ConvertItems {
		if v.ID.Cmp(logs.Mid) == 0 {
			item = v
			break
		}
	}

	if item == nil {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) ConvertItems [id:%d] is null", netName, logs.Mid.Uint64())
	}

	if item.ConvertType != ConvertType {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) ConvertType is [%d] not [%d] ", netName, ConvertType, item.ConvertType)
	}

	if isCip2 {
		if len(item.PubKey) > 0 {
			toaddresspuk, err := crypto.DecompressPubkey(item.PubKey)
			if err != nil || toaddresspuk == nil {
				toaddresspuk, err = crypto.UnmarshalPubkey(item.PubKey)
				if err != nil || toaddresspuk == nil {
					return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) toaddresspuk [puk:%s] is err: %s", netName, hex.EncodeToString(item.PubKey), err)
				}
			}
			toaddress := crypto.PubkeyToAddress(*toaddresspuk)
			if logs.To.String() != toaddress.String() {
				return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) [toaddress : %s] not [toaddress2 : %s]", netName, logs.To.String(), toaddress.String())
			}
		} else {
			fromAddr := common.BytesToAddress(item.Extra)
			if logs.To.String() != fromAddr.String() {
				return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) [toaddress : %s] not [toaddress2 : %s]", netName, logs.To.String(), fromAddr.String())
			}
		}
	}

	amount2 := big.NewInt(0).Sub(item.Amount, item.FeeAmount)
	if item.AssetType == ExpandedTxConvert_Czz {
		amount2 = new(big.Int).Div(amount2, Int10)
	}

	if logs.AmountIn.Cmp(amount2) != 0 {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) amount %d not %d", netName, logs.AmountIn, amount2)
	}

	extTx := transactionMap[TxHash]
	if extTx == nil {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) txjson is nil [txid:%s]", netName, TxHash)
	}

	if err := CheckToAddress(ConvertType, netName, extTx); err != nil {
		return nil, err
	}

	return item, nil
}

func crossToMainChainMap(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	args := struct {
		id              *big.Int
		amounts         *big.Int
		burnHash        common.Hash
		fromNetworkType *big.Int
		tewaka          common.Address
	}{}

	if evm.chainConfig.VerifySwitch {
		if err := verifyCrossToMainTxV3(evm, args.amounts, args.burnHash, args.fromNetworkType); err != nil {
			return nil, err
		}
	}

	method, _ := AbiTeWaKa.Methods["crossToMainChainMap"]
	err = method.Inputs.UnpackAtomic(&args, input)
	if err != nil {
		log.Error("Unpack convert pubkey error", "err", err)
		return nil, ErrStakingInvalidInput
	}

	from := contract.caller.Address()
	contractN := NewContract(AccountRef(from), AccountRef(from), big.NewInt(0), 100000)

	pinput := packInput("crossFromMainChain", args.id, args.amounts, args.burnHash, args.fromNetworkType)

	_, _, err = evm.Call(contractN, args.tewaka, pinput, 10000000, big.NewInt(0))

	return nil, err
}

func betweenSideChainCrossMap(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	args := struct {
		id              *big.Int
		amounts         *big.Int
		burnHash        common.Hash
		fromNetworkType *big.Int
		toNetworkType   *big.Int
		tewaka          common.Address
	}{}

	if evm.chainConfig.VerifySwitch {
		if err := verifyCrossToMainTxV3(evm, args.amounts, args.burnHash, args.fromNetworkType); err != nil {
			return nil, err
		}
	}

	method, _ := AbiTeWaKa.Methods["betweenSideChainCrossMap"]
	err = method.Inputs.UnpackAtomic(&args, input)
	if err != nil {
		log.Error("Unpack convert pubkey error", "err", err)
		return nil, ErrStakingInvalidInput
	}
	from := contract.caller.Address()
	contractN := NewContract(AccountRef(from), AccountRef(from), big.NewInt(0), 100000)

	pinput := packInput("betweenSideChainCrossMap", args.id, args.amounts, args.burnHash, args.fromNetworkType, args.toNetworkType)

	_, _, err = evm.Call(contractN, args.tewaka, pinput, 10000000, big.NewInt(0))

	return nil, err
}

func verifyCrossToMainTxV3(evm *EVM, amounts *big.Int, burnHash common.Hash, fromNetworkType *big.Int) error {

	client := evm.chainConfig.SideClients[uint8(fromNetworkType.Uint64())][0]
	if client == nil {
		return fmt.Errorf("verifyConvertEthereumTypeTx (%s) getTransactionReceipt SideClients is null ", fromNetworkType)
	}

	var receipt *types.Receipt
	if err := client.Call(&receipt, "eth_getTransactionReceipt", burnHash); err != nil {
		return fmt.Errorf("verifyConvertEthereumTypeTx (%d) getTransactionReceipt [txid:%s] err: %s", fromNetworkType, burnHash, err)
	}

	if receipt == nil {
		return fmt.Errorf("verifyConvertEthereumTypeTx (%d) [txid:%s] not find", fromNetworkType, burnHash)
	}

	if receipt.Status != 1 {
		return fmt.Errorf("verifyConvertEthereumTypeTx (%d) [txid:%s] Status [%d]", fromNetworkType, burnHash, receipt.Status)
	}

	if len(receipt.Logs) < 1 {
		return fmt.Errorf("verifyConvertEthereumTypeTx (%d)  receipt Logs length is 0 ", fromNetworkType)
	}

	var txLog *types.Log
	for _, log := range receipt.Logs {
		if log.Topics[0].String() == crossTopics {
			txLog = log
			break
		}
	}

	//
	if txLog == nil {
		return fmt.Errorf("verifyConvertEthereumTypeTx (%s) txLog is nil ", fromNetworkType)
	}

	logs := struct {
		From           common.Address
		AmountIn       *big.Int
		AmountOut      *big.Int
		ConvertType    *big.Int
		CrossToken     common.Address
		ToInfo         []byte
		ManagerAddress common.Address
	}{}

	if err := AbiTeWaKa.UnpackIntoInterface(&logs, "AtomBurnLog", txLog.Data); err != nil {
		return fmt.Errorf("verifyConvertEthereumTypeTx (%d)  UnpackIntoInterface err (%s)", fromNetworkType, err)
	}

	Amount := logs.AmountOut

	if Amount.Cmp(amounts) < 0 {
		return fmt.Errorf("verifyConvertEthereumTypeTx (%d)  Amount < amounts err", fromNetworkType)
	}

	var extTx *types.Transaction
	// Get the current block count.
	if err := client.Call(&extTx, "eth_getTransactionByHash", burnHash); err != nil {
		return fmt.Errorf("verifyConvertEthereumTypeTx (%s) getTransactionByHash [txid:%s] err: %s", fromNetworkType, extTx, err)
	}

	if err := CheckToAddress(uint8(fromNetworkType.Uint64()), fromNetworkType.String(), extTx); err != nil {
		return fmt.Errorf("verifyConvertEthereumTypeTx (%d) %s", fromNetworkType, err)
	}

	return nil
}

func CheckToAddress(ConvertType uint8, netName string, extTx *types.Transaction) error {
	// toaddress
	if ConvertType == ExpandedTxConvert_ECzz {
		if !strings.Contains(strings.ToUpper(ethPoolAddr), strings.ToUpper(extTx.To().String())) {
			return fmt.Errorf("verifyConvertEthereumTypeTx (%s) [ToAddress: %s] != [%s]", netName, extTx.To().String(), ethPoolAddr)
		}
	} else if ConvertType == ExpandedTxConvert_HCzz {
		if !strings.Contains(strings.ToUpper(hecoPoolAddr), strings.ToUpper(extTx.To().String())) {
			return fmt.Errorf("verifyConvertEthereumTypeTx (%s) [ToAddress: %s] != [%s]", netName, extTx.To().String(), hecoPoolAddr)
		}
	} else if ConvertType == ExpandedTxConvert_BCzz {
		if !strings.Contains(strings.ToUpper(bscPoolAddr), strings.ToUpper(extTx.To().String())) {
			return fmt.Errorf("verifyConvertEthereumTypeTx (%s) [ToAddress: %s] != [%s]", netName, extTx.To().String(), bscPoolAddr)
		}
	} else if ConvertType == ExpandedTxConvert_OCzz {
		if !strings.Contains(strings.ToUpper(oecPoolAddr), strings.ToUpper(extTx.To().String())) {
			return fmt.Errorf("verifyConvertEthereumTypeTx (%s) [ToAddress: %s] != [%s]", netName, extTx.To().String(), oecPoolAddr)
		}
	} else if ConvertType == ExpandedTxConvert_PCzz {
		if !strings.Contains(strings.ToUpper(polygonPoolAddr), strings.ToUpper(extTx.To().String())) {
			return fmt.Errorf("verifyConvertEthereumTypeTx (%s) [ToAddress: %s] != [%s]", netName, extTx.To().String(), polygonPoolAddr)
		}
	} else if ConvertType == ExpandedTxConvert_MCzz {
		if !strings.Contains(strings.ToUpper(metisPoolAddr), strings.ToUpper(extTx.To().String())) {
			return fmt.Errorf("verifyConvertEthereumTypeTx (%s) [ToAddress: %s] != [%s]", netName, extTx.To().String(), polygonPoolAddr)
		}
	} else if ConvertType == ExpandedTxConvert_GCzz {
		if !strings.Contains(strings.ToUpper(gatePoolAddr), strings.ToUpper(extTx.To().String())) {
			return fmt.Errorf("verifyConvertEthereumTypeTx (%s) [ToAddress: %s] != [%s]", netName, extTx.To().String(), polygonPoolAddr)
		}
	}
	return nil
}

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28 && v != 1 && v != 0
	}
	// anything not 27 or 28 is considered protected
	return true
}

// deriveChainId derives the chain id from the given v parameter
func deriveChainId(v *big.Int) *big.Int {
	if v.BitLen() <= 64 {
		v := v.Uint64()
		if v == 27 || v == 28 {
			return new(big.Int)
		}
		return new(big.Int).SetUint64((v - 35) / 2)
	}
	v = new(big.Int).Sub(v, big.NewInt(35))
	return v.Div(v, big.NewInt(2))
}

func packInput(abiMethod string, params ...interface{}) []byte {
	input, err := AbiTeWakaV3.Pack(abiMethod, params...)
	if err != nil {
		log.Error("packInput", "err", err)
	}
	return input
}

const TeWakaV3ABI = `
[
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "id",
				"type": "uint256"
			},
			{
				"internalType": "uint256",
				"name": "_amounts",
				"type": "uint256"
			},
			{
				"internalType": "bytes32",
				"name": "_burnHash",
				"type": "bytes32"
			},
			{
				"internalType": "uint256",
				"name": "_fromNetworkType",
				"type": "uint256"
			}
		],
		"name": "crossToMainChainMap",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "id",
				"type": "uint256"
			},
			{
				"internalType": "uint256",
				"name": "_amounts",
				"type": "uint256"
			},
			{
				"internalType": "bytes32",
				"name": "_burnHash",
				"type": "bytes32"
			},
			{
				"internalType": "uint256",
				"name": "_fromNetworkType",
				"type": "uint256"
			},
			{
				"internalType": "uint256",
				"name": "_toNetworkType",
				"type": "uint256"
			}
		],
		"name": "betweenSideChainCrossMap",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	}
]
`

const TeWakaABI = `
[
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "address",
				"name": "from_",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "amountIn",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "amountOut",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "convertType",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "address",
				"name": "crossToken",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "bytes",
				"name": "toInfo",
				"type": "bytes"
			},
			{
				"indexed": false,
				"internalType": "address",
				"name": "managerAddress",
				"type": "address"
			}
		],
		"name": "AtomBurnLog",
		"type": "event"
	},
    {
        "name":"mortgage",
        "inputs":[
            {
                "type":"bytes",
                "name":"pubKey"
            },
  			{
                "type":"address",
                "name":"toAddress"
            },
            {
                "type":"uint256",
                "name":"stakingAmount"
            },
            {
                "type":"address[]",
                "name":"coinBaseAddress"
            }
        ],
        "anonymous":false,
        "type":"event"
    },
    {
        "name":"mortgage",
        "outputs":[

        ],
        "inputs":[
 			{
                "type":"bytes",
                "name":"pubKey"
            },
            {
                "type":"address",
                "name":"toAddress"
            },
            {
                "type":"uint256",
                "name":"stakingAmount"
            },
            {
                "type":"address[]",
                "name":"coinBaseAddress"
            }
        ],
        "constant":false,
        "payable":false,
        "type":"function"
    },
    {
        "name":"update",
        "inputs":[
			{
                "type":"uint256",
                "name":"stakingAmount"
            },
            {
                "type":"address[]",
                "name":"coinBaseAddress"
            }
        ],
        "anonymous":false,
        "type":"event"
    },
    {
        "name":"update",
        "outputs":[

        ],
        "inputs":[
			{
                "type":"uint256",
                "name":"stakingAmount"
            },
            {
                "type":"address[]",
                "name":"coinBaseAddress"
            }
        ],
        "constant":false,
        "payable":false,
        "type":"function"
    },
    {
        "name":"convert",
        "inputs":[
            {
                "type":"uint256",
                "name":"ID"
            },
            {
                "type":"uint256",
                "name":"AssetType"
            },
            {
                "type":"uint256",
                "name":"ConvertType"
            },
            {
                "type":"string",
                "name":"TxHash"
            },
            {
                "type":"address[]",
                "name":"Path"
            },
            {
                "type":"address",
                "name":"RouterAddr"
            },
            {
                "type":"bytes",
                "name":"PubKey"
            },
            {
                "type":"uint256",
                "name":"Amount"
            },
            {
                "type":"uint256",
                "name":"FeeAmount"
            },
            {
                "type":"uint256",
                "name":"Slippage"
            },
            {
                "type":"bool",
                "name":"IsInsurance"
            },
            {
                "type":"bytes",
                "name":"Extra"
            }
        ],
        "anonymous":false,
        "type":"event"
    },
    {
        "name":"convert",
        "outputs":[

        ],
        "inputs":[
            {
                "type":"uint256",
                "name":"AssetType"
            },
            {
                "type":"string",
                "name":"TxHash"
            }
        ],
        "constant":false,
        "payable":false,
        "type":"function"
    },
    {
        "name":"confirm",
        "inputs":[
            {
                "type":"uint256",
                "name":"ID"
            },
            {
                "type":"uint256",
                "name":"AssetType"
            },
            {
                "type":"uint256",
                "name":"ConvertType"
            },
            {
                "type":"string",
                "name":"TxHash"
            }
        ],
        "anonymous":false,
        "type":"event"
    },
    {
        "name":"confirm",
        "outputs":[

        ],
        "inputs":[
            {
                "type":"uint256",
                "name":"ConvertType"
            },
            {
                "type":"string",
                "name":"TxHash"
            }
        ],
        "constant":false,
        "payable":false,
        "type":"function"
    },
    {
        "name":"casting",
        "inputs":[
            {
                "type":"uint256",
                "name":"ID"
            },
            {
                "type":"uint256",
                "name":"ConvertType"
            },
            {
                "type":"address[]",
                "name":"Path"
            },
            {
                "type":"bytes",
                "name":"PubKey"
            },
            {
                "type":"uint256",
                "name":"Amount"
            },
            {
                "type":"uint256",
                "name":"FeeAmount"
            },
            {
                "type":"address",
                "name":"RouterAddr"
            },
            {
                "type":"uint256",
                "name":"slippage"
            },
            {
                "type":"bool",
                "name":"IsInsurance"
            },
            {
                "type":"bytes",
                "name":"Extra"
            }
        ],
        "anonymous":false,
        "type":"event"
    },
    {
        "name":"casting",
        "outputs":[

        ],
        "inputs":[
            {
                "type":"uint256",
                "name":"ConvertType"
            },
            {
                "type":"uint256",
                "name":"Amount"
            },
            {
                "type":"address[]",
                "name":"Path"
            },
            {
                "type":"bytes",
                "name":"PubKey"
            },
            {
                "type":"address",
                "name":"RouterAddr"
            },
            {
                "type":"uint256",
                "name":"slippage"
            },
            {
                "type":"bool",
                "name":"IsInsurance"
            }
        ],
        "constant":false,
        "payable":false,
        "type":"function"
    },
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "id",
				"type": "uint256"
			},
			{
				"internalType": "uint256",
				"name": "_amounts",
				"type": "uint256"
			},
			{
				"internalType": "bytes32",
				"name": "_burnHash",
				"type": "bytes32"
			},
			{
				"internalType": "uint256",
				"name": "_fromNetworkType",
				"type": "uint256"
			},
			{
				"name": "tewaka",
				"type": "address"
			}
		],
		"name": "crossToMainChainMap",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "id",
				"type": "uint256"
			},
			{
				"internalType": "uint256",
				"name": "_amounts",
				"type": "uint256"
			},
			{
				"internalType": "bytes32",
				"name": "_burnHash",
				"type": "bytes32"
			},
			{
				"internalType": "uint256",
				"name": "_fromNetworkType",
				"type": "uint256"
			},
			{
				"internalType": "uint256",
				"name": "_toNetworkType",
				"type": "uint256"
			},
			{
				"name": "tewaka",
				"type": "address"
			}
		],
		"name": "betweenSideChainCrossMap",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	}
]
`

const CzzRouterABI = `
[
	{
		"anonymous": false,
		"inputs": [
			{
				"name": "from_",
				"type": "address"
			},
			{
				"name": "amountIn",
				"type": "uint256"
			},
			{
				"name": "amountOut",
				"type": "uint256"
			},
			{
				"name": "convertType",
				"type": "uint256"
			},
			{
				"name": "toPath",
				"type": "address[]"
			},
			{
				"name": "toRouterAddr",
				"type": "address"
			},
			{
				"name": "slippage",
				"type": "uint256"
			},
			{
				"name": "isInsurance",
				"type": "bool"
			},
			{
				"name": "extra",
				"type": "bytes"
			}
		],
		"name": "BurnToken",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"name": "to",
				"type": "address"
			},
			{
				"name": "mid",
				"type": "uint256"
			},
			{
				"name": "gas",
				"type": "uint256"
			},
			{
				"name": "amountIn",
				"type": "uint256"
			},
			{
				"name": "amountOut",
				"type": "uint256"
			}
		],
		"name": "MintToken",
		"type": "event"
	}
]
`
