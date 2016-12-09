// Copyright 2013 Kamil Kisiel
// Modifications copyright 2016 Palantir Technologies, Inc.
// Licensed under the MIT License. See LICENSE in the project root
// for license information.

package outparamcheck

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/loader"
)

const prog = `
package main

import (
	"encoding/json"
)

func main() {
	j := []byte("...")
	var x interface{}
	json.Unmarshal(j, x)
	json.Unmarshal(j, &x)
	json.Unmarshal(j, *&x)
	json.Unmarshal(j, nil)
}
`

func TestOutParamCheck(t *testing.T) {
	// write program to temp file
	tmpf, cleanup := writeTempFile(t, prog)
	defer cleanup()

	// parse program
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, tmpf, prog, 0)
	require.NoError(t, err)

	// type information will be populated by type checker
	info := types.Info{
		Types: map[ast.Expr]types.TypeAndValue{},
		Uses:  map[*ast.Ident]types.Object{},
	}

	// hypothetical package
	packagePath := "github.com/palantir/checks/outparamcheck"
	packageName := "main"
	pkg := types.NewPackage(packagePath, packageName)

	// run type checker
	cfg := &types.Config{
		Importer: importer.For("gc", nil),
	}
	files := []*ast.File{file}
	err = types.NewChecker(cfg, fset, pkg, &info).Files(files)
	require.NoError(t, err)

	// assert type information filled in
	assert.NotEqual(t, 0, len(info.Types))
	assert.NotEqual(t, 0, len(info.Uses))

	// run out-param checker
	errs := run(&loader.Program{
		Fset: fset,
		Created: []*loader.PackageInfo{{
			Pkg:   pkg,
			Files: files,
			Info:  info,
		}},
	}, defaultCfg)

	// there should be one failure
	expected := []OutParamError{
		{
			Pos: token.Position{
				Filename: tmpf,
				Offset:   116,
				Line:     11,
				Column:   20,
			},
			Line:     `json.Unmarshal(j, x)`,
			Method:   "Unmarshal",
			Argument: 1,
		},
	}
	assert.Equal(t, expected, errs)
}

func writeTempFile(t *testing.T, contents string) (path string, cleanup func()) {
	tmpf, err := ioutil.TempFile("", "")
	require.NoError(t, err, "failed to create temp file")

	cleanup = func() {
		err = os.Remove(tmpf.Name())
		assert.NoError(t, err, "failed to remove temp file")
	}

	remove := true
	defer func() {
		err = tmpf.Close()
		assert.NoError(t, err, "failed to close temp file")
		if remove {
			cleanup()
		}
	}()

	_, err = tmpf.WriteString(contents)
	require.NoError(t, err, "failed to write temp file")

	remove = false
	return tmpf.Name(), cleanup
}
