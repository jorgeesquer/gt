package lib

import (
	"fmt"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(Bytes, "")
}

var Bytes = []core.NativeFunction{
	core.NativeFunction{
		Name:      "Bytes.prototype.copyAt",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.Int {
				return core.NullValue, fmt.Errorf("expected arg 1 to be int, got %s", args[0].TypeName())
			}

			switch args[1].Type {
			case core.Bytes, core.Array, core.String:
			default:
				return core.NullValue, fmt.Errorf("expected arg 2 to be bytes, got %s", args[1].TypeName())
			}

			a := this.ToBytes()
			start := int(args[0].ToInt())
			b := args[1].ToBytes()

			lenB := len(b)

			if lenB+start > len(a) {
				return core.NullValue, fmt.Errorf("the array has not enough capacity")
			}

			for i := 0; i < lenB; i++ {
				a[i+start] = b[i]
			}

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "Bytes.prototype.append",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if this.Type != core.Bytes {
				return core.NullValue, fmt.Errorf("expected byte array, got %s", this.TypeName())
			}
			a := this.ToBytes()

			b := args[0]
			if b.Type != core.Bytes {
				return core.NullValue, fmt.Errorf("expected array, got %s", b.TypeName())
			}

			c := append(a, b.ToBytes()...)

			return core.NewBytes(c), nil
		},
	},
	core.NativeFunction{
		Name:      "Bytes.prototype.indexOf",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if this.Type != core.Bytes {
				return core.NullValue, fmt.Errorf("expected byte array, got %s", this.TypeName())
			}
			a := this.ToBytes()
			v := byte(args[0].ToInt())

			for i, j := range a {
				if j == v {
					return core.NewInt(i), nil
				}
			}

			return core.NewInt(-1), nil
		},
	},
	core.NativeFunction{
		Name: "Bytes.prototype.reverse",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if this.Type != core.Bytes {
				return core.NullValue, fmt.Errorf("expected byte array, got %s", this.TypeName())
			}
			a := this.ToBytes()
			l := len(a) - 1

			for i, k := 0, l/2; i <= k; i++ {
				a[i], a[l-i] = a[l-i], a[i]
			}

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "Bytes.prototype.slice",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if this.Type != core.Bytes {
				return core.NullValue, fmt.Errorf("expected string array, got %s", this.TypeName())
			}
			a := this.ToBytes()
			l := len(a)

			switch len(args) {
			case 0:
				a = a[0:]
			case 1:
				a = a[int(args[0].ToInt()):]
			case 2:
				start := int(args[0].ToInt())
				if start < 0 || start > l {
					return core.NullValue, fmt.Errorf("index out of range")
				}

				end := start + int(args[1].ToInt())
				if end < 0 || end > l {
					return core.NullValue, fmt.Errorf("index out of range")
				}

				a = a[start:end]
			default:
				return core.NullValue, fmt.Errorf("expected 0, 1 or 2 params, got %d", len(args))
			}

			return core.NewBytes(a), nil
		},
	},
	core.NativeFunction{
		Name:      "Bytes.prototype.range",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if this.Type != core.Bytes {
				return core.NullValue, fmt.Errorf("expected array, called on %s", this.TypeName())
			}
			a := this.ToBytes()
			l := len(a)

			switch len(args) {
			case 0:
				a = a[0:]
			case 1:
				a = a[int(args[0].ToInt()):]
			case 2:
				start := int(args[0].ToInt())
				if start < 0 || start > l {
					return core.NullValue, fmt.Errorf("index out of range")
				}

				end := int(args[1].ToInt())
				if end < 0 || end > l {
					return core.NullValue, fmt.Errorf("index out of range")
				}

				a = a[start:end]
			default:
				return core.NullValue, fmt.Errorf("expected 0, 1 or 2 params, got %d", len(args))
			}

			return core.NewBytes(a), nil
		},
	},
	// core.NativeFunc{
	// 	Name:      "Bytes.prototype.removeAt",
	// 	Arguments: 1,
	// 	Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
	// 		if this.Type != core.BytesType {
	// 			return core.NullValue, fmt.Errorf("expected array, got %s", this.TypeName())
	// 		}

	// 		if err := ValidateArgs(args, core.IntType); err != nil {
	// 			return core.NullValue, err
	// 		}

	// 		obj := this.ToBytes()
	// 		i := int(args[0].ToInt())

	// 		a := obj
	// 		copy(a[i:], a[i+1:])
	// 		a[len(a)-1] = core.NullValue
	// 		obj.Array = a[:len(a)-1]

	// 		return core.NullValue, nil
	// 	},
	// },
}
