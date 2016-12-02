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

func TestNovendor(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tmpDir, cleanup, err := dirs.TempDir(wd, "")
	defer cleanup()
	require.NoError(t, err)

	for i, currCase := range []struct {
		name               string
		getArgs            func(projectDir string) (string, []string)
		files              []gofiles.GoFileSpec
		defaultOutputLines func(files map[string]gofiles.GoFile) []string
		fullOutputLines    func(files map[string]gofiles.GoFile) []string
		noGroupOutputLines func(files map[string]gofiles.GoFile) []string
	}{
		{
			name: "package with no dependencies",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main`,
				},
			},
		},
		{
			name: "package with vendored import that is used",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import _ "{{index . "vendor/github.com/org/product/bar/bar.go"}}";`,
				},
				{
					RelPath: "vendor/github.com/org/product/bar/bar.go",
					Src:     `package bar`,
				},
			},
		},
		{
			name: "multi-level vendored imports: import a non-external package that uses vendoring to import a package that is visible to the non-external package but not to the base package",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import _ "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import _ "{{index . "bar/vendor/github.com/org/product/baz/baz.go"}}";`,
				},
				{
					RelPath: "bar/vendor/github.com/org/product/baz/baz.go",
					Src:     `package baz`,
				},
			},
		},
		{
			name: "unused vendored package causes error",
			getArgs: func(projectDir string) (string, []string) {
				return path.Join(projectDir), nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main`,
				},
				{
					RelPath: "vendor/github.com/org/library/subpackage/bar.go",
					Src:     `package bar`,
				},
			},
			defaultOutputLines: func(files map[string]gofiles.GoFile) []string {
				return []string{
					path.Dir(files["vendor/github.com/org/library/subpackage/bar.go"].ImportPath),
				}
			},
			fullOutputLines: func(files map[string]gofiles.GoFile) []string {
				basePkg := files["foo.go"].ImportPath
				return []string{
					path.Join(basePkg, "vendor", path.Dir(files["vendor/github.com/org/library/subpackage/bar.go"].ImportPath)),
				}
			},
			noGroupOutputLines: func(files map[string]gofiles.GoFile) []string {
				return []string{
					files["vendor/github.com/org/library/subpackage/bar.go"].ImportPath,
				}
			},
		},
		{
			name: "one subpackage of a vendored library is used but another is not",
			getArgs: func(projectDir string) (string, []string) {
				return path.Join(projectDir), nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import _ "{{index . "vendor/github.com/org/library/subpackage/bar.go"}}";`,
				},
				{
					RelPath: "vendor/github.com/org/library/subpackage/bar.go",
					Src:     `package bar`,
				},
				{
					RelPath: "vendor/github.com/org/library/subpackage-unused/baz.go",
					Src:     `package baz`,
				},
			},
			// group is used, so default output is empty
			defaultOutputLines: nil,
			fullOutputLines:    nil,
			noGroupOutputLines: func(files map[string]gofiles.GoFile) []string {
				return []string{
					files["vendor/github.com/org/library/subpackage-unused/baz.go"].ImportPath,
				}
			},
		},
		{
			name: "computes dependencies with all build tags as true",
			getArgs: func(projectDir string) (string, []string) {
				return path.Join(projectDir), nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					// imports the "github.com/org/library/bar package
					Src: `package main; import _ "{{index . "vendor/github.com/org/library/bar/bar_d.go"}}";`,
				},
				// build constraints makes it such that using default build context will import either "github.com/org/library/subpackage-darwin"
				// or "github.com/org/library/subpackage-linux", but not both.
				{
					RelPath: "vendor/github.com/org/library/bar/bar_d.go",
					Src: `// +build darwin

package bar; import _ "{{index . "vendor/github.com/org/library/subpackage_darwin/darwin_go_pkg.go"}}";`,
				},
				{
					RelPath: "vendor/github.com/org/library/bar/bar_l.go",
					Src: `// +build linux

package bar; import _ "{{index . "vendor/github.com/org/library/subpackage_linux/linux_go_pkg.go"}}";`,
				},
				{
					RelPath: "vendor/github.com/org/library/subpackage_darwin/darwin_go_pkg.go",
					Src:     `package red`,
				},
				{
					RelPath: "vendor/github.com/org/library/subpackage_linux/linux_go_pkg.go",
					Src:     `package blue`,
				},
			},
		},
	} {
		currTmpDir, err := ioutil.TempDir(tmpDir, "")
		require.NoError(t, err, "Case %d (%s)", i, currCase.name)

		files, err := gofiles.Write(currTmpDir, currCase.files)
		require.NoError(t, err, "Case %d (%s)", i, currCase.name)

		dir, args := currCase.getArgs(currTmpDir)

		verifyDoMain(t, i, currCase.name, dir, args, true, false, "default output lines", currCase.defaultOutputLines, files)
		verifyDoMain(t, i, currCase.name, dir, args, true, true, "full output lines", currCase.fullOutputLines, files)
		verifyDoMain(t, i, currCase.name, dir, args, false, false, "no group output lines", currCase.noGroupOutputLines, files)
	}
}

func verifyDoMain(t *testing.T, caseNum int, name, dir string, args []string, group, full bool, checkType string, f func(map[string]gofiles.GoFile) []string, files map[string]gofiles.GoFile) {
	buf := bytes.Buffer{}
	doMainErr := doNovendor(dir, args, group, full, &buf)
	expectedOutput := ""
	if f != nil {
		expectedOutput = fmt.Sprintln(strings.Join(f(files), "\n"))
	}
	if expectedOutput == "" {
		assert.NoError(t, doMainErr, "Case %d (%s): %s", caseNum, name, checkType)
	} else {
		assert.Error(t, doMainErr, fmt.Sprintf("Case %d (%s): %s", caseNum, name, checkType))
	}
	assert.Equal(t, expectedOutput, buf.String(), "Case %d (%s): %s", caseNum, name, checkType)
}
