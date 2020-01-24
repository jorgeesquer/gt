package lib

import (
	"testing"

	"github.com/gtlang/gt/core"
)

func runTest(t *testing.T, code string, args ...core.Value) core.Value {
	p, err := core.CompileStr(code)
	if err != nil {
		t.Fatal(err)
	}

	vm := core.NewVM(p)
	vm.Trusted = true
	vm.MaxSteps = 300

	// core.Print(p)

	v, err := vm.Run(args...)
	if err != nil {
		t.Fatal(err)
	}

	return v
}

func runExpr(t *testing.T, code string, funcs ...core.NativeFunction) (*core.VM, error) {
	for _, f := range funcs {
		core.AddNativeFunc(f)
	}

	p, err := core.CompileStr(code)
	if err != nil {
		return nil, err
	}

	vm := core.NewVM(p)

	for _, f := range funcs {
		core.AddNativeFunc(f)
	}

	vm.Trusted = true
	vm.MaxSteps = 1000

	_, err = vm.Run()
	return vm, err
}

func assertRegister(t *testing.T, register string, expected interface{}, code string) {
	vm, err := runExpr(t, code)
	if err != nil {
		t.Fatal(err)
	}

	ex := core.NewValue(expected)

	v, _ := vm.RegisterValue(register)
	if v != ex {
		t.Fatalf("Expected %v, got %v", ex, v)
	}
}
