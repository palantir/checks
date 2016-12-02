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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/nmiyake/pkg/dirs"
	"github.com/nmiyake/pkg/gofiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtimport(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tmpDir, cleanup, err := dirs.TempDir(wd, "")
	defer cleanup()
	require.NoError(t, err)

	cases := []struct {
		name          string
		getArgs       func(projectDir string) (string, []string)
		files         []gofiles.GoFileSpec
		verify        func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string)
		listOutput    func(files map[string]gofiles.GoFile) []string
		listAllOutput func(files map[string]gofiles.GoFile) []string
	}{
		{
			name: "standard library imports",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import "fmt"; func main(){}`,
				},
			},
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				assert.NoError(t, err, "Case %d (%s)", caseNum, caseName)
			},
		},
		{
			name: "imports within the same project",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar`,
				},
			},
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				assert.NoError(t, err, "Case %d (%s)", caseNum, caseName)
			},
		},
		{
			name: "vendored imports",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import "{{index . "vendor/github.com/org/product/bar/bar.go"}}";`,
				},
				{
					RelPath: "vendor/github.com/org/product/bar/bar.go",
					Src:     `package bar`,
				},
			},
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				assert.NoError(t, err, "Case %d (%s)", caseNum, caseName)
			},
		},
		{
			// if a project imports a non-external package that uses vendoring to import a package that is visible to the
			// non-external package but not to the base package, that is still a legal import
			name: "multi-level vendored imports",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import "{{index . "bar/vendor/github.com/org/product/baz/baz.go"}}";`,
				},
				{
					RelPath: "bar/vendor/github.com/org/product/baz/baz.go",
					Src:     `package baz`,
				},
			},
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				assert.NoError(t, err, "Case %d (%s)", caseNum, caseName)
			},
		},
		{
			// if a project imports a non-external package that uses an "internal" directory to import a package that is
			// visible to the non-external package but not to the base package, that is still a legal import.
			name: "indirect imports",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import "{{index . "bar/internal/baz/baz.go"}}";`,
				},
				{
					RelPath: "bar/internal/baz/baz.go",
					Src:     `package baz`,
				},
			},
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				assert.NoError(t, err, "Case %d (%s)", caseNum, caseName)
			},
		},
		{
			// vendored dependency has a test with an external dependency
			name: "external package imported transitively via external test",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src:     `package main; import "{{index . "vendor/github.com/org/product/bar/bar.go"}}";`,
				},
				{
					RelPath: "vendor/github.com/org/product/bar/bar.go",
					Src:     `package bar`,
				},
				{
					RelPath: "vendor/github.com/org/product/bar/bar_test.go",
					Src:     `package bar_test; import "{{index . "ext/ext.go"}}";`,
				},
				{
					RelPath: "ext/ext.go",
					Src:     `package ext`,
				},
			},
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				assert.NoError(t, err, "Case %d (%s)", caseNum, caseName)
			},
		},
		{
			name: "error if an external package is imported",
			getArgs: func(projectDir string) (string, []string) {
				return path.Join(projectDir, "foo"), nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src:     `package main; import "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar`,
				},
			},
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				require.Error(t, err, "Case %d (%s)", caseNum, caseName)
				want := fmt.Sprintf("%s:1:22: imports external package %s\n", files["foo/foo.go"].Path, files["bar/bar.go"].ImportPath)
				assert.Equal(t, want, got, "Case %d (%s)", caseNum, caseName)
			},
			listOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					files["bar/bar.go"].ImportPath,
				}
			},
		},
		{
			name: "error if multiple external packages are imported",
			getArgs: func(projectDir string) (string, []string) {
				return path.Join(projectDir, "foo"), nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src:     `package main; import "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "foo/another/foo.go",
					Src:     `package main; import "{{index . "baz/baz.go"}}";`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar`,
				},
				{
					RelPath: "baz/baz.go",
					Src:     `package baz`,
				},
			},
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				require.Error(t, err, "Case %d (%s)", caseNum, caseName)
				want := fmt.Sprintf("%s:1:22: imports external package %s\n", files["foo/foo.go"].Path, files["bar/bar.go"].ImportPath)
				want += fmt.Sprintf("%s:1:22: imports external package %s\n", files["foo/another/foo.go"].Path, files["baz/baz.go"].ImportPath)
				assert.Equal(t, want, got, "Case %d (%s)", caseNum, caseName)
			},
			listOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					files["bar/bar.go"].ImportPath,
					files["baz/baz.go"].ImportPath,
				}
			},
		},
		{
			name: "error if an external package is imported in a test package of the primary project",
			getArgs: func(projectDir string) (string, []string) {
				return path.Join(projectDir, "foo"), nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src:     `package main`,
				},
				{
					RelPath: "foo/foo_test.go",
					Src:     `package main_test; import "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar`,
				},
			},
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				require.Error(t, err, "Case %d (%s)", caseNum, caseName)
				want := fmt.Sprintf("%s:1:27: imports external package %s\n", files["foo/foo_test.go"].Path, files["bar/bar.go"].ImportPath)
				assert.Equal(t, want, got, "Case %d (%s)", caseNum, caseName)
			},
			listOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					files["bar/bar.go"].ImportPath,
				}
			},
		},
		{
			name: "error if an external package is imported transitively (vendored dependency has an external dependency)",
			getArgs: func(projectDir string) (string, []string) {
				return path.Join(projectDir, "foo"), []string{"./."}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src:     `package main; import "{{index . "foo/vendor/github.com/org/product/bar/bar.go"}}";`,
				},
				{
					RelPath: "foo/vendor/github.com/org/product/bar/bar.go",
					Src:     `package bar; import "{{index . "foo/vendor/github.com/org/product/baz/baz.go"}}";`,
				},
				{
					RelPath: "foo/vendor/github.com/org/product/baz/baz.go",
					Src:     `package baz; import "{{index . "ext/ext.go"}}";`,
				},
				{
					RelPath: "ext/ext.go",
					Src:     `package ext`,
				},
			},
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				require.Error(t, err, "Case %d (%s)", caseNum, caseName)
				want := fmt.Sprintf("%s:1:22: imports external package %s transitively via %s -> %s\n", files["foo/foo.go"].Path, files["ext/ext.go"].ImportPath, files["foo/vendor/github.com/org/product/bar/bar.go"].ImportPath, files["foo/vendor/github.com/org/product/baz/baz.go"].ImportPath)
				assert.Equal(t, want, got, "Case %d (%s)", caseNum, caseName)
			},
			listOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					files["ext/ext.go"].ImportPath,
				}
			},
		},
		{
			// external dependency has another external dependency
			name: "error on multi-level external dependency",
			getArgs: func(projectDir string) (string, []string) {
				return path.Join(projectDir, "foo"), []string{"./."}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src:     `package main; import "{{index . "ext/ext.go"}}";`,
				},
				{
					RelPath: "ext/ext.go",
					Src:     `package ext; import "{{index . "other/other.go"}}";`,
				},
				{
					RelPath: "other/other.go",
					Src:     `package other`,
				},
			},
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				require.Error(t, err, "Case %d (%s)", caseNum, caseName)
				want := fmt.Sprintf("%s:1:22: imports external package %s\n", files["foo/foo.go"].Path, files["ext/ext.go"].ImportPath)
				assert.Equal(t, want, got, "Case %d (%s)", caseNum, caseName)
			},
			listOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					files["ext/ext.go"].ImportPath,
				}
			},
			listAllOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					files["ext/ext.go"].ImportPath,
					files["other/other.go"].ImportPath,
				}
			},
		},
		{
			name: "error on multi-level external dependency with intermediate vendored dependency",
			getArgs: func(projectDir string) (string, []string) {
				return path.Join(projectDir, "foo"), []string{"./."}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src:     `package main; import "{{index . "ext/ext.go"}}";`,
				},
				{
					RelPath: "ext/ext.go",
					Src:     `package ext; import "{{index . "foo/vendor/github.com/org/product/bar/bar.go"}}";`,
				},
				{
					RelPath: "foo/vendor/github.com/org/product/bar/bar.go",
					Src:     `package bar; import "{{index . "other/other.go"}}";`,
				},
				{
					RelPath: "other/other.go",
					Src:     `package other`,
				},
			},
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				require.Error(t, err, "Case %d (%s)", caseNum, caseName)
				want := fmt.Sprintf("%s:1:22: imports external package %s\n", files["foo/foo.go"].Path, files["ext/ext.go"].ImportPath)
				assert.Equal(t, want, got, "Case %d (%s)", caseNum, caseName)
			},
			listOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					files["ext/ext.go"].ImportPath,
				}
			},
			listAllOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					files["ext/ext.go"].ImportPath,
					files["other/other.go"].ImportPath,
				}
			},
		},
		{
			name: "error on external package imported transitively via different paths",
			getArgs: func(projectDir string) (string, []string) {
				return path.Join(projectDir, "foo"), []string{"./."}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src:     `package main; import "{{index . "foo/vendor/github.com/org/product/bar/bar.go"}}"; import "{{index . "foo/vendor/github.com/org/product/baz/baz.go"}}";`,
				},
				{
					RelPath: "foo/vendor/github.com/org/product/bar/bar.go",
					Src:     `package bar; import "{{index . "ext/ext.go"}}";`,
				},
				{
					RelPath: "foo/vendor/github.com/org/product/baz/baz.go",
					Src:     `package baz; import "{{index . "ext/ext.go"}}";`,
				},
				{
					RelPath: "ext/ext.go",
					Src:     `package ext; import "{{index . "foo/vendor/github.com/org/product/bar/bar.go"}}";`,
				},
			},
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				require.Error(t, err, "Case %d (%s)", caseNum, caseName)
				want := fmt.Sprintf("%s:1:22: imports external package %s transitively via %s\n", files["foo/foo.go"].Path, files["ext/ext.go"].ImportPath, files["foo/vendor/github.com/org/product/bar/bar.go"].ImportPath)
				want += fmt.Sprintf("%s:1:59: imports external package %s transitively via %s\n", files["foo/foo.go"].Path, files["ext/ext.go"].ImportPath, files["foo/vendor/github.com/org/product/baz/baz.go"].ImportPath)
				assert.Equal(t, want, got, "Case %d (%s)", caseNum, caseName)
			},
			listOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					files["ext/ext.go"].ImportPath,
				}
			},
		},
	}

	for i, currCase := range cases {
		currTmpDir, err := ioutil.TempDir(tmpDir, "")
		require.NoError(t, err)

		files, err := gofiles.Write(currTmpDir, currCase.files)
		require.NoError(t, err)

		dir, args := currCase.getArgs(currTmpDir)

		buf := bytes.Buffer{}
		doMainErr := doExtimport(dir, args, false, false, &buf)
		currCase.verify(files, buf.String(), doMainErr, i, currCase.name)

		if currCase.listOutput != nil {
			buf := bytes.Buffer{}
			_ = doExtimport(dir, args, true, false, &buf)
			assert.Equal(t, strings.Join(currCase.listOutput(files), "\n")+"\n", buf.String(), "Case %d (%s)", i, currCase.name)

			listAllOutputFunc := currCase.listAllOutput
			if listAllOutputFunc == nil {
				listAllOutputFunc = currCase.listOutput
			}
			buf = bytes.Buffer{}
			_ = doExtimport(dir, args, true, true, &buf)
			assert.Equal(t, strings.Join(listAllOutputFunc(files), "\n")+"\n", buf.String(), "Case %d (%s)", i, currCase.name)
		}
	}
}
