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

package integration_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/nmiyake/pkg/dirs"
	"github.com/nmiyake/pkg/gofiles"
	"github.com/palantir/godel/pkg/products"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoCall(t *testing.T) {
	cli, err := products.Bin("nocall")
	require.NoError(t, err)

	wd, err := os.Getwd()
	require.NoError(t, err)

	tmpDir, cleanup, err := dirs.TempDir(wd, "")
	defer cleanup()
	require.NoError(t, err)

	for i, currCase := range []struct {
		name          string
		filesToCreate []gofiles.GoFileSpec
		args          []string
		wantStdout    string
	}{
		{
			name: "Basic case",
			filesToCreate: []gofiles.GoFileSpec{
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
			args: []string{
				"--json",
				`{"func (*net/http.Client).Do(req *net/http.Request) (*net/http.Response, error)": ""}`,
				"./foo",
			},
			wantStdout: "foo/foo.go:9:21: references to \"func (*net/http.Client).Do(req *net/http.Request) (*net/http.Response, error)\" are not allowed. Remove this reference or whitelist it by adding a comment of the form '// OK: [reason]' to the line before it.\n",
		},
		{
			name: "All flag",
			filesToCreate: []gofiles.GoFileSpec{
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
			args: []string{
				"--all",
				"./foo",
			},
			wantStdout: "foo/foo.go:9:21: func (*net/http.Client).Do(req *net/http.Request) (*net/http.Response, error)\n",
		},
	} {
		currCaseTmpDir, err := ioutil.TempDir(tmpDir, "")
		require.NoError(t, err)

		_, err = gofiles.Write(currCaseTmpDir, currCase.filesToCreate)
		require.NoError(t, err, "Case %d", i)

		var output []byte
		func() {
			err := os.Chdir(currCaseTmpDir)
			defer func() {
				err := os.Chdir(wd)
				require.NoError(t, err)
			}()
			require.NoError(t, err)

			cmd := exec.Command(cli, currCase.args...)
			output, err = cmd.CombinedOutput()
			require.NoError(t, err, "Case %d: %s\nOutput: %s", i, currCase.name, string(output))
		}()

		assert.Equal(t, currCase.wantStdout, string(output), "Case %d: %s\nOutput:\n%s", i, currCase.name, string(output))
	}
}
