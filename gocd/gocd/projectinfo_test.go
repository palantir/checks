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

	"github.com/palantir/checks/gocd/gocd"
)

func TestPkgInfos(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	tmpDir, err = filepath.Abs(tmpDir)
	require.NoError(t, err)

	for i, currCase := range []struct {
		name  string
		files []gofiles.GoFileSpec
		want  func(files map[string]gofiles.GoFile) gocd.PkgInfos
	}{
		{
			name: "main package with no imports",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/main.go",
					Src:     "package main",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfos {
				return []*gocd.PkgInfo{
					{
						Path:     files["projectDir/main.go"].ImportPath,
						Name:     "main",
						Imports:  map[string]map[string]struct{}{},
						NGoFiles: 1,
					},
				}
			},
		},
		{
			name: "main package with internal import",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/main.go",
					Src:     `package main; import _ "{{index . "projectDir/bar/bar.go"}}";`,
				},
				{
					RelPath: "projectDir/bar/bar.go",
					Src:     "package bar",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfos {
				return []*gocd.PkgInfo{
					{
						Path:     files["projectDir/main.go"].ImportPath,
						Name:     "main",
						NGoFiles: 1,
						Imports: map[string]map[string]struct{}{
							files["projectDir/bar/bar.go"].ImportPath: {
								files["projectDir/main.go"].Path: {},
							},
						},
					},
					{
						Path:     files["projectDir/bar/bar.go"].ImportPath,
						Name:     "bar",
						NGoFiles: 1,
						Imports:  map[string]map[string]struct{}{},
					},
				}
			},
		},
		{
			name: "main package with external import",
			files: []gofiles.GoFileSpec{
				{
					RelPath: "projectDir/main.go",
					Src:     `package main; import _ "{{index . "bar/bar.go"}}";`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     "package bar",
				},
			},
			want: func(files map[string]gofiles.GoFile) gocd.PkgInfos {
				return []*gocd.PkgInfo{
					{
						Path:     files["projectDir/main.go"].ImportPath,
						Name:     "main",
						NGoFiles: 1,
						Imports: map[string]map[string]struct{}{
							files["bar/bar.go"].ImportPath: {
								files["projectDir/main.go"].Path: {},
							},
						},
					},
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

		project, err := gocd.NewProjectPkgInfoer(currCaseProjectDir)
		require.NoError(t, err, "Case %d (%s)", i, currCase.name)

		assert.Equal(t, currCase.want(files), project.PkgInfos(), "Case %d (%s)", i, currCase.name)
	}
}
