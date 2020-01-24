package binary

import (
	"bytes"
	"testing"

	"github.com/gtlang/gt/core"
)

func TestBinary1(t *testing.T) {
	p := compile(t, `
		//gt: foo var

		function main() { 
			return 2 + 3
		}
	`)

	var buf bytes.Buffer

	err := Write(&buf, p)
	if err != nil {
		t.Fatal("Write: " + err.Error())
	}

	if p, err = Read(&buf); err != nil {
		t.Fatal("Read: " + err.Error())
	}

	if len(p.Directives) != 1 {
		t.Fatal("Expected a directive")
	}

	if p.Directives["foo"] != "var" {
		t.Fatal(p.Directives["foo"])
	}

	assertValue(t, 5, p)
}

func TestBinaryResources(t *testing.T) {
	p := compile(t, `
		function main() {}
	`)

	p.Resources = map[string][]byte{"foo": []byte("bar")}

	var buf bytes.Buffer

	err := Write(&buf, p)
	if err != nil {
		t.Fatal("Write: " + err.Error())
	}

	if p, err = Read(&buf); err != nil {
		t.Fatal("Read: " + err.Error())
	}

	s := string(p.Resources["foo"])

	if s != "bar" {
		t.Fatal(s)
	}
}

func TestBinaryNativeLib(t *testing.T) {
	core.AddNativeFunc(core.NativeFunction{
		Name:      "math.square",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			v := args[0].ToInt()
			return core.NewInt64(v * v), nil
		},
	})

	p := compile(t, `
		function main() {
			return math.square(2)
		}
	`)

	var buf bytes.Buffer

	err := Write(&buf, p)
	if err != nil {
		t.Fatal("Write: " + err.Error())
	}

	if p, err = Read(&buf); err != nil {
		t.Fatal("Read: " + err.Error())
	}

	assertValue(t, 4, p)
}

func TestConstants(t *testing.T) {
	p := compile(t, `
		function main() { 
			return ["aaa", 1, 1.2, true, false, null, undefined, 'a']
		}
	`)

	var buf bytes.Buffer

	err := Write(&buf, p)
	if err != nil {
		t.Fatal("Write: " + err.Error())
	}

	if p, err = Read(&buf); err != nil {
		t.Fatal("Read: " + err.Error())
	}

	v, err := core.NewVM(p).Run()
	if err != nil {
		t.Fatal(err)
	}

	a := v.ToArray()
	if a[0].ToString() != "aaa" {
		t.Fail()
	}
	if a[1].ToInt() != 1 {
		t.Fail()
	}
	if a[2].ToFloat() != 1.2 {
		t.Fail()
	}
	if !a[3].ToBool() {
		t.Fail()
	}
	if a[4].ToBool() {
		t.Fail()
	}
	if a[5] != core.NullValue {
		t.Fail()
	}
	if a[6] != core.UndefinedValue {
		t.Fail()
	}
	if a[7].ToRune() != 'a' {
		t.Fail()
	}
}

func compile(t *testing.T, code string) *core.Program {
	p, err := core.CompileStr(code)
	if err != nil {
		t.Fatal(err)
	}

	return p
}

func assertValue(t *testing.T, expected interface{}, p *core.Program) {
	vm := core.NewVM(p)

	// vm.MaxSteps = 50

	ret, err := vm.Run()
	if err != nil {
		t.Fatal(err)
	}

	v := core.NewValue(expected)

	if ret != v {
		t.Fatalf("Expected %v %T, got %v %T", expected, expected, ret, ret)
	}
}
