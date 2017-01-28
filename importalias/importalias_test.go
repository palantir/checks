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
	"io/ioutil"
	"os"
	"testing"

	"github.com/nmiyake/pkg/dirs"
	"github.com/nmiyake/pkg/gofiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportAlias(t *testing.T) {
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
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				assert.NoError(t, err, "Case %d (%s)", caseNum, caseName)
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
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				assert.NoError(t, err, "Case %d (%s)", caseNum, caseName)
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
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				assert.NoError(t, err, "Case %d (%s)", caseNum, caseName)
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
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				assert.NoError(t, err, "Case %d (%s)", caseNum, caseName)
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
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				assert.NoError(t, err, "Case %d (%s)", caseNum, caseName)
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
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				assert.NoError(t, err, "Case %d (%s)", caseNum, caseName)
			},
		},
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
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				assert.EqualError(t, err, "\"fmt\" is imported using multiple different aliases:\n\tbar:\n\t\tbar/bar.go\n\tfoo:\n\t\tfoo.go", "Case %d (%s)", caseNum, caseName)
			},
		},
		{
			name: "error if multiple files import the same package using a different alias for multiple aliases",
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
			verify: func(files map[string]gofiles.GoFile, got string, err error, caseNum int, caseName string) {
				assert.EqualError(t, err, "\"fmt\" is imported using multiple different aliases:\n\tbar:\n\t\tbar/bar.go\n\tfoo:\n\t\tfoo.go\n\"io\" is imported using multiple different aliases:\n\tbaz:\n\t\tbaz/baz.go\n\tother:\n\t\tother/other.go", "Case %d (%s)", caseNum, caseName)
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
		doMainErr := doImportAlias(dir, args, &buf)
		currCase.verify(files, buf.String(), doMainErr, i, currCase.name)
	}
}
