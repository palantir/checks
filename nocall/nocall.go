// Copyright 2016 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"os"

	"github.com/nmiyake/pkg/errorstringer"
	"github.com/palantir/pkg/cli"
	"github.com/palantir/pkg/cli/flag"
	"github.com/pkg/errors"

	"github.com/palantir/checks/nocall/nocall"
)

const (
	printAllFlagName   = "all"
	jsonConfigFlagName = "json"
	pkgsFlagName       = "pkgs"
)

var (
	printAllFlag = flag.BoolFlag{
		Name:  printAllFlagName,
		Usage: "print all function references",
	}
	jsonFlag = flag.StringFlag{
		Name:  jsonConfigFlagName,
		Usage: "JSON configuration specifying blacklisted functions",
	}
	pkgsFlag = flag.StringSlice{
		Name:  pkgsFlagName,
		Usage: "paths to the packages to check",
	}
)

func main() {
	app := cli.NewApp(cli.DebugHandler(errorstringer.SingleStack))
	app.Flags = append(
		app.Flags,
		printAllFlag,
		jsonFlag,
		pkgsFlag,
	)
	app.Action = func(ctx cli.Context) error {
		var jsonConfig map[string]string
		if ctx.Has(jsonConfigFlagName) {
			if err := json.Unmarshal([]byte(ctx.String(jsonConfigFlagName)), &jsonConfig); err != nil {
				return errors.Wrapf(err, "failed to read configuration")
			}
		}

		if len(jsonConfig) == 0 || ctx.Bool(printAllFlagName) {
			if err := nocall.PrintAllFuncRefs(ctx.Slice(pkgsFlagName), ctx.App.Stdout); err != nil {
				return errors.Wrapf(err, "Failed to determine all function references")
			}
			return nil
		}

		if err := nocall.PrintFuncRefUsages(ctx.Slice(pkgsFlagName), jsonConfig, ctx.App.Stdout); err != nil {
			return errors.Wrapf(err, "nocall failed")
		}
		return nil
	}
	os.Exit(app.Run(os.Args))
}
