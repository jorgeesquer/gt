package lib

import (
	"strings"
	"testing"

	"github.com/gtlang/gt/lib/x/templates"
)

func TestHeader(t *testing.T) {
	code := `<%@ var a = 1 < 2 %>`

	expected := `var a = 1 < 2

function foo(w){
`

	p, _, err := templates.CompileHtml(code, "function foo(w)")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(string(p), expected) {
		t.Fatal(string(p))
	}
}

func TestQuotes(t *testing.T) {
	p, _, err := templates.CompileHtml(`<%== "\"" %>`, "")
	if err != nil {
		t.Fatal(err)
	}

	if string(p) != `w.write("\"")`+"\n" {
		t.Fatal(string(p))
	}
}

func TestBackticks(t *testing.T) {
	p, _, err := templates.CompileHtml("a`b", "")
	if err != nil {
		t.Fatal(err)
	}

	if string(p) != "w.write(`a`)\nw.write(\"`\")\nw.write(`b`)\n" {
		t.Fatal(string(p))
	}
}

func TestTemplates1(t *testing.T) {
	p, _, err := templates.CompileHtml(`FOO<%= bar %> BAR `, "")
	if err != nil {
		t.Fatal(err)
	}

	if string(p) != "w.write(`FOO`)\nw.write(html.encode(bar))\nw.write(` BAR `)\n" {
		t.Fatal(string(p))
	}
}

func Test2(t *testing.T) {
	p, _, err := templates.CompileHtml(`<%== bar %>`, "")
	if err != nil {
		t.Fatal(err)
	}

	if string(p) != "w.write(bar)\n" {
		t.Fatal(string(p))
	}
}

func Test3(t *testing.T) {
	p, _, err := templates.CompileHtml(`<% print("foo") %>`, "")
	if err != nil {
		t.Fatal(err)
	}

	if string(p) != "print(\"foo\")\n" {
		t.Fatal(string(p))
	}
}

func Test4(t *testing.T) {
	p, _, err := templates.CompileHtml(`<% print("<%") %>`, "")
	if err != nil {
		t.Fatal(err)
	}

	if string(p) != "print(\"<%\")\n" {
		t.Fatal(string(p))
	}
}

func Test5(t *testing.T) {
	p, _, err := templates.CompileHtml(`<%= 1 %>`, "function foo(w)")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(string(p), "function foo(w){\nw.write(html.encode(1))") {
		t.Fatal(string(p))
	}
}

func Test6(t *testing.T) {
	source := `<%@ import "foo"; %>
	<%@ import "bar"; %>
	
	test
`

	expect := `import "foo";
import "bar";

function foo(w){
`

	p, _, err := templates.CompileHtml(source, "function foo(w)")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(string(p), expect) {
		t.Fatal(string(p))
	}
}

func Test7(t *testing.T) {
	source := `<%@ 	
var a = 1
var b = 2
%>
`

	expect := `var a = 1
var b = 2

`
	p, _, err := templates.CompileHtml(source, "function foo(w)")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(string(p), expect) {
		t.Fatal(string(p))
	}
}
