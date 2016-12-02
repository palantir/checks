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

func TestNGoFiles(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	tmpDir, err = filepath.Abs(tmpDir)
	require.NoError(t, err)

	for i, currCase := range []struct {
		name              string
		files             []gofiles.GoFileSpec
		pkg               func(files map[string]gofiles.GoFile) string
		wantNGoFiles      int
		wantNTotalGoFiles int
	}{
		{
			name: "main package with no imports",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/main.go",
					Src:     "package main",
				},
			},
			pkg: func(files map[string]gofiles.GoFile) string {
				return files["projectDir/main.go"].ImportPath
			},
			wantNGoFiles:      1,
			wantNTotalGoFiles: 1,
		},
		{
			name: "core package with multi-file imports",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/foo.go",
					Src:     `package foo; import _ "{{index . "projectDir/bar/bar.go"}}";`,
				},
				{
					RelPath: "projectDir/foo_2.go",
					Src:     "package foo",
				},
				{
					RelPath: "projectDir/bar/bar.go",
					Src:     `package bar; import _ "{{index . "projectDir/baz/baz.go"}}";`,
				},
				{
					RelPath: "projectDir/bar/bar_2.go",
					Src:     "package bar",
				},
				{
					RelPath: "projectDir/baz/baz.go",
					Src:     "package baz",
				},
			},
			pkg: func(files map[string]gofiles.GoFile) string {
				return files["projectDir/foo.go"].ImportPath
			},
			wantNGoFiles:      2,
			wantNTotalGoFiles: 5,
		},
	} {
		currCaseTmpDir, err := ioutil.TempDir(tmpDir, "")
		require.NoError(t, err, "Case %d (%s)", i, currCase.name)

		currCaseProjectDir := path.Join(currCaseTmpDir, "projectDir")
		err = os.Mkdir(currCaseProjectDir, 0755)
		require.NoError(t, err, "Case %d (%s)", i, currCase.name)

		files, err := gofiles.Write(currCaseTmpDir, currCase.files)
		require.NoError(t, err, "Case %d (%s)", i, currCase.name)

		project, err := gocd.NewProjectPkgInfoer(currCaseProjectDir)
		require.NoError(t, err, "Case %d (%s)", i, currCase.name)

		counter, err := gocd.NewProjectGoFileCounter(project)
		require.NoError(t, err, "Case %d (%s)", i, currCase.name)

		nGoFiles, ok := counter.NGoFiles(currCase.pkg(files))
		require.True(t, ok, "Case %d (%s)", i, currCase.name)
		assert.Equal(t, currCase.wantNGoFiles, nGoFiles, "Case %d (%s)", i, currCase.name)

		nTotalGoFiles, ok := counter.NTotalGoFiles(currCase.pkg(files))
		require.True(t, ok, "Case %d (%s)", i, currCase.name)
		assert.Equal(t, currCase.wantNTotalGoFiles, nTotalGoFiles, "Case %d (%s)", i, currCase.name)
	}
}
