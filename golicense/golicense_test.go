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

package golicense_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/nmiyake/pkg/dirs"
	"github.com/nmiyake/pkg/gofiles"
	"github.com/palantir/pkg/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/palantir/checks/golicense"
)

func TestLicenseFiles(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir("", "")
	defer cleanup()
	require.NoError(t, err)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			require.NoError(t, err)
		}
	}()

	for i, currCase := range []struct {
		name         string
		params       golicense.LicenseParams
		goFiles      []gofiles.GoFileSpec
		nonGoFiles   map[string]string
		wantModified []string
		wantContent  map[string]string
	}{
		{
			name: "license applied to Go files",
			params: golicense.LicenseParams{
				Header: `// Copyright 2016 Palantir Technologies, Inc.`,
			},
			goFiles: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package foo`,
				},
				{
					RelPath: "bar/bar.go",
					Src: `// Original comment
package bar`,
				},
			},
			wantModified: []string{
				"bar/bar.go",
				"foo.go",
			},
			wantContent: map[string]string{
				"foo.go": `// Copyright 2016 Palantir Technologies, Inc.
package foo`,
				"bar/bar.go": `// Copyright 2016 Palantir Technologies, Inc.
// Original comment
package bar`,
			},
		},
		{
			name: "license not applied to non-Go files",
			params: golicense.LicenseParams{
				Header: `// Copyright 2016 Palantir Technologies, Inc.`,
			},
			nonGoFiles: map[string]string{
				"foo.txt": `package foo`,
			},
			wantContent: map[string]string{
				"foo.txt": `package foo`,
			},
		},
		{
			name: "license not applied to excluded files",
			params: golicense.LicenseParams{
				Header:  `// Copyright 2016 Palantir Technologies, Inc.`,
				Exclude: matcher.Name("foo.go"),
			},
			goFiles: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package foo`,
				},
				{
					RelPath: "bar/bar.go",
					Src: `// Original comment
package bar`,
				},
			},
			wantModified: []string{
				"bar/bar.go",
			},
			wantContent: map[string]string{
				"foo.go": `package foo`,
				"bar/bar.go": `// Copyright 2016 Palantir Technologies, Inc.
// Original comment
package bar`,
			},
		},
		{
			name: "license not re-applied to files that already have license",
			params: golicense.LicenseParams{
				Header: `// Copyright 2016 Palantir Technologies, Inc.`,
			},
			goFiles: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package foo`,
				},
				{
					RelPath: "bar/bar.go",
					Src: `// Copyright 2016 Palantir Technologies, Inc.
// Original comment
package bar`,
				},
			},
			wantModified: []string{
				"foo.go",
			},
			wantContent: map[string]string{
				"foo.go": `// Copyright 2016 Palantir Technologies, Inc.
package foo`,
				"bar/bar.go": `// Copyright 2016 Palantir Technologies, Inc.
// Original comment
package bar`,
			},
		},
		{
			name: "custom license applied to files that match custom matchers",
			params: golicense.LicenseParams{
				Header: `// Copyright 2016 Palantir Technologies, Inc.`,
				CustomHeaders: []golicense.CustomLicenseParam{
					{
						Name:         "Custom Co.",
						Header:       "// Copyright 2016 Custom Co.",
						IncludePaths: []string{"bar/bar.go"},
					},
					{
						Name:         "Baz",
						Header:       "// Copyright 2006 Legacy Inc.",
						IncludePaths: []string{"baz/baz.go"},
					},
				},
			},
			goFiles: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package foo`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar`,
				},
				{
					RelPath: "baz/baz.go",
					Src:     `package baz`,
				},
			},
			wantModified: []string{
				"bar/bar.go",
				"baz/baz.go",
				"foo.go",
			},
			wantContent: map[string]string{
				"foo.go": `// Copyright 2016 Palantir Technologies, Inc.
package foo`,
				"bar/bar.go": `// Copyright 2016 Custom Co.
package bar`,
				"baz/baz.go": `// Copyright 2006 Legacy Inc.
package baz`,
			},
		},
		{
			name: "custom matchers match hierarchically",
			params: golicense.LicenseParams{
				Header: `// Copyright 2016 Palantir Technologies, Inc.`,
				CustomHeaders: []golicense.CustomLicenseParam{
					{
						Name:         "Custom Co.",
						Header:       "// Copyright 2016 Custom Co.",
						IncludePaths: []string{"bar"},
					},
					{
						Name:   "Baz",
						Header: "// Copyright 2006 Legacy Inc.",
						IncludePaths: []string{
							"bar/baz.go",
							"bar/subdir",
						},
					},
				},
			},
			goFiles: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package foo`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar`,
				},
				{
					RelPath: "bar/baz.go",
					Src:     `package bar`,
				},
				{
					RelPath: "bar/subdir/main.go",
					Src:     `package main`,
				},
			},
			wantModified: []string{
				"bar/bar.go",
				"bar/baz.go",
				"bar/subdir/main.go",
				"foo.go",
			},
			wantContent: map[string]string{
				"foo.go": `// Copyright 2016 Palantir Technologies, Inc.
package foo`,
				"bar/bar.go": `// Copyright 2016 Custom Co.
package bar`,
				"bar/baz.go": `// Copyright 2006 Legacy Inc.
package bar`,
				"bar/subdir/main.go": `// Copyright 2006 Legacy Inc.
package main`,
			},
		},
	} {
		currTmpDir, err := ioutil.TempDir(tmpDir, "")
		require.NoError(t, err, "Case %d: %s", i, currCase.name)

		err = os.Chdir(currTmpDir)
		require.NoError(t, err, "Case %d: %s", i, currCase.name)

		_, err = gofiles.Write(currTmpDir, currCase.goFiles)
		require.NoError(t, err, "Case %d: %s", i, currCase.name)
		writeFiles(t, currCase.nonGoFiles)

		files, err := matcher.ListFiles(currTmpDir, matcher.Name(`.+`), nil)
		require.NoError(t, err, "Case %d: %s", i, currCase.name)

		modified, err := golicense.LicenseFiles(files, currCase.params, true)
		require.NoError(t, err, "Case %d: %s", i, currCase.name)

		assert.Equal(t, currCase.wantModified, modified, "Case %d: %s", i, currCase.name)
		for k, v := range currCase.wantContent {
			bytes, err := ioutil.ReadFile(path.Join(currTmpDir, k))
			require.NoError(t, err, "Case %d: %s. File: %s", i, currCase.name, k)
			assert.Equal(t, v, string(bytes), "Case %d: %s. File: %s", i, currCase.name, k)
		}
	}
}

func TestUnlicenseFiles(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir("", "")
	defer cleanup()
	require.NoError(t, err)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			require.NoError(t, err)
		}
	}()

	for i, currCase := range []struct {
		name         string
		params       golicense.LicenseParams
		goFiles      []gofiles.GoFileSpec
		nonGoFiles   map[string]string
		wantModified []string
		wantContent  map[string]string
	}{
		{
			name: "unlicense applied to Go files",
			params: golicense.LicenseParams{
				Header: `// Copyright 2016 Palantir Technologies, Inc.`,
			},
			goFiles: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src: `// Copyright 2016 Palantir Technologies, Inc.
package foo`,
				},
				{
					RelPath: "bar/bar.go",
					Src: `// Copyright 2016 Palantir Technologies, Inc.
// Original comment
package bar`,
				},
			},
			wantModified: []string{
				"bar/bar.go",
				"foo.go",
			},
			wantContent: map[string]string{
				"foo.go": `package foo`,
				"bar/bar.go": `// Original comment
package bar`,
			},
		},
		{
			name: "unlicense not applied to non-Go files",
			params: golicense.LicenseParams{
				Header: `// Copyright 2016 Palantir Technologies, Inc.`,
			},
			nonGoFiles: map[string]string{
				"foo.txt": `// Copyright 2016 Palantir Technologies, Inc.
package foo`,
			},
			wantContent: map[string]string{
				"foo.txt": `// Copyright 2016 Palantir Technologies, Inc.
package foo`,
			},
		},
		{
			name: "unlicense not applied to excluded files",
			params: golicense.LicenseParams{
				Header:  `// Copyright 2016 Palantir Technologies, Inc.`,
				Exclude: matcher.Name("foo.go"),
			},
			goFiles: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src: `// Copyright 2016 Palantir Technologies, Inc.
package foo`,
				},
				{
					RelPath: "bar/bar.go",
					Src: `// Copyright 2016 Palantir Technologies, Inc.
// Original comment
package bar`,
				},
			},
			wantModified: []string{
				"bar/bar.go",
			},
			wantContent: map[string]string{
				"foo.go": `// Copyright 2016 Palantir Technologies, Inc.
package foo`,
				"bar/bar.go": `// Original comment
package bar`,
			},
		},
		{
			name: "unlicense not re-applied to files that already do not have license",
			params: golicense.LicenseParams{
				Header: `// Copyright 2016 Palantir Technologies, Inc.`,
			},
			goFiles: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package foo`,
				},
				{
					RelPath: "bar/bar.go",
					Src: `// Copyright 2016 Palantir Technologies, Inc.
// Original comment
package bar`,
				},
			},
			wantModified: []string{
				"bar/bar.go",
			},
			wantContent: map[string]string{
				"foo.go": `package foo`,
				"bar/bar.go": `// Original comment
package bar`,
			},
		},
		{
			name: "custom license removed from files that match custom matchers",
			params: golicense.LicenseParams{
				Header: `// Copyright 2016 Palantir Technologies, Inc.`,
				CustomHeaders: []golicense.CustomLicenseParam{
					{
						Name:         "Custom Co.",
						Header:       "// Copyright 2016 Custom Co.",
						IncludePaths: []string{"bar/bar.go"},
					},
					{
						Name:         "Baz",
						Header:       "// Copyright 2006 Legacy Inc.",
						IncludePaths: []string{"baz/baz.go"},
					},
				},
			},
			goFiles: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src: `// Copyright 2016 Palantir Technologies, Inc.
package foo`,
				},
				{
					RelPath: "bar/bar.go",
					Src: `// Copyright 2016 Custom Co.
package bar`,
				},
				{
					RelPath: "baz/baz.go",
					Src: `// Copyright 2006 Legacy Inc.
package baz`,
				},
			},
			wantModified: []string{
				"bar/bar.go",
				"baz/baz.go",
				"foo.go",
			},
			wantContent: map[string]string{
				"foo.go":     `package foo`,
				"bar/bar.go": `package bar`,
				"baz/baz.go": `package baz`,
			},
		},
	} {
		currTmpDir, err := ioutil.TempDir(tmpDir, "")
		require.NoError(t, err, "Case %d: %s", i, currCase.name)

		err = os.Chdir(currTmpDir)
		require.NoError(t, err, "Case %d: %s", i, currCase.name)

		_, err = gofiles.Write(currTmpDir, currCase.goFiles)
		require.NoError(t, err, "Case %d: %s", i, currCase.name)
		writeFiles(t, currCase.nonGoFiles)

		files, err := matcher.ListFiles(currTmpDir, matcher.Name(`.+`), nil)
		require.NoError(t, err, "Case %d: %s", i, currCase.name)

		modified, err := golicense.UnlicenseFiles(files, currCase.params, true)
		require.NoError(t, err, "Case %d: %s", i, currCase.name)

		assert.Equal(t, currCase.wantModified, modified, "Case %d: %s", i, currCase.name)
		for k, v := range currCase.wantContent {
			bytes, err := ioutil.ReadFile(path.Join(currTmpDir, k))
			require.NoError(t, err, "Case %d: %s", i, currCase.name)
			assert.Equal(t, v, string(bytes), "Case %d: %s", i, currCase.name)
		}
	}
}

func TestLicenseFilesValidatesParams(t *testing.T) {
	for i, currCase := range []struct {
		name    string
		params  golicense.LicenseParams
		wantErr string
	}{
		{
			name:   "empty configuration valid",
			params: golicense.LicenseParams{},
		},
		{
			name: "empty custom configuration name invalid",
			params: golicense.LicenseParams{
				CustomHeaders: []golicense.CustomLicenseParam{
					{
						Header:       "// Header",
						IncludePaths: []string{""},
					},
				},
			},
			wantErr: "license parameters invalid: custom header entries have blank names: [{Name: Header:// Header IncludePaths:[]}]",
		},
		{
			name: "non-unique custom configuration names invalid",
			params: golicense.LicenseParams{
				CustomHeaders: []golicense.CustomLicenseParam{
					{
						Name:         "foo",
						Header:       "// Header",
						IncludePaths: []string{""},
					},
					{
						Name:         "foo",
						Header:       "// Header",
						IncludePaths: []string{""},
					},
				},
			},
			wantErr: "license parameters invalid: multiple custom header entries have the same name:\n\tfoo: [{Name:foo Header:// Header IncludePaths:[]} {Name:foo Header:// Header IncludePaths:[]}]",
		},
		{
			name: "custom configurations with same paths invalid",
			params: golicense.LicenseParams{
				CustomHeaders: []golicense.CustomLicenseParam{
					{
						Name:   "foo",
						Header: "// Header",
						IncludePaths: []string{
							"foo",
							"bar",
						},
					},
					{
						Name:   "bar",
						Header: "// Header",
						IncludePaths: []string{
							"bar",
							"baz",
						},
					},
					{
						Name:   "ok",
						Header: "// Header",
						IncludePaths: []string{
							"ok",
						},
					},
					{
						Name:   "collides",
						Header: "// Header",
						IncludePaths: []string{
							"bar",
						},
					},
				},
			},
			wantErr: "license parameters invalid: the same path is defined by multiple custom header entries:\n\tbar: foo, bar, collides",
		},
	} {
		_, err := golicense.LicenseFiles(nil, currCase.params, false)
		if currCase.wantErr == "" {
			assert.NoError(t, err, "Case %d: %s", i, currCase.name)
		} else {
			assert.EqualError(t, err, currCase.wantErr, "Case %d: %s", i, currCase.name)
		}
	}
}

func writeFiles(t *testing.T, files map[string]string) {
	for k, v := range files {
		dir := path.Dir(k)
		if dir != "" {
			err := os.MkdirAll(dir, 0755)
			require.NoError(t, err)
		}
		err := ioutil.WriteFile(k, []byte(v), 0644)
		require.NoError(t, err)
	}
}
