// Copyright 2017 The go-classzz-v2 Authors
// This file is part of go-classzz-v2.
//
// go-classzz-v2 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-classzz-v2 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-classzz-v2. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"reflect"
	"unicode"

	"gopkg.in/urfave/cli.v1"

	"github.com/classzz/go-classzz-v2/cmd/utils"
	"github.com/classzz/go-classzz-v2/czz/catalyst"
	"github.com/classzz/go-classzz-v2/czz/czzconfig"
	"github.com/classzz/go-classzz-v2/internal/czzapi"
	"github.com/classzz/go-classzz-v2/log"
	"github.com/classzz/go-classzz-v2/metrics"
	"github.com/classzz/go-classzz-v2/node"
	"github.com/classzz/go-classzz-v2/params"
	"github.com/naoina/toml"
)

var (
	dumpConfigCommand = cli.Command{
		Action:      utils.MigrateFlags(dumpConfig),
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Flags:       append(nodeFlags, rpcFlags...),
		Category:    "MISCELLANEOUS COMMANDS",
		Description: `The dumpconfig command shows configuration values.`,
	}

	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}
)

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		id := fmt.Sprintf("%s.%s", rt.String(), field)
		if deprecated(id) {
			log.Warn("Config field is deprecated and won't have an effect", "name", id)
			return nil
		}
		var link string
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

type czzstatsConfig struct {
	URL string `toml:",omitempty"`
}

type gczzConfig struct {
	Czz      czzconfig.Config
	Node     node.Config
	Czzstats czzstatsConfig
	Metrics  metrics.Config
}

func loadConfig(file string, cfg *gczzConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tomlSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// Add file name to errors that have a line number.
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	return err
}

func defaultNodeConfig() node.Config {
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit(gitCommit, gitDate)
	cfg.HTTPModules = append(cfg.HTTPModules, "czz")
	cfg.WSModules = append(cfg.WSModules, "czz")
	cfg.IPCPath = "gczz.ipc"
	return cfg
}

// makeConfigNode loads gczz configuration and creates a blank node instance.
func makeConfigNode(ctx *cli.Context) (*node.Node, gczzConfig) {
	// Load defaults.
	cfg := gczzConfig{
		Czz:     czzconfig.Defaults,
		Node:    defaultNodeConfig(),
		Metrics: metrics.DefaultConfig,
	}

	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Apply flags.
	utils.SetNodeConfig(ctx, &cfg.Node)
	stack, err := node.New(&cfg.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	utils.SetEthConfig(ctx, stack, &cfg.Czz)
	if ctx.GlobalIsSet(utils.EthStatsURLFlag.Name) {
		cfg.Czzstats.URL = ctx.GlobalString(utils.EthStatsURLFlag.Name)
	}
	applyMetricConfig(ctx, &cfg)

	return stack, cfg
}

// makeFullNode loads gczz configuration and creates the Classzz backend.
func makeFullNode(ctx *cli.Context) (*node.Node, czzapi.Backend) {
	stack, cfg := makeConfigNode(ctx)
	//if ctx.GlobalIsSet(utils.OverrideLondonFlag.Name) {
	//	cfg.Czz.OverrideLondon = new(big.Int).SetUint64(ctx.GlobalUint64(utils.OverrideLondonFlag.Name))
	//}
	backend, czz := utils.RegisterEthService(stack, &cfg.Czz)

	// Configure catalyst.
	if ctx.GlobalBool(utils.CatalystFlag.Name) {
		if czz == nil {
			utils.Fatalf("Catalyst does not work in light client mode.")
		}
		if err := catalyst.Register(stack, czz); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Configure GraphQL if requested
	if ctx.GlobalIsSet(utils.GraphQLEnabledFlag.Name) {
		utils.RegisterGraphQLService(stack, backend, cfg.Node)
	}
	// Add the Classzz Stats daemon if requested.
	if cfg.Czzstats.URL != "" {
		utils.RegisterEthStatsService(stack, backend, cfg.Czzstats.URL)
	}
	return stack, backend
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	_, cfg := makeConfigNode(ctx)
	comment := ""

	if cfg.Czz.Genesis != nil {
		cfg.Czz.Genesis = nil
		comment += "# Note: this config doesn't contain the genesis block.\n\n"
	}

	out, err := tomlSettings.Marshal(&cfg)
	if err != nil {
		return err
	}

	dump := os.Stdout
	if ctx.NArg() > 0 {
		dump, err = os.OpenFile(ctx.Args().Get(0), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer dump.Close()
	}
	dump.WriteString(comment)
	dump.Write(out)

	return nil
}

func applyMetricConfig(ctx *cli.Context, cfg *gczzConfig) {
	if ctx.GlobalIsSet(utils.MetricsEnabledFlag.Name) {
		cfg.Metrics.Enabled = ctx.GlobalBool(utils.MetricsEnabledFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsEnabledExpensiveFlag.Name) {
		cfg.Metrics.EnabledExpensive = ctx.GlobalBool(utils.MetricsEnabledExpensiveFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsHTTPFlag.Name) {
		cfg.Metrics.HTTP = ctx.GlobalString(utils.MetricsHTTPFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsPortFlag.Name) {
		cfg.Metrics.Port = ctx.GlobalInt(utils.MetricsPortFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsEnableInfluxDBFlag.Name) {
		cfg.Metrics.EnableInfluxDB = ctx.GlobalBool(utils.MetricsEnableInfluxDBFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBEndpointFlag.Name) {
		cfg.Metrics.InfluxDBEndpoint = ctx.GlobalString(utils.MetricsInfluxDBEndpointFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBDatabaseFlag.Name) {
		cfg.Metrics.InfluxDBDatabase = ctx.GlobalString(utils.MetricsInfluxDBDatabaseFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBUsernameFlag.Name) {
		cfg.Metrics.InfluxDBUsername = ctx.GlobalString(utils.MetricsInfluxDBUsernameFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBPasswordFlag.Name) {
		cfg.Metrics.InfluxDBPassword = ctx.GlobalString(utils.MetricsInfluxDBPasswordFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBTagsFlag.Name) {
		cfg.Metrics.InfluxDBTags = ctx.GlobalString(utils.MetricsInfluxDBTagsFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsEnableInfluxDBV2Flag.Name) {
		cfg.Metrics.EnableInfluxDBV2 = ctx.GlobalBool(utils.MetricsEnableInfluxDBV2Flag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBTokenFlag.Name) {
		cfg.Metrics.InfluxDBToken = ctx.GlobalString(utils.MetricsInfluxDBTokenFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBBucketFlag.Name) {
		cfg.Metrics.InfluxDBBucket = ctx.GlobalString(utils.MetricsInfluxDBBucketFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBOrganizationFlag.Name) {
		cfg.Metrics.InfluxDBOrganization = ctx.GlobalString(utils.MetricsInfluxDBOrganizationFlag.Name)
	}
}

func deprecated(field string) bool {
	switch field {
	case "czzconfig.Config.EVMInterpreter":
		return true
	case "czzconfig.Config.EWASMInterpreter":
		return true
	default:
		return false
	}
}
