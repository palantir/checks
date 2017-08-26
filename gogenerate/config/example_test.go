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

package config_test

import (
	"fmt"

	"github.com/palantir/checks/gogenerate/config"
)

func Example() {
	yml := `
generators:
  foo:
    go-generate-dir: testbar
    gen-paths:
      names:
        - "bar"
      paths:
        - "testbar/output.txt"
    environment:
      GOOS: darwin
`
	cfg, err := config.LoadFromStrings(yml, "")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%q", fmt.Sprintf("%+v", cfg))
	// Output: "{Generators:map[foo:{GoGenDir:testbar GenPaths:{Names:[bar] Paths:[testbar/output.txt]} Environment:map[GOOS:darwin]}]}"
}
