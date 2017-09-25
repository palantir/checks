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

package ptimports

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVendorGrouper(t *testing.T) {
	grouper := newVendoredGrouper("github.com/palantir/checks/")

	for i, currCase := range []struct {
		path  string
		group int
	}{
		{path: "strings", group: 0},
		{path: "net/http", group: 0},
		{path: "github.com/stretchr/testify/assert", group: 1},
		{path: "github.com/palantir/pkg/pkgpath", group: 1},
		{path: "github.com/palantir/checks", group: 2},
		{path: "github.com/palantir/checks/ptimports", group: 2},
	} {
		assert.Equal(t, currCase.group, grouper.importGroup(currCase.path), "Case %d: %s", i, currCase.path)
	}
}
