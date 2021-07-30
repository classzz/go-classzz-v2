package main

import (
	"github.com/classzz/go-classzz-v2/cmd/utils"
	"github.com/classzz/go-classzz-v2/common"
	"github.com/classzz/go-classzz-v2/core/vm"
	"gopkg.in/urfave/cli.v1"
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

	toAddress := ctx.GlobalString(MortgageFlags[0].GetName())
	coinBaseAddress := ctx.GlobalString(MortgageFlags[2].GetName())
	cbas := []common.Address{common.HexToAddress(coinBaseAddress)}

	printBaseInfo(conn, url)
	PrintBalance(conn, from, common.HexToAddress(toAddress))

	stakingAmount := ctx.GlobalUint64(MortgageFlags[1].GetName())
	Amount := czzToWei(stakingAmount)
	if stakingAmount < TeWakaAmount {
		printError("mortgage value must bigger than ", TeWakaAmount)
	}

	input := packInput("mortgage", common.HexToAddress(toAddress), Amount, cbas)
	txHash := sendContractTransaction(conn, from, vm.TeWaKaAddress, nil, priKey, input)
	getResult(conn, txHash, true, false)

	return nil
}
