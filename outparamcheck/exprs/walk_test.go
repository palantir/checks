// Copyright 2013 Kamil Kisiel
// Modifications copyright 2016 Palantir Technologies, Inc.
// Licensed under the MIT License. See LICENSE in the project root
// for license information.

package exprs_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/palantir/checks/outparamcheck/exprs"
)

var stmtCases = []struct {
	stmtInput string
	expected  []string
}{
	{
		stmtInput: `end: return nil`,
		expected: []string{
			"*ast.Ident", // end
			"*ast.Ident", // nil
		},
	},
	{
		stmtInput: `fmt.Println("str", 0)`,
		expected: []string{
			"*ast.CallExpr",     // fmt.Println("str", 0)
			"*ast.SelectorExpr", // fmt.Println
			"*ast.Ident",        // fmt
			"*ast.Ident",        // Println
			"*ast.BasicLit",     // "str"
			"*ast.BasicLit",     // 0
		},
	},
	{
		stmtInput: `ch <- true`,
		expected: []string{
			"*ast.Ident", // ch
			"*ast.Ident", // true
		},
	},
	{
		stmtInput: `C++`,
		expected: []string{
			"*ast.Ident", // C
		},
	},
	{
		stmtInput: `x := 0`,
		expected: []string{
			"*ast.Ident",    // x
			"*ast.BasicLit", // 0
		},
	},
	{
		stmtInput: `go f(x)`,
		expected: []string{
			"*ast.CallExpr", // f(x)
			"*ast.Ident",    // f
			"*ast.Ident",    // x
		},
	},
	{
		stmtInput: `defer f(x)`,
		expected: []string{
			"*ast.CallExpr", // f(x)
			"*ast.Ident",    // f
			"*ast.Ident",    // x
		},
	},
	{
		stmtInput: `return x, y, z`,
		expected: []string{
			"*ast.Ident", // x
			"*ast.Ident", // y
			"*ast.Ident", // z
		},
	},
	{
		stmtInput: `return`,
		expected:  []string{},
	},
	{
		stmtInput: `goto x`,
		expected: []string{
			"*ast.Ident", // x
		},
	},
	{
		stmtInput: `if true { return nil }`,
		expected: []string{
			"*ast.Ident", // true
			"*ast.Ident", // nil
		},
	},
	{
		stmtInput: `switch x { case 0, 1: return x; default: return 0 }`,
		expected: []string{
			"*ast.Ident",    // x
			"*ast.BasicLit", // 0
			"*ast.BasicLit", // 1
			"*ast.Ident",    // x
			"*ast.BasicLit", // 0
		},
	},
	{
		stmtInput: `for i := 0; i < 10; i++ { return i }`,
		expected: []string{
			"*ast.BinaryExpr", // i < 10
			"*ast.Ident",      // i
			"*ast.BasicLit",   // 10
			"*ast.Ident",      // i (in i := 0)
			"*ast.BasicLit",   // 0
			"*ast.Ident",      // i (in i++)
			"*ast.Ident",      // i (in return i)
		},
	},
	{
		stmtInput: `for i := range x { return i }`,
		expected: []string{
			"*ast.Ident", // i
			"*ast.Ident", // x
			"*ast.Ident", // i
		},
	},
	{
		stmtInput: `for {}`,
		expected:  []string{},
	},
	{
		stmtInput: `type t int`,
		expected: []string{
			"*ast.Ident", // t
			"*ast.Ident", // int
		},
	},
}

var exprCases = []struct {
	exprInput string
	expected  []string
}{
	{
		exprInput: `[...]a{}`,
		expected: []string{
			"*ast.CompositeLit", // [...]a{}
			"*ast.ArrayType",    // [...]a
			"*ast.Ellipsis",     // ...
			"*ast.Ident",        // a
		},
	},
	{
		exprInput: `(b)`,
		expected: []string{
			"*ast.ParenExpr", // (b)
			"*ast.Ident",     // b
		},
	},
	{
		exprInput: `c.d`,
		expected: []string{
			"*ast.SelectorExpr", // c.d
			"*ast.Ident",        // c
			"*ast.Ident",        // d
		},
	},
	{
		exprInput: `e[f]`,
		expected: []string{
			"*ast.IndexExpr", // e[f]
			"*ast.Ident",     // e
			"*ast.Ident",     // f
		},
	},
	{
		exprInput: `g[:]`,
		expected: []string{
			"*ast.SliceExpr", // g[:]
			"*ast.Ident",     // g
		},
	},
	{
		exprInput: `h[i:]`,
		expected: []string{
			"*ast.SliceExpr", // h[i:]
			"*ast.Ident",     // h
			"*ast.Ident",     // i
		},
	},
	{
		exprInput: `j[:k]`,
		expected: []string{
			"*ast.SliceExpr", // j[:k]
			"*ast.Ident",     // j
			"*ast.Ident",     // k
		},
	},
	{
		exprInput: `l[m:n]`,
		expected: []string{
			"*ast.SliceExpr", // l[m:n]
			"*ast.Ident",     // l
			"*ast.Ident",     // m
			"*ast.Ident",     // n
		},
	},
	{
		exprInput: `o[p:q:r]`,
		expected: []string{
			"*ast.SliceExpr", // o[p:q:r]
			"*ast.Ident",     // o
			"*ast.Ident",     // p
			"*ast.Ident",     // q
			"*ast.Ident",     // r
		},
	},
	{
		exprInput: `s.(t)`,
		expected: []string{
			"*ast.TypeAssertExpr", // s.(t)
			"*ast.Ident",          // s
			"*ast.Ident",          // t
		},
	},
	{
		exprInput: `u(v...)`,
		expected: []string{
			"*ast.CallExpr", // u(v...)
			"*ast.Ident",    // u
			"*ast.Ident",    // v
		},
	},
	{
		exprInput: `*w`,
		expected: []string{
			"*ast.StarExpr", // *w
			"*ast.Ident",    // w
		},
	},
	{
		exprInput: `&x`,
		expected: []string{
			"*ast.UnaryExpr", // &x
			"*ast.Ident",     // x
		},
	},
	{
		exprInput: `_{y: z}`,
		expected: []string{
			"*ast.CompositeLit", // _{y: z}
			"*ast.Ident",        // _
			"*ast.KeyValueExpr", // y: z
			"*ast.Ident",        // y
			"*ast.Ident",        // z
		},
	},
}

var typeCases = []struct {
	typeInput string
	expected  []string
}{
	{
		typeInput: `[1]A`,
		expected: []string{
			"*ast.ArrayType", // [1]A
			"*ast.BasicLit",  // 1
			"*ast.Ident",     // A
		},
	},
	{
		typeInput: `struct{B C "D"}`,
		expected: []string{
			"*ast.StructType", // struct{B C "D"}
			"*ast.Ident",      // B
			"*ast.Ident",      // C
			"*ast.BasicLit",   // "D"
		},
	},
	{
		typeInput: `func(E F) (G H)`,
		expected: []string{
			"*ast.FuncType", // func(E F) (G H)
			"*ast.Ident",    // E
			"*ast.Ident",    // F
			"*ast.Ident",    // G
			"*ast.Ident",    // H
		},
	},
	{
		typeInput: `interface{I()}`,
		expected: []string{
			"*ast.InterfaceType", // interface{I()}
			"*ast.Ident",         // I
			"*ast.FuncType",      // func()
		},
	},
	{
		typeInput: `map[J]K`,
		expected: []string{
			"*ast.MapType", // map[J]K
			"*ast.Ident",   // J
			"*ast.Ident",   // K
		},
	},
	{
		typeInput: `chan L`,
		expected: []string{
			"*ast.ChanType", // chan L
			"*ast.Ident",    // L
		},
	},
	{
		typeInput: `<-chan M`,
		expected: []string{
			"*ast.ChanType", // <-chan M
			"*ast.Ident",    // M
		},
	},
	{
		typeInput: `chan<- N`,
		expected: []string{
			"*ast.ChanType", // chan<- N
			"*ast.Ident",    // N
		},
	},
}

func testWalk(t *testing.T, skel, input string, expected []string) {
	prog := fmt.Sprintf(skel, input)
	file, err := parser.ParseFile(token.NewFileSet(), "", prog, 0)
	if assert.NoError(t, err, input) {
		v := &testVisitor{[]string{}}
		exprs.Walk(v, file.Decls[0])
		assert.Equal(t, expected, v.visited, input)
	}
}

func TestStmt(t *testing.T) {
	skel := `
		package irrelevant
		func main() {
			%v
		}
	`
	preamble := []string{
		"*ast.Ident",    // main
		"*ast.FuncType", // func()
	}
	for _, test := range stmtCases {
		expected := append(preamble, test.expected...)
		testWalk(t, skel, test.stmtInput, expected)
	}
}

func TestExpr(t *testing.T) {
	skel := `
		package irrelevant
		func main() {
			return %v
		}
	`
	preamble := []string{
		"*ast.Ident",    // main
		"*ast.FuncType", // func()
	}
	for _, test := range exprCases {
		expected := append(preamble, test.expected...)
		testWalk(t, skel, test.exprInput, expected)
	}
}

func TestType(t *testing.T) {
	skel := `
		package irrelevant
		func main() {
			var _ %v
		}
	`
	preamble := []string{
		"*ast.Ident",    // main
		"*ast.FuncType", // func()
		"*ast.Ident",    // _
	}
	for _, test := range typeCases {
		expected := append(preamble, test.expected...)
		testWalk(t, skel, test.typeInput, expected)
	}
}

type testVisitor struct {
	visited []string
}

func (v *testVisitor) Visit(expr ast.Expr) {
	v.visited = append(v.visited, fmt.Sprintf("%T", expr))
}
