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

	MortgageFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "mortgage.toaddress",
			Usage: "",
			Value: "",
		},
		cli.StringFlag{
			Name:  "mortgage.stakingamount",
			Usage: "",
			Value: "",
		},
		cli.StringFlag{
			Name:  "mortgage.coinbaseaddress",
			Usage: "",
			Value: "",
		},
	}

	ConvertFlags = []cli.Flag{
		cli.Uint64Flag{
			Name:  "convert.assettype",
			Usage: "",
			Value: 0,
		},
		cli.StringFlag{
			Name:  "convert.txhash",
			Usage: "",
			Value: "",
		},
	}

	ConfirmFlags = []cli.Flag{
		cli.Uint64Flag{
			Name:  "confirm.assettype",
			Usage: "",
			Value: 0,
		},
		cli.Uint64Flag{
			Name:  "confirm.converttype",
			Usage: "",
			Value: 0,
		},
		cli.StringFlag{
			Name:  "confirm.txhash",
			Usage: "",
			Value: "",
		},
	}

	CastingFlags = []cli.Flag{
		cli.Uint64Flag{
			Name:  "casting.converttype",
			Usage: "",
			Value: 0,
		},
		cli.Uint64Flag{
			Name:  "casting.amount",
			Usage: "",
			Value: 0,
		},
		cli.StringFlag{
			Name:  "casting.totoken",
			Usage: "",
			Value: "",
		},
		cli.StringFlag{
			Name:  "casting.pubkey",
			Usage: "",
			Value: "",
		},
		cli.Uint64Flag{
			Name:  "casting.slippage",
			Usage: "",
		},
		cli.StringFlag{
			Name:  "casting.routeraddr",
			Usage: "",
			Value: "",
		},
		cli.BoolFlag{
			Name:  "casting.isinsurance",
			Usage: "",
		},
	}

	TeWakaFlags = []cli.Flag{
		KeyFlag,
		KeyStoreFlag,
		utils.LegacyRPCListenAddrFlag,
		utils.LegacyRPCPortFlag,
		CzzValueFlag,

		MortgageFlags[0],
		MortgageFlags[1],
		MortgageFlags[2],

		ConvertFlags[0],
		ConvertFlags[1],

		ConfirmFlags[0],
		ConfirmFlags[1],
		ConfirmFlags[2],

		CastingFlags[0],
		CastingFlags[1],
		CastingFlags[2],
		CastingFlags[3],
		CastingFlags[4],
		CastingFlags[5],
	}
)

func init() {
	app = cli.NewApp()
	app.Usage = "Classzz Tewaka tool"
	app.Name = filepath.Base(os.Args[0])
	app.Version = "1.0.0"
	app.Copyright = "Copyright 2019-2021 The Classzz Authors"
	app.Flags = TeWakaFlags

	// Add subcommands.
	app.Commands = []cli.Command{
		MortgageCommand,
		UpdateCommand,
		ConvertCommand,
		ConfirmCommand,
		CastingCommand,
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
