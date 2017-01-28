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
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nmiyake/pkg/dirs"
	"github.com/nmiyake/pkg/errorstringer"
	"github.com/palantir/pkg/cli"
	"github.com/palantir/pkg/cli/flag"
	"github.com/palantir/pkg/pkgpath"
	"github.com/pkg/errors"
)

const (
	pkgsFlagName = "pkgs"
)

var (
	pkgsFlag = flag.StringSlice{
		Name:     pkgsFlagName,
		Usage:    "paths to the packages to check",
		Optional: true,
	}
)

func main() {
	app := cli.NewApp(cli.DebugHandler(errorstringer.SingleStack))
	app.Flags = append(app.Flags,
		pkgsFlag,
	)
	app.Action = func(ctx cli.Context) error {
		wd, err := dirs.GetwdEvalSymLinks()
		if err != nil {
			return errors.Wrapf(err, "Failed to get working directory")
		}
		return doImportAlias(wd, ctx.Slice(pkgsFlagName), ctx.App.Stdout)
	}
	os.Exit(app.Run(os.Args))
}

func doImportAlias(projectDir string, pkgPaths []string, w io.Writer) error {
	if !path.IsAbs(projectDir) {
		return errors.Errorf("projectDir %s must be an absolute path", projectDir)
	}

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		return errors.Errorf("GOPATH environment variable must be set")
	}

	if relPath, err := filepath.Rel(path.Join(gopath, "src"), projectDir); err != nil || strings.HasPrefix(relPath, "../") {
		return errors.Wrapf(err, "Project directory %s must be a subdirectory of $GOPATH/src (%s)", projectDir, path.Join(gopath, "src"))
	}

	if len(pkgPaths) == 0 {
		pkgs, err := pkgpath.PackagesInDir(projectDir, pkgpath.DefaultGoPkgExcludeMatcher())
		if err != nil {
			return errors.Wrapf(err, "Failed to list packages")
		}

		pkgPaths, err = pkgs.Paths(pkgpath.Relative)
		if err != nil {
			return errors.Wrapf(err, "Failed to convert package paths")
		}
	}

	// package import path -> alias -> files that import using alias
	imports := make(map[string]map[string][]string)

	for _, pkgPath := range pkgPaths {
		currPath := path.Join(projectDir, pkgPath)
		fis, err := ioutil.ReadDir(currPath)
		if err != nil {
			return errors.Wrapf(err, "Failed to list contents of directory %s", currPath)
		}
		for _, fi := range fis {
			if !fi.IsDir() && strings.HasSuffix(fi.Name(), ".go") {
				currFile := path.Join(currPath, fi.Name())
				fileImports, err := processFile(currFile)
				if err != nil {
					return errors.Wrapf(err, "Failed to process file %s", currFile)
				}
				for k, v := range fileImports {
					if v == "_" || v == "." {
						// do not record "_" or "." aliases
						continue
					}

					// if package path is not in imports map, allocate map
					if _, ok := imports[k]; !ok {
						imports[k] = make(map[string][]string)
					}
					innerMap := imports[k]
					innerMap[v] = append(innerMap[v], path.Join(pkgPath, fi.Name()))
				}
			}
		}
	}

	var pkgsWithMultipleAliases []string
	for k := range imports {
		if len(imports[k]) > 1 {
			// package is imported using more than 1 alias
			pkgsWithMultipleAliases = append(pkgsWithMultipleAliases, k)
			for _, vv := range imports[k] {
				sort.Strings(vv)
			}
		}
	}
	sort.Strings(pkgsWithMultipleAliases)
	if len(pkgsWithMultipleAliases) > 0 {
		var output []string
		for _, k := range pkgsWithMultipleAliases {
			var sortedAliases []string
			aliasToFile := imports[k]
			for kk := range aliasToFile {
				sortedAliases = append(sortedAliases, kk)
			}
			sort.Strings(sortedAliases)

			output = append(output, fmt.Sprintf("%s is imported using multiple different aliases:", k))
			for _, currAlias := range sortedAliases {
				output = append(output, fmt.Sprintf("\t%s:\n\t\t%s", currAlias, strings.Join(aliasToFile[currAlias], "\n\t\t")))
			}
		}
		return errors.New(strings.Join(output, "\n"))
	}
	return nil
}

// processFile returns a map from all of the import paths in the file to the alias used for that import.
func processFile(filename string) (map[string]string, error) {
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse file %s", filename)
	}

	aliasMap := make(map[string]string)
	var visitor visitFn
	visitor = visitFn(func(node ast.Node) ast.Visitor {
		if node == nil {
			return visitor
		}
		switch v := node.(type) {
		case *ast.ImportSpec:
			if v.Name != nil {
				// import has alias: record
				aliasMap[v.Path.Value] = v.Name.Name
				break
			}
		}
		return visitor
	})
	ast.Walk(visitor, file)
	return aliasMap, nil
}

type visitFn func(node ast.Node) ast.Visitor

func (fn visitFn) Visit(node ast.Node) ast.Visitor {
	return fn(node)
}
