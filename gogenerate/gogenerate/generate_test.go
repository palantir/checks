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

package gogenerate_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/nmiyake/pkg/dirs"
	"github.com/nmiyake/pkg/gofiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/palantir/checks/gogenerate/config"
	"github.com/palantir/checks/gogenerate/gogenerate"
)

func TestGenerate(t *testing.T) {
	testDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	specs := []gofiles.GoFileSpec{
		{
			RelPath: "gen/testbar.go",
			Src: `package testbar

//go:generate go run generator_main.go
`,
		},
		{
			RelPath: "gen/generator_main.go",
			Src: `// +build ignore

package main

import (
	"io/ioutil"
)

func main() {
	if err := ioutil.WriteFile("output.txt", []byte("foo-output"), 0644); err != nil {
		panic(err)
	}
}
`,
		},
	}
	_, err = gofiles.Write(testDir, specs)
	require.NoError(t, err)

	const configYML = `
generators:
  foo:
    go-generate-dir: gen
    gen-paths:
      paths:
        - "gen/output.txt"
`
	cfg, err := config.LoadFromStrings(configYML, "")
	require.NoError(t, err)

	err = gogenerate.Run(testDir, cfg, false, os.Stdout)
	require.NoError(t, err)

	outputTxt, err := ioutil.ReadFile(path.Join(testDir, "gen", "output.txt"))
	require.NoError(t, err)

	assert.Equal(t, "foo-output", string(outputTxt))
}

func TestGenerateEnvVars(t *testing.T) {
	testDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	specs := []gofiles.GoFileSpec{
		{
			RelPath: "gen/testbar.go",
			Src: `package testbar

//go:generate go run generator_main.go
`,
		},
		{
			RelPath: "gen/generator_main.go",
			Src: `// +build ignore

package main

import (
	"io/ioutil"
	"os"
)

func main() {
	if err := ioutil.WriteFile("output.txt", []byte(os.Getenv("GOGEN_VAR")), 0644); err != nil {
		panic(err)
	}
}
`,
		},
	}
	_, err = gofiles.Write(testDir, specs)
	require.NoError(t, err)

	const configYML = `
generators:
  foo:
    go-generate-dir: gen
    gen-paths:
      paths:
        - "gen/output.txt"
    environment:
      GOGEN_VAR: test-val
`
	cfg, err := config.LoadFromStrings(configYML, "")
	require.NoError(t, err)

	err = gogenerate.Run(testDir, cfg, false, os.Stdout)
	require.NoError(t, err)

	outputTxt, err := ioutil.ReadFile(path.Join(testDir, "gen", "output.txt"))
	require.NoError(t, err)

	assert.Equal(t, "test-val", string(outputTxt))
}

func TestGenerateVerifyErrors(t *testing.T) {
	testDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	for currCaseNum, currCase := range []struct {
		name         string
		configYML    string
		gofiles      []gofiles.GoFileSpec
		initialState func(caseNum int, caseName, testDir string)
		wantError    string
	}{
		{
			name: "generated output contains new file",
			configYML: `
generators:
  foo:
    go-generate-dir: gen
    gen-paths:
      paths:
        - "gen/generated"
`,
			gofiles: []gofiles.GoFileSpec{
				{
					RelPath: "gen/testbar.go",
					Src: `package testbar

//go:generate go run generator_main.go
`,
				},
				{
					RelPath: "gen/generator_main.go",
					Src: `// +build ignore

package main

import (
	"io/ioutil"
	"os"
)

func main() {
	if err := os.MkdirAll("generated", 0755); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile("generated/output.txt", []byte("foo-output"), 0644); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile("generated/output-2.txt", []byte("foo-output"), 0644); err != nil {
		panic(err)
	}
}
`,
				},
			},
			initialState: func(caseNum int, caseName, testDir string) {
				err := os.MkdirAll(path.Join(testDir, "gen", "generated"), 0755)
				require.NoError(t, err, "Case %d: %s", caseNum, caseName)
				err = ioutil.WriteFile(path.Join(testDir, "gen", "generated", "output.txt"), []byte("foo-output"), 0644)
				require.NoError(t, err, "Case %d: %s", caseNum, caseName)
			},
			wantError: `Generators produced output that differed from what already exists: [foo]
  foo:
    gen/generated/output-2.txt: did not exist before, now exists`,
		},
		{
			name: "generated output removes existing file",
			configYML: `
generators:
  foo:
    go-generate-dir: gen
    gen-paths:
      paths:
        - "gen/generated"
`,
			gofiles: []gofiles.GoFileSpec{
				{
					RelPath: "gen/testbar.go",
					Src: `package testbar

//go:generate go run generator_main.go
`,
				},
				{
					RelPath: "gen/generator_main.go",
					Src: `// +build ignore

package main

import (
	"io/ioutil"
	"os"
)

func main() {
	if err := os.MkdirAll("generated", 0755); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile("generated/output.txt", []byte("foo-output"), 0644); err != nil {
		panic(err)
	}
	if err := os.Remove("generated/output-2.txt"); err != nil {
		panic(err)
	}
}
`,
				},
			},
			initialState: func(caseNum int, caseName, testDir string) {
				err := os.MkdirAll(path.Join(testDir, "gen", "generated"), 0755)
				require.NoError(t, err, "Case %d: %s", caseNum, caseName)
				err = ioutil.WriteFile(path.Join(testDir, "gen", "generated", "output.txt"), []byte("foo-output"), 0644)
				require.NoError(t, err, "Case %d: %s", caseNum, caseName)
				err = ioutil.WriteFile(path.Join(testDir, "gen", "generated", "output-2.txt"), []byte("foo-output"), 0644)
				require.NoError(t, err, "Case %d: %s", caseNum, caseName)
			},
			wantError: `Generators produced output that differed from what already exists: [foo]
  foo:
    gen/generated/output-2.txt: existed before, no longer exists`,
		},
		{
			name: "generated output changes file to directory",
			configYML: `
generators:
  foo:
    go-generate-dir: gen
    gen-paths:
      paths:
        - "gen/generated"
`,
			gofiles: []gofiles.GoFileSpec{
				{
					RelPath: "gen/testbar.go",
					Src: `package testbar

//go:generate go run generator_main.go
`,
				},
				{
					RelPath: "gen/generator_main.go",
					Src: `// +build ignore

package main

import (
	"os"
)

func main() {
	if err := os.MkdirAll("generated", 0755); err != nil {
		panic(err)
	}
	if err := os.RemoveAll("generated/output"); err != nil {
		panic(err)
	}
	if err := os.MkdirAll("generated/output", 0755); err != nil {
		panic(err)
	}
}
`,
				},
			},
			initialState: func(caseNum int, caseName, testDir string) {
				err := os.MkdirAll(path.Join(testDir, "gen", "generated"), 0755)
				require.NoError(t, err, "Case %d: %s", caseNum, caseName)
				err = ioutil.WriteFile(path.Join(testDir, "gen", "generated", "output"), []byte("foo-output"), 0644)
				require.NoError(t, err, "Case %d: %s", caseNum, caseName)
			},
			wantError: `Generators produced output that differed from what already exists: [foo]
  foo:
    gen/generated/output: was previously a file, is now a directory`,
		},
		{
			name: "generated output changes directory to file",
			configYML: `
generators:
  foo:
    go-generate-dir: gen
    gen-paths:
      paths:
        - "gen/generated"
`,
			gofiles: []gofiles.GoFileSpec{
				{
					RelPath: "gen/testbar.go",
					Src: `package testbar

//go:generate go run generator_main.go
`,
				},
				{
					RelPath: "gen/generator_main.go",
					Src: `// +build ignore

package main

import (
	"io/ioutil"
	"os"
)

func main() {
	if err := os.MkdirAll("generated", 0755); err != nil {
		panic(err)
	}
	if err := os.RemoveAll("generated/output"); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile("generated/output", []byte("foo-output"), 0644); err != nil {
		panic(err)
	}
}
`,
				},
			},
			initialState: func(caseNum int, caseName, testDir string) {
				err := os.MkdirAll(path.Join(testDir, "gen", "generated", "output"), 0755)
				require.NoError(t, err, "Case %d: %s", caseNum, caseName)
			},
			wantError: `Generators produced output that differed from what already exists: [foo]
  foo:
    gen/generated/output: was previously a directory, is now a file`,
		},
		{
			name: "generated output differs",
			configYML: `
generators:
  foo:
    go-generate-dir: gen
    gen-paths:
      paths:
        - "gen/output.txt"
`,
			gofiles: []gofiles.GoFileSpec{
				{
					RelPath: "gen/testbar.go",
					Src: `package testbar

//go:generate go run generator_main.go
`,
				},
				{
					RelPath: "gen/generator_main.go",
					Src: `// +build ignore

package main

import (
	"io/ioutil"
)

func main() {
	if err := ioutil.WriteFile("output.txt", []byte("foo-output"), 0644); err != nil {
		panic(err)
	}
}
`,
				},
			},
			initialState: func(caseNum int, caseName, testDir string) {
				err := ioutil.WriteFile(path.Join(testDir, "gen", "output.txt"), []byte("bar-output-baz"), 0644)
				require.NoError(t, err, "Case %d: %s", caseNum, caseName)
			},
			wantError: `Generators produced output that differed from what already exists: [foo]
  foo:
    gen/output.txt: previously had checksum 0fd6feace2703f1be2b4d05ef9931b70627e46a0dcd5c32acc460e392eb0c537, now has checksum 380a300b764683667309818ff127a401c6ea6ab1959f386fe0f05505d660ba37`,
		},
	} {
		currCaseDir, err := ioutil.TempDir(testDir, "")
		require.NoError(t, err, "Case %d: %s", currCaseNum, currCase.name)

		_, err = gofiles.Write(currCaseDir, currCase.gofiles)
		require.NoError(t, err, "Case %d: %s", currCaseNum, currCase.name)

		cfg, err := config.LoadFromStrings(currCase.configYML, "")
		require.NoError(t, err, "Case %d: %s", currCaseNum, currCase.name)

		if currCase.initialState != nil {
			currCase.initialState(currCaseNum, currCase.name, currCaseDir)
		}

		err = gogenerate.Run(currCaseDir, cfg, true, os.Stdout)
		require.Error(t, err, fmt.Sprintf("Case %d: %s", currCaseNum, currCase.name))

		assert.EqualError(t, err, currCase.wantError, "Case %d: %s\n%s", currCaseNum, currCase.name, err.Error())
	}
}
