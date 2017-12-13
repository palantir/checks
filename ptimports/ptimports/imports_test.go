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

package ptimports_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/palantir/checks/ptimports/ptimports"
)

func TestPtImports(t *testing.T) {
	for i, tc := range []struct {
		name string
		in   string
		want string
	}{
		{
			"Groups imports based on builtin, external, and project-local",
			`package foo

import "github.com/palantir/checks/ptimports/ptimports"
import "bytes"
import "golang.org/x/tools/imports"

func Foo() {
	_ = bytes.Buffer{}
	_ = ptimports.Process
	_ = imports.Process
}
`,
			`package foo

import (
	"bytes"

	"golang.org/x/tools/imports"

	"github.com/palantir/checks/ptimports/ptimports"
)

func Foo() {
	_ = bytes.Buffer{}
	_ = ptimports.Process
	_ = imports.Process
}
`,
		},
		{
			"CGo import with multi-line comment",
			`package foo

// import "C"

import "unsafe"
import "io"

/*
#include <stdio.h>
#include <stdlib.h>

void myprint(char* s) {
	printf("%s\n", s);
}
*/
import "C"
import "archive/tar"


func Example() {
	/*
	multi-line comment
	 */
	cs := C.CString("Hello from stdio\n")
	C.myprint(cs)
	C.free(unsafe.Pointer(cs))

	// inline comment
	_ = io.Copy
	_ = tar.ErrFieldTooLong

}
`,
			`package foo

// import "C"

/*
#include <stdio.h>
#include <stdlib.h>

void myprint(char* s) {
	printf("%s\n", s);
}
*/
import "C"

import (
	"archive/tar"
	"io"
	"unsafe"
)

func Example() {
	/*
		multi-line comment
	*/
	cs := C.CString("Hello from stdio\n")
	C.myprint(cs)
	C.free(unsafe.Pointer(cs))

	// inline comment
	_ = io.Copy
	_ = tar.ErrFieldTooLong

}
`,
		},
		{
			"CGo import with single-line and multi-line comments",
			`package foo

// #include <stdio.h>
// #include <stdlib.h>
import "C"
import "unsafe"

/*
#include <stdio.h>
#include <stdlib.h>

void myprint(char* s) {
	printf("%s\n", s);
}
*/
import "C"

func Print(s string) {
	cs := C.CString(s)
	C.fputs(cs, (*C.FILE)(C.stdout))
	C.free(unsafe.Pointer(cs))
}
`,
			`package foo

// #include <stdio.h>
// #include <stdlib.h>
import "C"

/*
#include <stdio.h>
#include <stdlib.h>

void myprint(char* s) {
	printf("%s\n", s);
}
*/
import "C"

import (
	"unsafe"
)

func Print(s string) {
	cs := C.CString(s)
	C.fputs(cs, (*C.FILE)(C.stdout))
	C.free(unsafe.Pointer(cs))
}
`,
		},
	} {
		got, err := ptimports.Process("test.go", []byte(tc.in))
		require.NoError(t, err, "Case %d: %s", i, tc.name)
		assert.Equal(t, tc.want, string(got), "Case %d: %s", i, tc.name)
	}
}
