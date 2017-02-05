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
	"strings"
	"testing"

	"github.com/nmiyake/pkg/dirs"
	"github.com/nmiyake/pkg/gofiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportAliasNoError(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tmpDir, cleanup, err := dirs.TempDir(wd, "")
	defer cleanup()
	require.NoError(t, err)

	cases := []struct {
		name    string
		getArgs func(projectDir string) (string, []string)
		files   []gofiles.GoFileSpec
	}{
		{
			name: "no error",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
			},
		},
		{
			name: "no error for multiple files that use the same alias for an import",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import foo "fmt"; func Bar(){ foo.Println() }`,
				},
			},
		},
		{
			name: "no error if multiple files import same package with one using alias and other not using an alias",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import "fmt"; func Bar(){ fmt.Println() }`,
				},
			},
		},
		{
			name: "no error if multiple files import same package with one using alias and other using _",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import _ "fmt"; func Bar(){}`,
				},
			},
		},
		{
			name: "no error if multiple files import same package with one using alias and other using .",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import . "fmt"; func Bar(){}`,
				},
			},
		},
		{
			name: "no error if multiple files import diffefrent packages using the same alias",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import foo "io"; func Bar(){ var w foo.Writer; _ = w }`,
				},
			},
		},
	}

	for i, currCase := range cases {
		currTmpDir, err := ioutil.TempDir(tmpDir, "")
		require.NoError(t, err)

		_, err = gofiles.Write(currTmpDir, currCase.files)
		require.NoError(t, err)

		dir, args := currCase.getArgs(currTmpDir)

		buf := bytes.Buffer{}
		doMainErr := doImportAlias(dir, args, true, &buf)
		assert.NoError(t, doMainErr, "Case %d (%s)", i, currCase.name)
	}
}

func TestImportAliasError(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tmpDir, cleanup, err := dirs.TempDir(wd, "")
	defer cleanup()
	require.NoError(t, err)

	cases := []struct {
		name          string
		getArgs       func(projectDir string) (string, []string)
		files         []gofiles.GoFileSpec
		regularOutput func(files map[string]gofiles.GoFile) []string
		verboseOutput func(files map[string]gofiles.GoFile) []string
	}{
		{
			name: "error if multiple files import the same package using a different alias",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import bar "fmt"; func Bar(){ bar.Println() }`,
				},
			},
			regularOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					`bar/bar.go:1:21: uses alias "bar" to import package "fmt". No consensus alias exists for this import in the project ("bar" and "foo" are both used once each).`,
					`foo.go:1:22: uses alias "foo" to import package "fmt". No consensus alias exists for this import in the project ("bar" and "foo" are both used once each).`,
				}
			},
			verboseOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					"\"fmt\" is imported using multiple different aliases:",
					"\tbar (1 file):",
					"\t\tbar/bar.go:1:21",
					"\tfoo (1 file):",
					"\t\tfoo.go:1:22",
				}
			},
		},
		{
			name: "error if multiple files import the same package using a different alias for multiple packages",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import bar "fmt"; func Bar(){ bar.Println() }`,
				},
				{
					RelPath: "baz/baz.go",
					Src:     `package baz; import baz "io"; func Baz(){ var w baz.Writer; _ = w }`,
				},
				{
					RelPath: "other/other.go",
					Src:     `package other; import other "io"; func Other(){ var w other.Writer; _ = w }`,
				},
			},
			regularOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					`bar/bar.go:1:21: uses alias "bar" to import package "fmt". No consensus alias exists for this import in the project ("bar" and "foo" are both used once each).`,
					`baz/baz.go:1:21: uses alias "baz" to import package "io". No consensus alias exists for this import in the project ("baz" and "other" are both used once each).`,
					`foo.go:1:22: uses alias "foo" to import package "fmt". No consensus alias exists for this import in the project ("bar" and "foo" are both used once each).`,
					`other/other.go:1:23: uses alias "other" to import package "io". No consensus alias exists for this import in the project ("baz" and "other" are both used once each).`,
				}
			},
			verboseOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					"\"fmt\" is imported using multiple different aliases:",
					"\tbar (1 file):",
					"\t\tbar/bar.go:1:21",
					"\tfoo (1 file):",
					"\t\tfoo.go:1:22",
					"\"io\" is imported using multiple different aliases:",
					"\tbaz (1 file):",
					"\t\tbaz/baz.go:1:21",
					"\tother (1 file):",
					"\t\tother/other.go:1:23",
				}
			},
		},
		{
			name: "if multiple files import the same package using a different alias but one is more common, suggest the more common one",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import bar "fmt"; func Bar(){ bar.Println() }`,
				},
				{
					RelPath: "baz/baz.go",
					Src:     `package baz; import foo "fmt"; func Baz(){ foo.Println() }`,
				},
			},
			regularOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					`bar/bar.go:1:21: uses alias "bar" to import package "fmt". Use alias "foo" instead.`,
				}
			},
			verboseOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					"\"fmt\" is imported using multiple different aliases:",
					"\tfoo (2 files):",
					"\t\tbaz/baz.go:1:21",
					"\t\tfoo.go:1:22",
					"\tbar (1 file):",
					"\t\tbar/bar.go:1:21",
				}
			},
		},
		{
			name: "verify correct message if there are more than 2 aliases used for an import",
			getArgs: func(projectDir string) (string, []string) {
				return projectDir, nil
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import bar "fmt"; func Bar(){ bar.Println() }`,
				},
				{
					RelPath: "baz/baz.go",
					Src:     `package baz; import baz "fmt"; func Baz(){ baz.Println() }`,
				},
			},
			regularOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					`bar/bar.go:1:21: uses alias "bar" to import package "fmt". No consensus alias exists for this import in the project ("bar", "baz" and "foo" are all used once each).`,
					`baz/baz.go:1:21: uses alias "baz" to import package "fmt". No consensus alias exists for this import in the project ("bar", "baz" and "foo" are all used once each).`,
					`foo.go:1:22: uses alias "foo" to import package "fmt". No consensus alias exists for this import in the project ("bar", "baz" and "foo" are all used once each).`,
				}
			},
			verboseOutput: func(files map[string]gofiles.GoFile) []string {
				return []string{
					"\"fmt\" is imported using multiple different aliases:",
					"\tbar (1 file):",
					"\t\tbar/bar.go:1:21",
					"\tbaz (1 file):",
					"\t\tbaz/baz.go:1:21",
					"\tfoo (1 file):",
					"\t\tfoo.go:1:22",
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
		doMainErr := doImportAlias(dir, args, false, &buf)
		require.Error(t, doMainErr, fmt.Sprintf("Case %d (%s)", i, currCase.name))
		assert.Equal(t, currCase.regularOutput(files), strings.Split(doMainErr.Error(), "\n"), "Case %d (%s)", i, currCase.name)

		doMainErr = doImportAlias(dir, args, true, &buf)
		require.Error(t, doMainErr, fmt.Sprintf("Case %d (%s)", i, currCase.name))
		assert.Equal(t, currCase.verboseOutput(files), strings.Split(doMainErr.Error(), "\n"), "Case %d (%s)", i, currCase.name)
	}
}
