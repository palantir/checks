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

func TestCompiles(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tmpDir, cleanup, err := dirs.TempDir(wd, "")
	require.NoError(t, err)
	defer cleanup()

	cases := []struct {
		files []gofiles.GoFileSpec
		want  func(files map[string]gofiles.GoFile) string
	}{
		{
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src: `package foo
				func Foo() {
					return "Foo"
				}`,
				},
				{
					RelPath: "bar/bar.go",
					Src: `package bar
				import "fmt"`,
				},
			},
			want: func(files map[string]gofiles.GoFile) string {
				lines := []string{
					files["foo/foo.go"].Path + `:3:13: no result values expected`,
					files["bar/bar.go"].Path + `:2:12: "fmt" imported but not used`,
					files["bar/bar.go"].Path + `:2:12: "fmt" imported but not used`,
					"",
				}
				return strings.Join(lines, "\n")
			},
		},
		{
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src: `package foo
				func Foo() string {
					return "Foo"
				}`,
				},
				{
					RelPath: "foo/foo_test.go",
					Src: `package foo
				import (
					"testing"
					"{{index . "foo/foo.go"}}"
				)
				func TestFoo(t *testing.T) {
					bar := foo.Foo()
				}`,
				},
			},
			want: func(files map[string]gofiles.GoFile) string {
				lines := []string{
					files["foo/foo_test.go"].Path + `:7:6: bar declared but not used`,
					"",
				}
				return strings.Join(lines, "\n")
			},
		},
		{
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src: `package foo
				func Foo() string {
					return "Foo"
				}`,
				},
				{
					RelPath: "foo/foo_test.go",
					Src: `package foo_test
				import (
					"testing"
					"{{index . "foo/foo.go"}}"
				)
				func TestFoo(t *testing.T) {
					bar := foo.Foo()
				}`,
				},
			},
			want: func(files map[string]gofiles.GoFile) string {
				lines := []string{
					files["foo/foo_test.go"].Path + `:7:6: bar declared but not used`,
					"",
				}
				return strings.Join(lines, "\n")
			},
		},
	}

	for i, currCase := range cases {
		projectDir, err := ioutil.TempDir(tmpDir, "")
		require.NoError(t, err)

		buf := bytes.Buffer{}
		files, err := gofiles.Write(projectDir, currCase.files)
		require.NoError(t, err)

		err = doCompiles(projectDir, nil, &buf)
		require.Error(t, err, fmt.Sprintf("Case %d", i))

		assert.Equal(t, currCase.want(files), buf.String(), "Case %d", i)
	}
}
