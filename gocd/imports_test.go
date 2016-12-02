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
	"fmt"
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

func TestDirPkgInfo(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	tmpDir, err = filepath.Abs(tmpDir)
	require.NoError(t, err)

	for i, currCase := range []struct {
		name      string
		files     []gofiles.GoFileSpec
		mode      gocd.PkgMode
		wantEmpty bool
		want      func(files map[string]gofiles.GoFile) gocd.PkgInfo
	}{
		{
			name: "core package with no imports",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/bar.go",
					Src:     "package bar",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfo {
				return gocd.PkgInfo{
					Path:     files["projectDir/bar.go"].ImportPath,
					Name:     "bar",
					NGoFiles: 1,
					Imports:  make(map[string]map[string]struct{}),
				}
			},
		},
		{
			name: "core package with internal import",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/bar.go",
					Src:     `package bar; import _ "{{index . "projectDir/foo/foo.go"}}";`,
				},
				{
					RelPath: "projectDir/foo/foo.go",
					Src:     "package foo",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfo {
				return gocd.PkgInfo{
					Path:     files["projectDir/bar.go"].ImportPath,
					Name:     "bar",
					NGoFiles: 1,
					Imports: map[string]map[string]struct{}{
						files["projectDir/foo/foo.go"].ImportPath: {
							files["projectDir/bar.go"].Path: {},
						},
					},
				}
			},
		},
		{
			name: "core package with external import",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/bar.go",
					Src:     `package bar; import _ "{{index . "foo/foo.go"}}";`,
				},
				{
					RelPath: "foo/foo.go",
					Src:     "package foo",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfo {
				return gocd.PkgInfo{
					Path:     files["projectDir/bar.go"].ImportPath,
					Name:     "bar",
					NGoFiles: 1,
					Imports: map[string]map[string]struct{}{
						files["foo/foo.go"].ImportPath: {
							files["projectDir/bar.go"].Path: {},
						},
					},
				}
			},
		},
		{
			name: "core package using test mode",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/bar.go",
					Src:     "package bar",
				},
			},
			mode:      gocd.Test,
			wantEmpty: true,
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfo {
				return gocd.PkgInfo{
					Path:     files["projectDir/bar.go"].ImportPath + "_test",
					Name:     "bar",
					NGoFiles: 1,
					Imports:  make(map[string]map[string]struct{}),
				}
			},
		},
		{
			name: "main package with no imports",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/main.go",
					Src:     "package main",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfo {
				return gocd.PkgInfo{
					Path:     files["projectDir/main.go"].ImportPath,
					Name:     "main",
					NGoFiles: 1,
					Imports:  make(map[string]map[string]struct{}),
				}
			},
		},
		{
			name: "main package with internal import",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/main.go",
					Src:     `package main; import _ "{{index . "projectDir/foo/foo.go"}}";`,
				},
				{
					RelPath: "projectDir/foo/foo.go",
					Src:     "package foo",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfo {
				return gocd.PkgInfo{
					Path:     files["projectDir/main.go"].ImportPath,
					Name:     "main",
					NGoFiles: 1,
					Imports: map[string]map[string]struct{}{
						files["projectDir/foo/foo.go"].ImportPath: {
							files["projectDir/main.go"].Path: {},
						},
					},
				}
			},
		},
		{
			name: "main package with external import",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/main.go",
					Src:     `package main; import _ "{{index . "foo/foo.go"}}";`,
				},
				{
					RelPath: "foo/foo.go",
					Src:     "package foo",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfo {
				return gocd.PkgInfo{
					Path:     files["projectDir/main.go"].ImportPath,
					Name:     "main",
					NGoFiles: 1,
					Imports: map[string]map[string]struct{}{
						files["foo/foo.go"].ImportPath: {
							files["projectDir/main.go"].Path: {},
						},
					},
				}
			},
		},
		{
			name: "test package with no imports",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/bar_test.go",
					Src:     "package bar_test",
				},
			},
			mode: gocd.Test,
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfo {
				return gocd.PkgInfo{
					Path:     files["projectDir/bar_test.go"].ImportPath + "_test",
					Name:     "bar",
					NGoFiles: 1,
					Imports:  make(map[string]map[string]struct{}),
				}
			},
		},
		{
			name: "test package with internal import",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/bar_test.go",
					Src:     `package bar_test; import _ "{{index . "projectDir/foo/foo.go"}}";`,
				},
				{
					RelPath: "projectDir/foo/foo.go",
					Src:     "package foo",
				},
			},
			mode: gocd.Test,
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfo {
				return gocd.PkgInfo{
					Path:     files["projectDir/bar_test.go"].ImportPath + "_test",
					Name:     "bar",
					NGoFiles: 1,
					Imports: map[string]map[string]struct{}{
						files["projectDir/foo/foo.go"].ImportPath: {
							files["projectDir/bar_test.go"].Path: {},
						},
					},
				}
			},
		},
		{
			name: "test package with external import",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/bar_test.go",
					Src:     `package bar_test; import _ "{{index . "foo/foo.go"}}";`,
				},
				{
					RelPath: "foo/foo.go",
					Src:     "package foo",
				},
			},
			mode: gocd.Test,
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfo {
				return gocd.PkgInfo{
					Path:     files["projectDir/bar_test.go"].ImportPath + "_test",
					Name:     "bar",
					NGoFiles: 1,
					Imports: map[string]map[string]struct{}{
						files["foo/foo.go"].ImportPath: {
							files["projectDir/bar_test.go"].Path: {},
						},
					},
				}
			},
		},
		{
			name: "test package using default mode",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/bar_test.go",
					Src:     "package bar_test",
				},
			},
			mode:      gocd.Default,
			wantEmpty: true,
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfo {
				return gocd.PkgInfo{
					Path:     files["projectDir/bar_test.go"].ImportPath,
					Name:     "bar",
					NGoFiles: 1,
					Imports:  make(map[string]map[string]struct{}),
				}
			},
		},
		{
			name: "import excluded by build contraint is found",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/foo.go",
					Src: `// +build android

package foo; import "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     "package bar; func Bar(){}",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfo {
				return gocd.PkgInfo{
					Path:     files["projectDir/foo.go"].ImportPath,
					Name:     "foo",
					NGoFiles: 1,
					Imports: map[string]map[string]struct{}{
						files["bar/bar.go"].ImportPath: {
							files["projectDir/foo.go"].Path: {},
						},
					},
				}
			},
		},
		{
			name: "handles multiple packages where a package is excluded by build constraint",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/main.go",
					Src: `// +build ignore

package main`,
				},
				{
					RelPath: "projectDir/foo.go",
					Src:     "package foo",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfo {
				return gocd.PkgInfo{
					Path:     files["projectDir/foo.go"].ImportPath,
					Name:     "foo",
					NGoFiles: 2,
					Imports:  make(map[string]map[string]struct{}),
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

		got, empty, err := gocd.DirPkgInfo(currCaseProjectDir, currCase.mode)
		require.NoError(t, err, "Case %d (%s)", i, currCase.name)

		assert.Equal(t, currCase.want(files), got, "Case %d (%s)", i, currCase.name)
		assert.Equal(t, currCase.wantEmpty, empty, "Case %d (%s)", i, currCase.name)
	}
}

func TestImportPkgInfo(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	tmpDir, err = filepath.Abs(tmpDir)
	require.NoError(t, err)

	for i, currCase := range []struct {
		name       string
		files      []gofiles.GoFileSpec
		mode       gocd.PkgMode
		importPath func(files map[string]gofiles.GoFile) string
		srcDir     string
		wantEmpty  bool
		want       func(files map[string]gofiles.GoFile) gocd.PkgInfo
	}{
		{
			name: "vendored import",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/bar.go",
					Src:     "package bar",
				},
				{
					RelPath: "projectDir/vendor/github.com/foo/foo.go",
					Src:     "package foo",
				},
			},
			importPath: func(files map[string]gofiles.GoFile) string {
				return files["projectDir/vendor/github.com/foo/foo.go"].ImportPath
			},
			srcDir: "projectDir",
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfo {
				vendorDir := path.Join(files["projectDir/bar.go"].ImportPath, "vendor")
				fmt.Println(vendorDir)
				return gocd.PkgInfo{
					Path:     path.Join(vendorDir, files["projectDir/vendor/github.com/foo/foo.go"].ImportPath),
					Name:     "foo",
					NGoFiles: 1,
					Imports:  make(map[string]map[string]struct{}),
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

		got, empty, err := gocd.ImportPkgInfo(currCase.importPath(files), path.Join(currCaseTmpDir, currCase.srcDir), currCase.mode)
		require.NoError(t, err, "Case %d (%s)", i, currCase.name)

		assert.Equal(t, currCase.want(files), got, "Case %d (%s)", i, currCase.name)
		assert.Equal(t, currCase.wantEmpty, empty, "Case %d (%s)", i, currCase.name)
	}
}
