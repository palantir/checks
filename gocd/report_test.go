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

package gocd_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/nmiyake/pkg/dirs"
	"github.com/nmiyake/pkg/gofiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/palantir/checks/gocd"
)

// TestGetProjectPkgInfoAllPkgs tests that GetProjectPkgInfo finds all packages.
func TestImportReport(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	tmpDir, err = filepath.Abs(tmpDir)
	require.NoError(t, err)

	for i, currCase := range []struct {
		name  string
		files []gofiles.GoFileSpec
		want  func(files map[string]gofiles.GoFile) gocd.ImportReport
	}{
		{
			name: "main package with no imports",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/main.go",
					Src:     "package main",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.ImportReport {
				return gocd.ImportReport{
					Imports:         []gocd.ImportReportPkg{},
					MainOnlyImports: []gocd.ImportReportPkg{},
					TestOnlyImports: []gocd.ImportReportPkg{},
				}
			},
		},
		{
			name: "package imported by only core package",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/foo.go",
					Src:     `package foo; import _ "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     "package bar",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.ImportReport {
				return gocd.ImportReport{
					Imports: []gocd.ImportReportPkg{
						{
							Path:             files["bar/bar.go"].ImportPath,
							NGoFiles:         1,
							NImportedGoFiles: 0,
							ImportSrc: []string{
								files["projectDir/foo.go"].ImportPath,
							},
						},
					},
					MainOnlyImports: []gocd.ImportReportPkg{},
					TestOnlyImports: []gocd.ImportReportPkg{},
				}
			},
		},
		{
			name: "package imported by only main package",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/main.go",
					Src:     `package main; import _ "{{index . "bar/bar.go"}}"`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     "package bar",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.ImportReport {
				return gocd.ImportReport{
					Imports: []gocd.ImportReportPkg{},
					MainOnlyImports: []gocd.ImportReportPkg{
						{
							Path:             files["bar/bar.go"].ImportPath,
							NGoFiles:         1,
							NImportedGoFiles: 0,
							ImportSrc: []string{
								files["projectDir/main.go"].ImportPath,
							},
						},
					},
					TestOnlyImports: []gocd.ImportReportPkg{},
				}
			},
		},
		{
			name: "package imported by only test package",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/foo_test.go",
					Src:     `package foo; import _ "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     "package bar",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.ImportReport {
				return gocd.ImportReport{
					Imports:         []gocd.ImportReportPkg{},
					MainOnlyImports: []gocd.ImportReportPkg{},
					TestOnlyImports: []gocd.ImportReportPkg{
						{
							Path:             files["bar/bar.go"].ImportPath,
							NGoFiles:         1,
							NImportedGoFiles: 0,
							ImportSrc: []string{
								files["projectDir/foo_test.go"].ImportPath + "_test",
							},
						},
					},
				}
			},
		},
		{
			name: "package imported by core and main package",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/main/main.go",
					Src:     `package main; import _ "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "projectDir/foo.go",
					Src:     `package foo; import _ "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     "package bar",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.ImportReport {
				return gocd.ImportReport{
					Imports: []gocd.ImportReportPkg{
						{
							Path:             files["bar/bar.go"].ImportPath,
							NGoFiles:         1,
							NImportedGoFiles: 0,
							ImportSrc: []string{
								files["projectDir/foo.go"].ImportPath,
								files["projectDir/main/main.go"].ImportPath,
							},
						},
					},
					MainOnlyImports: []gocd.ImportReportPkg{},
					TestOnlyImports: []gocd.ImportReportPkg{},
				}
			},
		},
		{
			name: "package imported by core and test package",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/foo_test.go",
					Src:     `package foo; import _ "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "projectDir/baz/baz.go",
					Src:     `package baz; import _ "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     "package bar",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.ImportReport {
				return gocd.ImportReport{
					Imports: []gocd.ImportReportPkg{
						{
							Path:             files["bar/bar.go"].ImportPath,
							NGoFiles:         1,
							NImportedGoFiles: 0,
							ImportSrc: []string{
								files["projectDir/baz/baz.go"].ImportPath,
								files["projectDir/foo_test.go"].ImportPath + "_test",
							},
						},
					},
					MainOnlyImports: []gocd.ImportReportPkg{},
					TestOnlyImports: []gocd.ImportReportPkg{},
				}
			},
		},
		{
			name: "package imported by main and test packages",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/foo_test.go",
					Src:     `package foo; import _ "{{index . "bar/bar.go"}}"`,
				},
				{
					RelPath: "projectDir/main/main.go",
					Src:     `package main; import _ "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     "package bar",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.ImportReport {
				return gocd.ImportReport{
					Imports: []gocd.ImportReportPkg{},
					MainOnlyImports: []gocd.ImportReportPkg{
						{
							Path:             files["bar/bar.go"].ImportPath,
							NGoFiles:         1,
							NImportedGoFiles: 0,
							ImportSrc: []string{
								files["projectDir/main/main.go"].ImportPath,
								files["projectDir/foo_test.go"].ImportPath + "_test",
							},
						},
					},
					TestOnlyImports: []gocd.ImportReportPkg{},
				}
			},
		},
		{
			name: "package with imports counts number of imported files",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/foo.go",
					Src:     `package foo; import _ "{{index . "bar/bar.go"}}"`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import _ "{{index . "baz/baz.go"}}"`,
				},
				{
					RelPath: "bar/bar_2.go",
					Src:     "package bar",
				},
				{
					RelPath: "baz/baz.go",
					Src:     "package baz",
				},
				{
					RelPath: "baz/baz_2.go",
					Src:     "package baz",
				},
				{
					RelPath: "baz/baz_3.go",
					Src:     "package baz",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.ImportReport {
				return gocd.ImportReport{
					Imports: []gocd.ImportReportPkg{
						{
							Path:             files["bar/bar.go"].ImportPath,
							NGoFiles:         2,
							NImportedGoFiles: 3,
							ImportSrc: []string{
								files["projectDir/foo.go"].ImportPath,
							},
						},
					},
					MainOnlyImports: []gocd.ImportReportPkg{},
					TestOnlyImports: []gocd.ImportReportPkg{},
				}
			},
		},
		{
			name: "imports are not double-counted",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/main.go",
					Src:     `package main; import _ "{{index . "foo/foo.go"}}";`,
				},
				{
					RelPath: "foo/foo.go",
					Src:     `package foo; import _ "{{index . "bar/bar.go"}}"; import _ "{{index . "baz/baz.go"}}";`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import _ "{{index . "common/common.go"}}"`,
				},
				{
					RelPath: "baz/baz.go",
					Src:     `package baz; import _ "{{index . "common/common.go"}}"`,
				},
				{
					RelPath: "common/common.go",
					Src:     "package common",
				},
				{
					RelPath: "common/common_2.go",
					Src:     "package common",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.ImportReport {
				return gocd.ImportReport{
					Imports: []gocd.ImportReportPkg{},
					MainOnlyImports: []gocd.ImportReportPkg{
						{
							Path:             files["foo/foo.go"].ImportPath,
							NGoFiles:         1,
							NImportedGoFiles: 4,
							ImportSrc: []string{
								files["projectDir/main.go"].ImportPath,
							},
						},
					},
					TestOnlyImports: []gocd.ImportReportPkg{},
				}
			},
		},
	} {
		currCaseTmpDir, err := ioutil.TempDir(tmpDir, "")
		require.NoError(t, err, "Case %d (%s)", i, currCase.name)

		currCaseProjectDir := path.Join(currCaseTmpDir, "projectDir")
		err = os.Mkdir(currCaseProjectDir, 0755)
		require.NoError(t, err, "Case %d (%s)", i, currCase.name)

		files, err := gofiles.Write(currCaseTmpDir, currCase.files)
		require.NoError(t, err, "Case %d (%s)", i, currCase.name)

		got, err := gocd.CreateImportReport(currCaseProjectDir)
		require.NoError(t, err, "Case %d (%s)", i, currCase.name)
		assert.Equal(t, currCase.want(files), got, "Case %d (%s)", i, currCase.name)
	}
}
