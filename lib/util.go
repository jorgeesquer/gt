package lib

import (
	"fmt"
	"strconv"

	"github.com/gtlang/gt/core"
)

// validate the number of args ant type
func ValidateArgs(args []core.Value, t ...interface{}) error {
	exp := len(t)
	got := len(args)
	if exp != got {
		return fmt.Errorf("expected %d arguments, got %d", exp, got)
	}

	for i, v := range t {
		a := args[i]
		if v != nil && !validateType(v.(core.Type), a.Type) {
			return fmt.Errorf("expected argument %d to be %v, got %s", i, v, a.TypeName())
		}
	}

	return nil
}

// validate that if present, args are of type t
func ValidateOptionalArgs(args []core.Value, t ...core.Type) error {
	exp := len(t)
	got := len(args)
	if got > exp {
		return fmt.Errorf("expected %d arguments max, got %d", exp, got)
	}

	for i, v := range args {
		a := t[i]
		t := v.Type
		if t == core.Undefined {
			continue
		}
		if !validateType(t, a) {
			return fmt.Errorf("expected argument %d to be %v, got %s", i, a, v.TypeName())
		}
	}

	return nil
}

func validateType(v, t core.Type) bool {
	if v == t {
		return true
	}

	if v == core.Bytes && t == core.String {
		return true
	}
	if v == core.String && t == core.Bytes {
		return true
	}

	if v == core.Int && t == core.Float {
		return true
	}
	if v == core.Float && t == core.Int {
		return true
	}

	if v == core.Int && t == core.Bool {
		return true
	}

	return false
}

func ValidateArgRange(args []core.Value, counts ...int) error {
	l := len(args)
	for _, v := range counts {
		if l == v {
			return nil
		}
	}

	var s string
	j := len(counts) - 1
	for i, v := range counts {
		if i == j {
			s += " or "
		} else if i > 0 {
			s += ", "
		}
		s += strconv.Itoa(v)
	}

	return fmt.Errorf("expected %s arguments, got %d", s, l)
}

func runFuncOrClosure(vm *core.VM, fn core.Value, args ...core.Value) error {
	m, err := cloneForAsync(vm)
	if err != nil {
		return err
	}

	switch fn.Type {
	case core.Func:
		_, err := m.RunFuncIndex(fn.ToFunction(), args...)
		return err

	case core.Object:
		c, ok := fn.ToObject().(core.Closure)
		if !ok {
			return fmt.Errorf("expected a function, got: %s", fn.TypeName())
		}
		_, err := m.RunClosure(c, args...)
		return err

	default:
		return fmt.Errorf("expected a function, got: %s", fn.TypeName())
	}
}
