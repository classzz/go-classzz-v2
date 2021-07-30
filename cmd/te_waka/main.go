package main

import (
	"fmt"
	"github.com/classzz/go-classzz-v2/cmd/utils"
	"github.com/classzz/go-classzz-v2/internal/flags"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
	"sort"
)

var (
	// Git SHA1 commit hash of the release (set via linker flags)
	gitCommit = ""
	gitDate   = ""
	// The app that holds all commands and flags.
	app *cli.App

	// Flags needed by abigen
	KeyFlag = cli.StringFlag{
		Name:  "key",
		Usage: "Private key file path",
		Value: "",
	}
	KeyStoreFlag = cli.StringFlag{
		Name:  "keystore",
		Usage: "Keystore file path",
	}
	CzzValueFlag = cli.Uint64Flag{
		Name:  "value",
		Usage: "Staking value units one true",
		Value: 0,
	}
	AddressFlag = cli.StringFlag{
		Name:  "address",
		Usage: "Transfer address",
		Value: "",
	}
	TxHashFlag = cli.StringFlag{
		Name:  "txhash",
		Usage: "Input transaction hash",
		Value: "",
	}
	PubKeyKeyFlag = cli.StringFlag{
		Name:  "pubkey",
		Usage: "Committee public key for BFT (no 0x prefix)",
		Value: "",
	}
	BFTKeyKeyFlag = cli.StringFlag{
		Name:  "bftkey",
		Usage: "Committee bft key for BFT (no 0x prefix)",
		Value: "",
	}

	MortgageFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "mortgage.toaddress",
			Usage: "Committee bft key for BFT (no 0x prefix)",
			Value: "",
		},
		cli.StringFlag{
			Name:  "mortgage.stakingamount",
			Usage: "Committee bft key for BFT (no 0x prefix)",
			Value: "",
		},
		cli.StringFlag{
			Name:  "mortgage.coinbaseaddress",
			Usage: "Committee bft key for BFT (no 0x prefix)",
			Value: "",
		},
	}

	TeWakaFlags = []cli.Flag{
		KeyFlag,
		KeyStoreFlag,
		utils.LegacyRPCListenAddrFlag,
		utils.LegacyRPCPortFlag,
		CzzValueFlag,
		PubKeyKeyFlag,
		BFTKeyKeyFlag,
		MortgageFlags[0],
		MortgageFlags[1],
		MortgageFlags[2],
	}
)

func init() {
	app = cli.NewApp()
	app.Usage = "Classzz Tewaka tool"
	app.Name = filepath.Base(os.Args[0])
	app.Version = "1.0.0"
	app.Copyright = "Copyright 2019-2021 The Classzz Authors"
	app.Flags = TeWakaFlags
	//app.Action = utils.MigrateFlags(impawn)
	//app.CommandNotFound = func(ctx *cli.Context, cmd string) {
	//	fmt.Fprintf(os.Stderr, "No such command: %s\n", cmd)
	//	os.Exit(1)
	//}
	// Add subcommands.
	app.Commands = []cli.Command{
		MortgageCommand,
	}
	cli.CommandHelpTemplate = flags.CommandHelpTemplate
	sort.Sort(cli.CommandsByName(app.Commands))
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
