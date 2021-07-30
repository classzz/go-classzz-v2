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
	"errors"
	"fmt"
	"github.com/classzz/go-classzz-v2/crypto"
	"github.com/classzz/go-classzz-v2/rpc"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/classzz/go-classzz-v2/accounts/abi"
	"github.com/classzz/go-classzz-v2/common"
	"github.com/classzz/go-classzz-v2/core/types"
	"github.com/classzz/go-classzz-v2/log"
)

const (
	// Entangle Transcation type
	ExpandedTxConvert_Czz uint8 = iota
	ExpandedTxConvert_ECzz
	ExpandedTxConvert_HCzz
	ExpandedTxConvert_BCzz
	ExpandedTxConvert_OCzz
)

var (
	baseUnit  = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	fbaseUnit = new(big.Float).SetFloat64(float64(baseUnit.Int64()))
	mixImpawn = new(big.Int).Mul(big.NewInt(1000), baseUnit)
	Base      = new(big.Int).SetUint64(10000)

	// i.e. contractAddress = 0x000000000000000000747275657374616b696E67
	TeWaKaAddress = common.BytesToAddress([]byte("tewaka"))
	CoinPools     = map[uint8]common.Address{
		ExpandedTxConvert_ECzz: {0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 101},
		ExpandedTxConvert_HCzz: {0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 102},
		ExpandedTxConvert_BCzz: {0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 103},
		ExpandedTxConvert_OCzz: {0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 104},
	}

	ethPoolAddr  = "0x9ac88c5136240312f8817dbb99497ace62b03f12|0xB2451147c6154659c350EaC39ED37599bff4d32e|0xF0f50ce5054289a178fb45Ab2E373899580d12bf"
	hecoPoolAddr = "0x711d839cd1e6e81b971f5b6bbb4a6bd7c4b60ac6|0xdc3013FcF6A748c6b468de21b8A1680dbcb979ca|0x93E00a89F5CBF9c66a50aF7206c9c6f54601EC15|0x30d0e3F30D527373a27A2177fAcb4bdCc046DC1C"
	bscPoolAddr  = "0x007c98F9f2c70746a64572E67FBCc41a2b8bba18|0x711D839CD1E6E81B971F5b6bBB4a6BD7C4B60Ac6|0xdf10e0Caa2BBe67f7a1E91A3e6660cC1e34e81B9|0xa5D17B93f4156afd96be9f5B40888ffb47fA4bc1"
	okexPoolAddr = "0x007c98F9f2c70746a64572E67FBCc41a2b8bba18|0x711D839CD1E6E81B971F5b6bBB4a6BD7C4B60Ac6|0xdf10e0Caa2BBe67f7a1E91A3e6660cC1e34e81B9|0xa5D17B93f4156afd96be9f5B40888ffb47fA4bc1"

	burnTopics = "0x86f32d6c7a935bd338ee00610630fcfb6f043a6ad755db62064ce2ad92c45caa"
	mintTopics = "0x8fb5c7bffbb272c541556c455c74269997b816df24f56dd255c2391d92d4f1e9"
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
var abiTeWaKa abi.ABI

type StakeContract struct{}

func init() {
	abiTeWaKa, _ = abi.JSON(strings.NewReader(TeWakaABI))
}

// RunStaking execute staking contract
func RunStaking(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {

	method, err := abiTeWaKa.MethodById(input)

	if err != nil {
		log.Error("No method found")
		return nil, ErrExecutionReverted
	}

	data := input[4:]

	switch method.Name {
	case "mortgage":
		ret, err = mortgage(evm, contract, data)
	case "update":
		ret, err = update(evm, contract, data)
	case "convert":
		ret, err = convert(evm, contract, data)
	case "confirm":
		ret, err = confirm(evm, contract, data)
	case "casting":
		ret, err = casting(evm, contract, data)
	default:
		log.Warn("Staking call fallback function")
		err = ErrStakingInvalidInput
	}

	if err != nil {
		log.Warn("Staking error code", "code", err)
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
	method, _ := abiTeWaKa.Methods["mortgage"]

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
	event := abiTeWaKa.Events["mortgage"]
	logData, err := event.Inputs.Pack(args.ToAddress, args.StakingAmount, args.CoinBaseAddress)
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
	log.Info("mortgage", context...)
	return nil, nil
}

// Update
func update(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	t0 := time.Now()
	args := struct {
		CoinBaseAddress []common.Address
	}{}

	method, _ := abiTeWaKa.Methods["update"]
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

	t2 := time.Now()
	tewaka.Update(from, args.CoinBaseAddress)

	t3 := time.Now()
	err = tewaka.Save(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking save state error", "error", err)
		return nil, err
	}

	t4 := time.Now()
	event := abiTeWaKa.Events["update"]
	logData, err := event.Inputs.Pack(args.CoinBaseAddress)
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
	log.Info("update", context...)
	return nil, nil
}

// Convert
func convert(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	t0 := time.Now()
	args := struct {
		AssetType   uint8
		ConvertType uint8
		TxHash      string
	}{}

	method, _ := abiTeWaKa.Methods["convert"]
	err = method.Inputs.UnpackAtomic(&args, input)
	if err != nil {
		log.Error("Unpack convert pubkey error", "err", err)
		return nil, ErrStakingInvalidInput
	}

	from := contract.caller.Address()
	t1 := time.Now()

	if exit := evm.StateDB.HasRecord(common.HexToHash(args.TxHash)); exit {
		return nil, ErrTxhashAlreadyInput
	}

	tewaka := NewTeWakaImpl()
	err = tewaka.Load(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking load error", "error", err)
		return nil, err
	}

	var item *types.ConvertItem
	switch args.AssetType {
	case ExpandedTxConvert_ECzz:
		client := evm.chainConfig.EthClient[rand.Intn(len(evm.chainConfig.EthClient))]
		if item, err = verifyConvertEthereumTypeTx("ETH", evm, client, tewaka, args.AssetType, args.ConvertType, args.TxHash); err != nil {
			return nil, err
		}
	case ExpandedTxConvert_HCzz:
		client := evm.chainConfig.HecoClient[rand.Intn(len(evm.chainConfig.HecoClient))]
		if item, err = verifyConvertEthereumTypeTx("HECO", evm, client, tewaka, args.AssetType, args.ConvertType, args.TxHash); err != nil {
			return nil, err
		}
	case ExpandedTxConvert_BCzz:
		client := evm.chainConfig.BscClient[rand.Intn(len(evm.chainConfig.BscClient))]
		if item, err = verifyConvertEthereumTypeTx("BSC", evm, client, tewaka, args.AssetType, args.ConvertType, args.TxHash); err != nil {
			return nil, err
		}
	case ExpandedTxConvert_OCzz:
		client := evm.chainConfig.OkexClient[rand.Intn(len(evm.chainConfig.OkexClient))]
		if item, err = verifyConvertEthereumTypeTx("OKEX", evm, client, tewaka, args.AssetType, args.ConvertType, args.TxHash); err != nil {
			return nil, err
		}
	}

	item.FeeAmount = big.NewInt(0).Div(item.Amount, big.NewInt(1000))
	item.ID = big.NewInt(rand.New(rand.NewSource(time.Now().Unix())).Int63())
	item.Committee = tewaka.GetCommittee()

	t2 := time.Now()
	tewaka.Convert(item)

	evm.StateDB.SubBalance(CoinPools[item.AssetType], item.Amount)
	evm.StateDB.AddBalance(CoinPools[item.ConvertType], new(big.Int).Sub(item.Amount, item.FeeAmount))

	t3 := time.Now()
	err = tewaka.Save(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking save state error", "error", err)
		return nil, err
	}
	evm.StateDB.WriteRecord(common.HexToHash(args.TxHash))

	t4 := time.Now()
	event := abiTeWaKa.Events["convert"]
	logData, err := event.Inputs.Pack(item.AssetType, item.ConvertType, item.TxHash, item.PubKey, item.ToToken, item.Committee, item.Amount, item.FeeAmount)
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
		"AssetType", args.AssetType, "ConvertType", args.ConvertType, "TxHash", args.TxHash,
		"input", common.PrettyDuration(t1.Sub(t0)), "load", common.PrettyDuration(t2.Sub(t1)),
		"insert", common.PrettyDuration(t3.Sub(t2)), "save", common.PrettyDuration(t4.Sub(t3)),
		"log", common.PrettyDuration(time.Since(t4)), "elapsed", common.PrettyDuration(time.Since(t0)),
	}
	log.Info("convert", context...)

	return nil, nil
}

// Confirm
func confirm(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	t0 := time.Now()
	args := struct {
		AssetType   uint8
		ConvertType uint8
		TxHash      string
	}{}

	method, _ := abiTeWaKa.Methods["confirm"]
	err = method.Inputs.UnpackAtomic(&args, input)
	if err != nil {
		log.Error("Unpack convert pubkey error", "err", err)
		return nil, ErrStakingInvalidInput
	}

	from := contract.caller.Address()
	t1 := time.Now()

	if exit := evm.StateDB.HasRecord(common.HexToHash(args.TxHash)); exit {
		return nil, ErrTxhashAlreadyInput
	}

	tewaka := NewTeWakaImpl()
	err = tewaka.Load(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking load error", "error", err)
		return nil, err
	}

	var item *types.ConvertItem
	switch args.AssetType {
	case ExpandedTxConvert_ECzz:
		client := evm.chainConfig.EthClient[rand.Intn(len(evm.chainConfig.EthClient))]
		if item, err = verifyConfirmEthereumTypeTx("ETH", client, tewaka, args.AssetType, args.ConvertType, args.TxHash); err != nil {
			return nil, err
		}
	case ExpandedTxConvert_HCzz:
		client := evm.chainConfig.HecoClient[rand.Intn(len(evm.chainConfig.HecoClient))]
		if item, err = verifyConfirmEthereumTypeTx("HECO", client, tewaka, args.AssetType, args.ConvertType, args.TxHash); err != nil {
			return nil, err
		}
	case ExpandedTxConvert_BCzz:
		client := evm.chainConfig.BscClient[rand.Intn(len(evm.chainConfig.BscClient))]
		if item, err = verifyConfirmEthereumTypeTx("BSC", client, tewaka, args.AssetType, args.ConvertType, args.TxHash); err != nil {
			return nil, err
		}
	case ExpandedTxConvert_OCzz:
		client := evm.chainConfig.OkexClient[rand.Intn(len(evm.chainConfig.OkexClient))]
		if item, err = verifyConfirmEthereumTypeTx("OKEX", client, tewaka, args.AssetType, args.ConvertType, args.TxHash); err != nil {
			return nil, err
		}
	}

	t2 := time.Now()
	tewaka.Confirm(item)

	t3 := time.Now()
	err = tewaka.Save(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking save state error", "error", err)
		return nil, err
	}

	evm.StateDB.WriteRecord(common.HexToHash(args.TxHash))

	t4 := time.Now()
	event := abiTeWaKa.Events["confirm"]
	logData, err := event.Inputs.Pack(item.AssetType, item.ConvertType, item.TxHash, item.PubKey, item.ToToken, item.Amount, item.FeeAmount)
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
		"AssetType", args.AssetType, "ConvertType", args.ConvertType, "TxHash", args.TxHash,
		"input", common.PrettyDuration(t1.Sub(t0)), "load", common.PrettyDuration(t2.Sub(t1)),
		"insert", common.PrettyDuration(t3.Sub(t2)), "save", common.PrettyDuration(t4.Sub(t3)),
		"log", common.PrettyDuration(time.Since(t4)), "elapsed", common.PrettyDuration(time.Since(t0)),
	}
	log.Info("convert", context...)

	return nil, nil
}

// Casting
func casting(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	t0 := time.Now()
	args := struct {
		ConvertType uint8
		Amount      *big.Int
		ToToken     string
	}{}

	method, _ := abiTeWaKa.Methods["casting"]
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

	item := &types.ConvertItem{
		ConvertType: args.ConvertType,
		ToToken:     args.ToToken,
		Amount:      args.Amount,
	}

	t2 := time.Now()
	tewaka.Convert(item)

	if have, want := evm.StateDB.GetBalance(from), args.Amount; have.Cmp(want) < 0 {
		return nil, fmt.Errorf("%w: address %v have %v want %v", errors.New("insufficient funds for gas * price + value"), from, have, want)
	}

	evm.StateDB.SubBalance(from, item.Amount)

	t3 := time.Now()
	err = tewaka.Save(evm.StateDB, TeWaKaAddress)
	if err != nil {
		log.Error("Staking save state error", "error", err)
		return nil, err
	}

	t4 := time.Now()
	event := abiTeWaKa.Events["casting"]
	logData, err := event.Inputs.Pack(item.ConvertType, item.Amount, item.PubKey, item.ToToken)
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
	log.Info("convert", context...)

	return nil, nil
}

func verifyConvertEthereumTypeTx(netName string, evm *EVM, client *rpc.Client, tewaka *TeWakaImpl, AssetType uint8, ConvertType uint8, TxHash string) (*types.ConvertItem, error) {

	if AssetType == ConvertType {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) AssetType = ConvertType = [%d]", netName, ConvertType)
	}

	if _, ok := CoinPools[ConvertType]; !ok && ConvertType != 0 {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) ConvertType is [%d] CoinPools not find", netName, ConvertType)
	}

	var receipt *types.Receipt
	if err := client.Call(&receipt, "eth_getTransactionReceipt", TxHash); err != nil {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) getTransactionReceipt [txid:%s] err: %s", netName, TxHash, err)
	}

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

	amount := txLog.Data[:32]
	ntype := txLog.Data[32:64]
	toToken := txLog.Data[64:]

	TxAmount := big.NewInt(0).SetBytes(amount)
	amountPool := evm.StateDB.GetBalance(CoinPools[AssetType])
	if TxAmount.Cmp(amountPool) > 0 {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) tx amount [%d] > pool [%d]", netName, TxAmount.Uint64(), amountPool)
	}

	if big.NewInt(0).SetBytes(ntype).Uint64() != uint64(ConvertType) {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s)  ntype [%d] not [%d]", netName, big.NewInt(0).SetBytes(ntype), ConvertType)
	}

	var extTx *types.Transaction
	// Get the current block count.
	if err := client.Call(&extTx, "eth_getTransactionByHash", TxHash); err != nil {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) getTransactionByHash [txid:%s] err: %s", netName, TxHash, err)
	}

	if AssetType == ExpandedTxConvert_ECzz {
		if !strings.Contains(ethPoolAddr, extTx.To().String()) {
			return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) ETh [ToAddress: %s] != [%s]", netName, extTx.To().String(), ethPoolAddr)
		}
	} else if AssetType == ExpandedTxConvert_HCzz {
		if !strings.Contains(hecoPoolAddr, extTx.To().String()) {
			return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) Heco [ToAddress: %s] != [%s]", netName, extTx.To().String(), ethPoolAddr)
		}
	} else if AssetType == ExpandedTxConvert_BCzz {
		if !strings.Contains(bscPoolAddr, extTx.To().String()) {
			return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) Bsc [ToAddress: %s] != [%s]", netName, extTx.To().String(), ethPoolAddr)
		}
	} else if AssetType == ExpandedTxConvert_OCzz {
		if !strings.Contains(bscPoolAddr, extTx.To().String()) {
			return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) Bsc [ToAddress: %s] != [%s]", netName, extTx.To().String(), ethPoolAddr)
		}
	}

	Vb, R, S := extTx.RawSignatureValues()
	var V byte

	var chainID *big.Int
	if isProtectedV(Vb) {
		chainID = deriveChainId(Vb)
		V = byte(Vb.Uint64() - 35 - 2*chainID.Uint64())
	} else {
		V = byte(Vb.Uint64() - 27)
	}

	if !crypto.ValidateSignatureValues(V, R, S, false) {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) ValidateSignatureValues err", netName)
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	a := types.NewEIP155Signer(chainID)
	pk, err := crypto.Ecrecover(a.Hash(extTx).Bytes(), sig)
	if err != nil {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) Ecrecover err: %s", netName, err)
	}

	item := &types.ConvertItem{
		AssetType:   AssetType,
		ConvertType: ConvertType,
		TxHash:      TxHash,
		PubKey:      pk,
		Amount:      TxAmount,
		ToToken:     string(toToken),
	}

	return item, nil
}

func verifyConfirmEthereumTypeTx(netName string, client *rpc.Client, tewaka *TeWakaImpl, AssetType uint8, ConvertType uint8, TxHash string) (*types.ConvertItem, error) {

	if AssetType == ConvertType {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) AssetType = ConvertType = [%d]", netName, ConvertType)
	}

	if _, ok := CoinPools[AssetType]; !ok && AssetType != 0 {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) AssetType is [%d] CoinPools not find", netName, AssetType)
	}

	var receipt *types.Receipt
	if err := client.Call(&receipt, "eth_getTransactionReceipt", TxHash); err != nil {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) getTransactionReceipt [txid:%s] err: %s", netName, TxHash, err)
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

	address := txLog.Topics[1]
	id := txLog.Data[32:64]
	amount := txLog.Data[64:]
	ID := new(big.Int).SetBytes(id)

	var item *types.ConvertItem
	for _, v := range tewaka.ConvertItems {
		if v.ID.Cmp(ID) == 0 {
			item = v
			break
		}
	}

	if item == nil {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) ConvertItems [id:%d] is null", netName, ID)
	}

	toaddresspuk, err := crypto.DecompressPubkey(item.PubKey)
	if err != nil || toaddresspuk == nil {
		toaddresspuk, err = crypto.UnmarshalPubkey(item.PubKey)
		if err != nil || toaddresspuk == nil {
			return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) toaddresspuk [puk:%s] is err: %s", netName, hex.EncodeToString(item.PubKey), err)
		}
	}

	toaddress := common.BytesToAddress(address.Bytes())
	toaddress2 := crypto.PubkeyToAddress(*toaddresspuk)

	if toaddress.String() != toaddress2.String() {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) [toaddress : %s] not [toaddress2 : %s]", netName, toaddress.String(), toaddress2.String())
	}

	amount2 := big.NewInt(0).Sub(item.Amount, item.FeeAmount)
	if big.NewInt(0).SetBytes(amount).Cmp(amount2) != 0 {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) amount %d not %d", netName, big.NewInt(0).SetBytes(amount), amount2)
	}

	var extTx *types.Transaction
	// Get the current block count.
	if err := client.Call(&extTx, "eth_getTransactionByHash", TxHash); err != nil {
		return nil, err
	}

	if extTx == nil {
		return nil, fmt.Errorf("verifyConfirmEthereumTypeTx (%s) txjson is nil [txid:%s]", netName, TxHash)
	}

	// toaddress
	if ConvertType == ExpandedTxConvert_ECzz {
		if !strings.Contains(ethPoolAddr, extTx.To().String()) {
			return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) ETh [ToAddress: %s] != [%s]", netName, extTx.To().String(), ethPoolAddr)
		}
	} else if ConvertType == ExpandedTxConvert_HCzz {
		if !strings.Contains(hecoPoolAddr, extTx.To().String()) {
			return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) Heco [ToAddress: %s] != [%s]", netName, extTx.To().String(), hecoPoolAddr)
		}
	} else if ConvertType == ExpandedTxConvert_BCzz {
		if !strings.Contains(bscPoolAddr, extTx.To().String()) {
			return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) Bsc [ToAddress: %s] != [%s]", netName, extTx.To().String(), bscPoolAddr)
		}
	} else if ConvertType == ExpandedTxConvert_OCzz {
		if !strings.Contains(okexPoolAddr, extTx.To().String()) {
			return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) Okex [ToAddress: %s] != [%s]", netName, extTx.To().String(), okexPoolAddr)
		}
	}

	return item, nil
}

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28
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

const TeWakaABI = `
[
  {
    "name": "mortgage",
    "inputs": [
        {
        "type": "address",
        "name": "toAddress"
      },
      {
        "type": "uint256",
        "unit": "wei",
        "name": "stakingAmount"
      },
      {
        "type": "address[]",
        "name": "coinBaseAddress"
      }
    ],
    "anonymous": false,
    "type": "event"
  },
  {
    "name": "mortgage",
    "outputs": [],
    "inputs": [
      {
        "type": "address",
        "name": "toAddress"
      },
      {
        "type": "uint256",
        "unit": "wei",
        "name": "stakingAmount"
      },
      {
        "type": "address[]",
        "name": "coinBaseAddress"
      }
    ],
    "constant": false,
    "payable": false,
    "type": "function"
  }, {
    "name": "update",
    "inputs": [
      {
        "type": "address[]",
        "name": "coinBaseAddress"
      }
    ],
    "anonymous": false,
    "type": "event"
  },
  {
    "name": "update",
    "outputs": [],
    "inputs": [
      {
        "type": "address[]",
        "name": "coinBaseAddress"
      }
    ],
    "constant": false,
    "payable": false,
    "type": "function"
  }, 
{
    "name": "convert",
    "inputs": [
      {
        "type": "uint256",
        "name": "AssetType"
      },{
        "type": "uint256",
        "name": "ConvertType"
      },{
        "type": "string",
        "name": "TxHash"
      }
    ],
    "anonymous": false,
    "type": "event"
  },
  {
    "name": "convert",
    "outputs": [],
    "inputs": [
      {
        "type": "uint256",
        "name": "AssetType"
      },{
        "type": "uint256",
        "name": "ConvertType"
      },{
        "type": "string",
        "name": "TxHash"
      },{
        "type": "string",
        "name": "ToToken"
      },{
        "type": "bytes",
        "name": "PubKey"
      },{
        "type": "address",
        "name": "Committee"
      },{
        "type": "uint256",
        "name": "Amount"
      },{
        "type": "uint256",
        "name": "FeeAmount"
      }
    ],
    "constant": false,
    "payable": false,
    "type": "function"
  },
{
    "name": "confirm",
    "inputs": [
      {
        "type": "uint256",
        "name": "AssetType"
      },{
        "type": "uint256",
        "name": "ConvertType"
      },{
        "type": "string",
        "name": "TxHash"
      }
    ],
    "anonymous": false,
    "type": "event"
  },
  {
    "name": "confirm",
    "outputs": [],
    "inputs": [
      {
        "type": "uint256",
        "name": "AssetType"
      },{
        "type": "uint256",
        "name": "ConvertType"
      },{
        "type": "string",
        "name": "TxHash"
      },{
        "type": "string",
        "name": "ToToken"
      },{
        "type": "bytes",
        "name": "PubKey"
      },{
        "type": "uint256",
        "name": "Amount"
      },{
        "type": "uint256",
        "name": "FeeAmount"
      }
    ],
    "constant": false,
    "payable": false,
    "type": "function"
  },
{
    "name": "casting",
    "inputs": [
      {
        "type": "uint256",
        "name": "AssetType"
      },{
        "type": "uint256",
        "name": "Amount"
      },{
        "type": "string",
        "name": "ToToken"
      }
    ],
    "anonymous": false,
    "type": "event"
  },
  {
    "name": "casting",
    "outputs": [],
    "inputs": [
      {
        "type": "uint256",
        "name": "ConvertType"
      },{
        "type": "uint256",
        "name": "Amount"
      },{
        "type": "bytes",
        "name": "PubKey"
      },{
        "type": "string",
        "name": "ToToken"
      }
    ],
    "constant": false,
    "payable": false,
    "type": "function"
  }
]
`
