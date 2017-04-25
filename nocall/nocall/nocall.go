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

package nocall

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"regexp"
	"sort"
)

// FuncRef is a reference to a specific function. Matches the string representation of *types.Func, which is of the
// form "func (*net/http.Client).Do(req *net/http.Request) (*net/http.Response, error)".
type FuncRef string

func PrintAllFuncRefs(pkgs []string, stdout io.Writer) error {
	return printFuncRefUsages(pkgs, nil, stdout)
}

func PrintFuncRefUsages(pkgs []string, sigs map[string]string, stdout io.Writer) error {
	if len(sigs) == 0 {
		// if there are no signatures, there will be no output
		return nil
	}
	return printFuncRefUsages(pkgs, sigs, stdout)
}

func printFuncRefUsages(pkgs []string, sigs map[string]string, stdout io.Writer) error {
	for _, currPkg := range pkgs {
		fset := token.NewFileSet()
		parsedPkgs, _ := parser.ParseDir(fset, currPkg, nil, parser.ParseComments)

		var pkgNames []string
		for k := range parsedPkgs {
			pkgNames = append(pkgNames, k)
		}
		sort.Strings(pkgNames)
		for _, k := range pkgNames {
			var fileNames []string
			for currFilename := range parsedPkgs[k].Files {
				fileNames = append(fileNames, currFilename)
			}
			sort.Strings(fileNames)
			for _, currFilename := range fileNames {
				currOutput, err := findFuncRefUsage(currPkg, parsedPkgs[k].Files[currFilename], fset, sigs)
				if err != nil {
					return err
				}

				if len(sigs) == 0 {
					// "all" mode -- print all references
					visitInOrder(currOutput, func(pos token.Position, ref FuncRef) {
						fmt.Fprintf(stdout, "%s: %s\n", pos.String(), ref)
					})
					continue
				}

				// filter out any matches that have a whitelist comment
				filterFuncRefs(currOutput, okCommentRegxp.MatchString)

				visitInOrder(currOutput, func(pos token.Position, ref FuncRef) {
					reason, ok := sigs[string(ref)]
					if !ok {
						return
					}
					if reason == "" {
						reason = fmt.Sprintf("references to %q are not allowed. Remove this reference or whitelist it by adding a comment of the form '// OK: [reason]' to the line before it.", ref)
					}
					fmt.Fprintf(stdout, "%s: %s\n", pos.String(), reason)
				})
			}
		}
	}
	return nil
}

// matches a single-line comment beginning with "// OK: " followed by at least one non-whitespace character.
var okCommentRegxp = regexp.MustCompile(regexp.QuoteMeta(`// OK: `) + `\S.*`)

func filterFuncRefs(funcRefs map[FuncRef]map[token.Position]string, filter func(string) bool) {
	for _, refPosToRefComment := range funcRefs {
		for pos, comment := range refPosToRefComment {
			if filter(comment) {
				delete(refPosToRefComment, pos)
			}
		}
	}
}

func visitInOrder(funcRefs map[FuncRef]map[token.Position]string, visitor func(token.Position, FuncRef)) {
	var allPos []token.Position
	posToFuncRef := make(map[token.Position]FuncRef)

	for funcRef, posToComment := range funcRefs {
		for pos := range posToComment {
			allPos = append(allPos, pos)
			posToFuncRef[pos] = funcRef
		}
	}
	sort.Sort(posSlice(allPos))

	for _, currPos := range allPos {
		visitor(currPos, posToFuncRef[currPos])
	}
}

type posSlice []token.Position

func (a posSlice) Len() int      { return len(a) }
func (a posSlice) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a posSlice) Less(i, j int) bool {
	if a[i].Line != a[j].Line {
		return a[i].Line < a[j].Line
	}
	return a[j].Column < a[j].Column
}

// findFuncRefUsage returns all of the function references in the specified package. If "sigs" is non-empty, then only
// function signature that match a key in the "sigs" map are included; otherwise, all function references are returned.
func findFuncRefUsage(pkgPath string, f *ast.File, fset *token.FileSet, sigs map[string]string) (map[FuncRef]map[token.Position]string, error) {
	rv := make(map[FuncRef]map[token.Position]string)

	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{
		Uses: make(map[*ast.Ident]types.Object),
	}
	if _, err := conf.Check(pkgPath, fset, []*ast.File{f}, info); err != nil {
		return nil, err
	}

	// map from line to comments in file
	lineToComment := make(map[int]string)
	for _, commentGroup := range f.Comments {
		for _, comment := range commentGroup.List {
			lineToComment[fset.Position(comment.Pos()).Line] = comment.Text
		}
	}

	var keys []*ast.Ident
	for k := range info.Uses {
		keys = append(keys, k)
	}
	sort.Sort(identSlice(keys))

	for _, id := range keys {
		obj := info.Uses[id]
		funcPtr, ok := obj.(*types.Func)
		if !ok {
			continue
		}

		currSig := FuncRef(funcPtr.String())

		if len(sigs) > 0 {
			if _, ok := sigs[string(currSig)]; !ok {
				// if sigs is non-empty, skip any entries that don't match the signature
				continue
			}
		}

		lineMap := rv[currSig]
		if lineMap == nil {
			rv[currSig] = make(map[token.Position]string)
			lineMap = rv[currSig]
		}

		currSigPos := fset.Position(id.Pos())
		lineMap[currSigPos] = lineToComment[currSigPos.Line-1]
	}
	return rv, nil
}

type identSlice []*ast.Ident

func (a identSlice) Len() int           { return len(a) }
func (a identSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a identSlice) Less(i, j int) bool { return a[i].Pos() < a[j].Pos() }
