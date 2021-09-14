package main

import (
	"encoding/hex"
	"github.com/classzz/go-classzz-v2/cmd/utils"
	"github.com/classzz/go-classzz-v2/common"
	"github.com/classzz/go-classzz-v2/core/vm"
	"gopkg.in/urfave/cli.v1"
	"math/big"
)

var MortgageCommand = cli.Command{
	Name:   "mortgage",
	Usage:  "mortgage validator deposit staking count",
	Action: utils.MigrateFlags(Mortgage),
	Flags:  append(TeWakaFlags),
}

func Mortgage(ctx *cli.Context) error {

	loadPrivate(ctx)

	conn, url := dialConn(ctx)

	pubKey := ctx.GlobalString(MortgageFlags[0].GetName())
	pubKeyb, _ := hex.DecodeString(pubKey)

	toAddress := ctx.GlobalString(MortgageFlags[1].GetName())

	stakingAmount := ctx.GlobalInt64(MortgageFlags[2].GetName())
	Amount := czzToWei(stakingAmount)
	if stakingAmount < TeWakaAmount {
		printError("mortgage value must bigger than ", TeWakaAmount)
	}

	coinBaseAddress := ctx.GlobalStringSlice(MortgageFlags[3].GetName())
	cbas := []common.Address{}
	for _, v := range coinBaseAddress {
		cbas = append(cbas, common.HexToAddress(v))
	}

	printBaseInfo(conn, url)
	PrintBalance(conn, from, common.HexToAddress(toAddress))

	input := packInput("mortgage", pubKeyb, common.HexToAddress(toAddress), Amount, cbas)
	txHash := sendContractTransaction(conn, from, vm.TeWaKaAddress, nil, priKey, input)
	getResult(conn, txHash, true, false)

	return nil
}

var UpdateCommand = cli.Command{
	Name:   "update",
	Usage:  "update",
	Action: utils.MigrateFlags(Update),
	Flags:  append(TeWakaFlags),
}

func Update(ctx *cli.Context) error {

	loadPrivate(ctx)

	conn, _ := dialConn(ctx)

	stakingAmount := ctx.GlobalInt64(MortgageFlags[2].GetName())
	Amount := czzToWei(stakingAmount)
	if stakingAmount != 0 && stakingAmount < TeWakaAmount {
		printError("mortgage value must bigger than ", TeWakaAmount)
	}

	coinBaseAddress := ctx.GlobalStringSlice(MortgageFlags[3].GetName())
	cbas := []common.Address{}
	for _, v := range coinBaseAddress {
		cbas = append(cbas, common.HexToAddress(v))
	}

	input := packInput("update", Amount, cbas)
	txHash := sendContractTransaction(conn, from, vm.TeWaKaAddress, nil, priKey, input)
	getResult(conn, txHash, true, false)

	return nil
}

var ConvertCommand = cli.Command{
	Name:   "convert",
	Usage:  "convert",
	Action: utils.MigrateFlags(Convert),
	Flags:  append(TeWakaFlags),
}

func Convert(ctx *cli.Context) error {

	loadPrivate(ctx)
	conn, _ := dialConn(ctx)

	AssetType := big.NewInt(ctx.GlobalInt64(ConvertFlags[0].GetName()))
	TxHash := ctx.GlobalString(ConvertFlags[1].GetName())

	input := packInput("convert", AssetType, TxHash)
	txHash := sendContractTransaction(conn, from, vm.TeWaKaAddress, nil, priKey, input)
	getResult(conn, txHash, true, false)

	return nil
}

var ConfirmCommand = cli.Command{
	Name:   "confirm",
	Usage:  "confirm",
	Action: utils.MigrateFlags(Confirm),
	Flags:  append(TeWakaFlags),
}

func Confirm(ctx *cli.Context) error {

	loadPrivate(ctx)

	conn, _ := dialConn(ctx)

	AssetType := ctx.GlobalUint64(ConfirmFlags[0].GetName())
	ConvertType := ctx.GlobalUint64(ConfirmFlags[1].GetName())
	TxHash := ctx.GlobalString(ConfirmFlags[2].GetName())

	input := packInput("confirm", AssetType, ConvertType, TxHash)
	txHash := sendContractTransaction(conn, from, vm.TeWaKaAddress, nil, priKey, input)
	getResult(conn, txHash, true, false)

	return nil
}

var CastingCommand = cli.Command{
	Name:   "casting",
	Usage:  "casting",
	Action: utils.MigrateFlags(Casting),
	Flags:  append(TeWakaFlags),
}

func Casting(ctx *cli.Context) error {

	loadPrivate(ctx)
	conn, _ := dialConn(ctx)
	ConvertType := ctx.GlobalUint64(CastingFlags[0].GetName())

	Amount := ctx.GlobalInt64(CastingFlags[1].GetName())
	ToAmount := czzToWei(Amount)

	Path := ctx.GlobalStringSlice(CastingFlags[2].GetName())
	ToPath := []common.Address{}
	for _, v := range Path {
		ToPath = append(ToPath, common.HexToAddress(v))
	}

	PubKey := ctx.GlobalString(CastingFlags[3].GetName())
	ToPubKey, _ := hex.DecodeString(PubKey)

	RouterAddr := ctx.GlobalString(CastingFlags[4].GetName())
	ToRouterAddr := common.HexToAddress(RouterAddr)

	Slippage := ctx.GlobalUint64(CastingFlags[5].GetName())
	IsInsurance := ctx.GlobalBool(CastingFlags[6].GetName())

	input := packInput("casting", big.NewInt(int64(ConvertType)), ToAmount, ToPath, ToPubKey, ToRouterAddr, big.NewInt(int64(Slippage)), IsInsurance)
	txHash := sendContractTransaction(conn, from, vm.TeWaKaAddress, nil, priKey, input)
	getResult(conn, txHash, true, false)

	return nil
}
