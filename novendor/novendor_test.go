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
			// package imports vendored package that contains files that declare a "foo" and "main" package,
			// but "main" package is excluded using build constraint. The "foo" package imports another
			// package, which is also vendored. The vendored package should not be reported as unused.
			name: "simple multi-package case",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "main.go",
					Src:     `package main; import _ "github.com/foo";`,
				},
				{
					RelPath: "vendor/github.com/foo/foo.go",
					Src:     `package foo; import _ "github.com/bar"`,
				},
				{
					RelPath: "vendor/github.com/foo/main.go",
					Src: `// +build ignore

package main`,
				},
				{
					RelPath: "vendor/github.com/bar/bar.go",
					Src:     `package bar`,
				},
			},
		},
		{
			// vendored import has 3 different packages in it: "library" (2 files), "main" (1 file) and
			// "library2" (1 file), where "main" and "library" both have ignore build directives and all of
			// these packages vendor different packages. The logic for multi-package build directive parsing
			// should ensure that none of the packages vendored by the 3 different packages are reported as
			// unused.
			name: "complicated case of multi-package vendored import with build constraints",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "main.go",
					Src:     `package main; import _ "github.com/org/library";`,
				},
				{
					RelPath: "vendor/github.com/org/library/library1.go",
					Src:     `package library; import _ "github.com/lib1import"`,
				},
				{
					RelPath: "vendor/github.com/lib1import/import.go",
					Src:     `package lib1import`,
				},
				{
					RelPath: "vendor/github.com/org/library/library1_too.go",
					Src:     `package library; import _ "github.com/anotherlib1import"`,
				},
				{
					RelPath: "vendor/github.com/anotherlib1import/import.go",
					Src:     `package anotherlib1import`,
				},
				{
					RelPath: "vendor/github.com/org/library/main.go",
					Src: `// +build ignore

package main; import _ "github.com/mainimport"`,
				},
				{
					RelPath: "vendor/github.com/mainimport/import.go",
					Src:     `package mainimport`,
				},
				{
					RelPath: "vendor/github.com/org/library/library2.go",
					Src: `// +build ignore

package library2; import _ "github.com/lib2import"`,
				},
				{
					RelPath: "vendor/github.com/lib2import/import.go",
					Src:     `package lib2import`,
				},
			},
		},
		{
			// primary package has 3 different packages in it: "foo" (1 file, 1 test and 1 external test),
			// "main" (1 file) and "other" (1 file), where "main" and "other" both have ignore build
			// directives and all of these packages (and tests) vendor different packages. The logic for
			// multi-package build directive parsing should ensure that none of the packages vendored by the
			// 3 different packages and tests are reported as unused.
			name: "complicated case of multi-package package with build constraints with tests",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package foo; import _ "github.com/fooimport";`,
				},
				{
					RelPath: "foo_ext_test.go",
					Src:     `package foo_test; import _ "github.com/fooexttestimport";`,
				},
				{
					RelPath: "foo_test.go",
					Src:     `package foo; import _ "github.com/footestimport";`,
				},
				{
					RelPath: "main.go",
					Src: `// +build ignore

package main; import _ "github.com/mainimport";`,
				},
				{
					RelPath: "other.go",
					Src: `// +build ignore

package other; import _ "github.com/otherimport";`,
				},
				{
					RelPath: "vendor/github.com/fooimport/fooimport.go",
					Src:     `package fooimport`,
				},
				{
					RelPath: "vendor/github.com/fooexttestimport/fooexttestimport.go",
					Src:     `package fooexttestimport`,
				},
				{
					RelPath: "vendor/github.com/footestimport/footestimport.go",
					Src:     `package footestimport`,
				},
				{
					RelPath: "vendor/github.com/mainimport/mainimport.go",
					Src:     `package mainimport`,
				},
				{
					RelPath: "vendor/github.com/otherimport/otherimport.go",
					Src:     `package otherimport`,
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
		{
			name: "does not consider vendored libraries in hidden directories",
			getArgs: func(projectDir string) (string, []string) {
				return path.Join(projectDir), nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main`,
				},
				{
					RelPath: ".hidden/vendor/github.com/org/library/bar/bar.go",
					Src:     `package bar`,
				},
			},
		},
		{
			name: "considers multiple vendor directories",
			getArgs: func(projectDir string) (string, []string) {
				return path.Join(projectDir), nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main`,
				},
				{
					RelPath: "vendor/github.com/org/library/bar/bar.go",
					Src:     `package bar`,
				},
				{
					RelPath: "subdir/vendor/github.com/org/library/bar/bar.go",
					Src:     `package bar`,
				},
			},
			defaultOutputLines: func(files map[string]gofiles.GoFile) []string {
				return []string{
					"github.com/org/library",
					"github.com/org/library",
				}
			},
			fullOutputLines: func(files map[string]gofiles.GoFile) []string {
				basePkg := files["foo.go"].ImportPath
				return []string{
					path.Join(basePkg, "subdir", "vendor", path.Dir(files["vendor/github.com/org/library/bar/bar.go"].ImportPath)),
					path.Join(basePkg, "vendor", path.Dir(files["vendor/github.com/org/library/bar/bar.go"].ImportPath)),
				}
			},
			noGroupOutputLines: func(files map[string]gofiles.GoFile) []string {
				return []string{
					"github.com/org/library/bar",
					"github.com/org/library/bar",
				}
			},
		},
		{
			name: "ignore specified package and its dependencies",
			getArgs: func(projectDir string) (string, []string) {
				return path.Join(projectDir), []string{
					"-ignore",
					"./vendor/github.com/org/library/bar",
				}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main`,
				},
				{
					RelPath: "vendor/github.com/org/library/bar/bar.go",
					Src:     `package bar; import _ "github.com/org/other-lib/foo";`,
				},
				{
					RelPath: "vendor/github.com/org/other-lib/foo/foo.go",
					Src:     `package foo`,
				},
				{
					RelPath: "subdir/vendor/github.com/org/library/bar/bar.go",
					Src:     `package bar`,
				},
			},
			defaultOutputLines: func(files map[string]gofiles.GoFile) []string {
				return []string{
					"github.com/org/library",
				}
			},
			fullOutputLines: func(files map[string]gofiles.GoFile) []string {
				basePkg := files["foo.go"].ImportPath
				return []string{
					path.Join(basePkg, "subdir", "vendor", path.Dir(files["vendor/github.com/org/library/bar/bar.go"].ImportPath)),
				}
			},
			noGroupOutputLines: func(files map[string]gofiles.GoFile) []string {
				return []string{
					"github.com/org/library/bar",
				}
			},
		},
		{
			name: "ignore multiple packages",
			getArgs: func(projectDir string) (string, []string) {
				return path.Join(projectDir), []string{
					"-ignore",
					"./vendor/github.com/org/library/bar",
					"-ignore",
					"./subdir/vendor/github.com/org/library/bar",
				}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main`,
				},
				{
					RelPath: "vendor/github.com/org/library/bar/bar.go",
					Src:     `package bar; import _ "github.com/org/other-lib/foo";`,
				},
				{
					RelPath: "vendor/github.com/org/other-lib/foo/foo.go",
					Src:     `package foo`,
				},
				{
					RelPath: "subdir/vendor/github.com/org/library/bar/bar.go",
					Src:     `package bar`,
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
	doMainErr := doNovendor(dir, args, group, full, false, &buf)
	expectedOutput := ""
	if f != nil {
		expectedOutput = fmt.Sprintln(strings.Join(f(files), "\n"))
	}
	if expectedOutput == "" {
		assert.NoError(t, doMainErr, "Case %d (%s): %s", caseNum, name, checkType)
	} else {
		assert.Error(t, doMainErr, fmt.Sprintf("Case %d (%s): %s", caseNum, name, checkType))
	}
	assert.Equal(t, expectedOutput, buf.String(), "Case %d (%s): %s\nOutput:\n%s", caseNum, name, checkType, buf.String())
}
