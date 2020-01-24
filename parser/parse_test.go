package parser

import (
	"regexp"
	"strings"
	"testing"

	"github.com/gtlang/filesystem"
	"github.com/gtlang/gt/ast"
)

func TestParseBasic(t *testing.T) {
	p, err := ParseStr(`
		fmt.println("hi") 
	`)
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, p, `*ast.CallStmt {
		CallExpr *ast.CallExpr {
			Ident *ast.SelectorExpr {
				X *ast.IdentExpr {
					Name string "fmt"
				}
				Sel *ast.IdentExpr {
					Name string "println"
				}
			}
			Args []ast.Expr[
				*ast.ConstantExpr {
					Kind ast.Type STRING
					Value string "hi"
				}
			]
			Spread bool false
		}`)
}

func TestParseImport(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("main.ts", []byte(`
		import * as runtime from "runtime"

		function main() {

		}
	`))

	fs.WritePath("runtime.ts", []byte(`
		export const FOO = 3
	`))

	p, err := Parse(fs, "main.ts")
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, p, `Modules map[string]*ast.File{
		"/runtime": *ast.File {
				Path string "/runtime.ts"
				Stms []ast.Stmt[
						*ast.VarDeclStmt {
								Name string "FOO"
								Value *ast.ConstantExpr {
										Kind ast.Type INT
										Value string "3"
								}
								Exported bool true
								IsEnum bool false
						}
				]
				Comments []*ast.Comment
				Imports []*ast.ImportStmt
				Directives []string
		}`)
}

func assertContains(t *testing.T, p *ast.Module, expected string) {
	s, err := ast.Sprint(p)
	if err != nil {
		t.Fatal(err)
	}

	s = normalize(s)
	expected = normalize(expected)

	if !strings.Contains(s, expected) {
		t.Fatal(s)
	}
}

func normalize(s string) string {
	var reg = regexp.MustCompile(`\s+`)
	s = reg.ReplaceAllString(s, ` `)
	return s
}
