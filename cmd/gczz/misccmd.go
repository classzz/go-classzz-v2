// Copyright 2016 The go-classzz-v2 Authors
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
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/classzz/go-classzz-v2/cmd/utils"
	"github.com/classzz/go-classzz-v2/params"
	"gopkg.in/urfave/cli.v1"
)

var (
	VersionCheckUrlFlag = cli.StringFlag{
		Name:  "check.url",
		Usage: "URL to use when checking vulnerabilities",
		Value: "https://gczz.classzz.org/docs/vulnerabilities/vulnerabilities.json",
	}
	VersionCheckVersionFlag = cli.StringFlag{
		Name:  "check.version",
		Usage: "Version to check",
		Value: fmt.Sprintf("Gczz/v%v/%v-%v/%v",
			params.VersionWithCommit(gitCommit, gitDate),
			runtime.GOOS, runtime.GOARCH, runtime.Version()),
	}

	versionCommand = cli.Command{
		Action:    utils.MigrateFlags(version),
		Name:      "version",
		Usage:     "Print version numbers",
		ArgsUsage: " ",
		Category:  "MISCELLANEOUS COMMANDS",
		Description: `
The output of this command is supposed to be machine-readable.
`,
	}
	versionCheckCommand = cli.Command{
		Action: utils.MigrateFlags(versionCheck),
		Flags: []cli.Flag{
			VersionCheckUrlFlag,
			VersionCheckVersionFlag,
		},
		Name:      "version-check",
		Usage:     "Checks (online) whether the current version suffers from any known security vulnerabilities",
		ArgsUsage: "<versionstring (optional)>",
		Category:  "MISCELLANEOUS COMMANDS",
		Description: `
The version-check command fetches vulnerability-information from https://gczz.classzz.org/docs/vulnerabilities/vulnerabilities.json, 
and displays information about any security vulnerabilities that affect the currently executing version.
`,
	}
	licenseCommand = cli.Command{
		Action:    utils.MigrateFlags(license),
		Name:      "license",
		Usage:     "Display license information",
		ArgsUsage: " ",
		Category:  "MISCELLANEOUS COMMANDS",
	}
)

func version(ctx *cli.Context) error {
	fmt.Println(strings.Title(clientIdentifier))
	fmt.Println("Version:", params.VersionWithMeta)
	if gitCommit != "" {
		fmt.Println("Git Commit:", gitCommit)
	}
	if gitDate != "" {
		fmt.Println("Git Commit Date:", gitDate)
	}
	fmt.Println("Architecture:", runtime.GOARCH)
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("Operating System:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())
	return nil
}

func license(_ *cli.Context) error {
	fmt.Println(`Gczz is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Gczz is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with gczz. If not, see <http://www.gnu.org/licenses/>.`)
	return nil
}
