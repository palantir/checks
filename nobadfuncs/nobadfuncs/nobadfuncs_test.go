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

package nobadfuncs_test

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
	"github.com/palantir/pkg/pkgpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/palantir/checks/nobadfuncs/nobadfuncs"
)

func TestPrintFuncRefUsages(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tmpDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	for i, currCase := range []struct {
		name  string
		specs []gofiles.GoFileSpec
		sigs  map[string]string
		want  func(testDir string) string
	}{
		{
			name: "empty output when no functions match",
			specs: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src:     `package foo`,
				},
			},
			sigs: map[string]string{
				"func (*net/http.Client).Do(*net/http.Request) (*net/http.Response, error)": "",
			},
			want: func(testDir string) string {
				return ""
			},
		},
		{
			name: "function with matching signature with default message",
			specs: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src: `
package foo

import (
	"net/http"
)

func MyFunction() {
	http.DefaultClient.Do(nil)
}
`,
				},
			},
			sigs: map[string]string{
				"func (*net/http.Client).Do(*net/http.Request) (*net/http.Response, error)": "",
			},
			want: func(testDir string) string {
				return fmt.Sprintf("%s:9:21: references to \"func (*net/http.Client).Do(*net/http.Request) (*net/http.Response, error)\" are not allowed. Remove this reference or whitelist it by adding a comment of the form '// OK: [reason]' to the line before it.\n", path.Join(wd, testDir, "foo/foo.go"))
			},
		},
		{
			name: "function signature matches through vendors",
			specs: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src: `
package foo

import (
	"github.com/bar"
)

func MyFunction() {
	bar.Bar()
}
`,
				},
				{
					RelPath: "vendor/github.com/bar/bar.go",
					Src: `
package bar

func Bar() {}
`,
				},
			},
			sigs: map[string]string{
				"func github.com/bar.Bar()": "",
			},
			want: func(testDir string) string {
				return fmt.Sprintf("%s:9:6: references to \"func github.com/bar.Bar()\" are not allowed. Remove this reference or whitelist it by adding a comment of the form '// OK: [reason]' to the line before it.\n", path.Join(wd, testDir, "foo/foo.go"))
			},
		},
		{
			name: "function with matching signature with custom message",
			specs: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src: `
package foo

import (
	"net/http"
)

func MyFunction() {
	http.DefaultClient.Do(nil)
}
`,
				},
			},
			sigs: map[string]string{
				"func (*net/http.Client).Do(*net/http.Request) (*net/http.Response, error)": "TEST: don't use this please",
			},
			want: func(testDir string) string {
				return fmt.Sprintf("%s:9:21: TEST: don't use this please\n", path.Join(wd, testDir, "foo/foo.go"))
			},
		},
		{
			name: "function with matching signature is skipped when whitelisted",
			specs: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src: `
package foo

import (
	"net/http"
)

func MyFunction() {
	// OK: my reason for this being good to call
	http.DefaultClient.Do(nil)
}
`,
				},
			},
			sigs: map[string]string{
				"func (*net/http.Client).Do(*net/http.Request) (*net/http.Response, error)": "",
			},
			want: func(testDir string) string {
				return ""
			},
		},
		{
			name: "find references in various forms",
			specs: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src: `
package foo

import (
	"net/http"
)

func CallViaReference() {
	myRef := http.DefaultClient.Do
	myRef(nil)
}

type myVar struct {
	Hidden http.Client
}

func InStruct() {
	var v myVar
	v.Hidden.Do(nil)
}

type MyAlias http.Client

func TypeAlias() {
	var v MyAlias
	(*http.Client)(&v).Do(nil)
}
`,
				},
			},
			sigs: map[string]string{
				"func (*net/http.Client).Do(*net/http.Request) (*net/http.Response, error)": "No",
			},
			want: func(testDir string) string {
				return strings.Join([]string{
					fmt.Sprintf("%s:9:30: No", path.Join(wd, testDir, "foo/foo.go")),
					fmt.Sprintf("%s:19:11: No", path.Join(wd, testDir, "foo/foo.go")),
					fmt.Sprintf("%s:26:21: No", path.Join(wd, testDir, "foo/foo.go")),
				}, "\n") + "\n"
			},
		},
	} {
		currCaseTmpDir, err := ioutil.TempDir(tmpDir, fmt.Sprintf("case-%d-", i))
		require.NoError(t, err)

		files, err := gofiles.Write(currCaseTmpDir, currCase.specs)
		require.NoError(t, err, "Case %d: %s", i, currCase.name)

		var pkgs []string
		for _, val := range files {
			currPkg, err := pkgpath.NewAbsPkgPath(path.Dir(val.Path)).GoPathSrcRel()
			require.NoError(t, err)
			pkgs = append(pkgs, currPkg)
		}

		var got bytes.Buffer
		_, err = nobadfuncs.PrintBadFuncRefs(pkgs, currCase.sigs, &got)
		require.NoError(t, err, "Case %d: %s", i, currCase.name)

		assert.Equal(t, currCase.want(currCaseTmpDir), got.String(), "Case %d: %s\nOutput:\n%s", i, currCase.name, got.String())
	}

}

func TestPrintAllFuncRefs(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tmpDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	for i, currCase := range []struct {
		name  string
		specs []gofiles.GoFileSpec
		want  func(testDir string) string
	}{
		{
			name: "all function signatures are printed",
			specs: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src: `
package foo

import (
	"fmt"
	"net/http"
)

func MyFunction() {
	http.DefaultClient.Do(nil)
	http.DefaultClient.PostForm("", nil)

	// OK: this isn't ignored when printing all
	fmt.Println("hello")
}
`,
				},
			},
			want: func(testDir string) string {
				return strings.Join([]string{
					fmt.Sprintf("%s:10:21: func (*net/http.Client).Do(*net/http.Request) (*net/http.Response, error)", path.Join(wd, testDir, "foo/foo.go")),
					fmt.Sprintf("%s:11:21: func (*net/http.Client).PostForm(string, net/url.Values) (*net/http.Response, error)", path.Join(wd, testDir, "foo/foo.go")),
					fmt.Sprintf("%s:14:6: func fmt.Println(...interface{}) (int, error)", path.Join(wd, testDir, "foo/foo.go")),
				}, "\n") + "\n"
			},
		},
		{
			name: "prints vendored signatures but omits vendor from package",
			specs: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src: `
package foo

import (
	"github.com/bar"
)

func MyFunction() {
	var b bar.BarType
	b.Bar(bar.BarType(""))

	bar.FreeBar()
}
`,
				},
				{
					RelPath: "vendor/github.com/bar/bar.go",
					Src: `
package bar

type BarType string

func FreeBar() {}

func (b BarType) Bar(in BarType) BarType {
	return in
}
`,
				},
			},
			want: func(testDir string) string {
				return strings.Join([]string{
					fmt.Sprintf("%s:10:4: func (github.com/bar.BarType).Bar(github.com/bar.BarType) github.com/bar.BarType", path.Join(wd, testDir, "foo/foo.go")),
					fmt.Sprintf("%s:12:6: func github.com/bar.FreeBar()", path.Join(wd, testDir, "foo/foo.go")),
				}, "\n") + "\n"
			},
		},
	} {
		currCaseTmpDir, err := ioutil.TempDir(tmpDir, fmt.Sprintf("case-%d-", i))
		require.NoError(t, err)

		files, err := gofiles.Write(currCaseTmpDir, currCase.specs)
		require.NoError(t, err, "Case %d: %s", i, currCase.name)

		var pkgs []string
		for _, val := range files {
			currPkg, err := pkgpath.NewAbsPkgPath(path.Dir(val.Path)).GoPathSrcRel()
			require.NoError(t, err)
			pkgs = append(pkgs, currPkg)
		}

		var got bytes.Buffer
		err = nobadfuncs.PrintAllFuncRefs(pkgs, &got)
		require.NoError(t, err, "Case %d: %s", i, currCase.name)

		assert.Equal(t, currCase.want(currCaseTmpDir), got.String(), "Case %d: %s\nOutput:\n%s", i, currCase.name, got.String())
	}
}
