package lib

import (
	"fmt"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(Reflect, `

declare namespace reflect {
    export const program: runtime.Program

    export function is<T>(v: any, name: string): v is T

    export function typeOf(v: any): string

    export function isValue(v: any): boolean
    export function isNativeObject(v: any): boolean
    export function isArray(v: any): boolean
    export function isMap(v: any): boolean

    export function getFunction(name: string): Function

    export function call(name: string, ...params: any[]): any

    export function runFunc(name: string, ...params: any[]): any
}


`)
}

var Reflect = []core.NativeFunction{
	core.NativeFunction{
		Name:      "->reflect.program",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			p := vm.Program
			return core.NewObject(&program{prog: p}), nil
		},
	},
	core.NativeFunction{
		Name:      "reflect.is",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0].TypeName()
			b := args[1]
			if b.Type != core.String {
				return core.NullValue, fmt.Errorf("argument 2 must be a string, got %s", b.TypeName())
			}
			return core.NewBool(a == b.ToString()), nil
		},
	},
	core.NativeFunction{
		Name:      "reflect.isValue",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			switch args[0].Type {
			case core.Int, core.Float, core.Bool, core.String:
				return core.FalseValue, nil
			}
			return core.TrueValue, nil
		},
	},
	core.NativeFunction{
		Name:      "reflect.isNativeObject",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			v := args[0].Type == core.Object
			return core.NewBool(v), nil
		},
	},
	core.NativeFunction{
		Name:      "reflect.isArray",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			v := args[0].Type == core.Array
			return core.NewBool(v), nil
		},
	},
	core.NativeFunction{
		Name:      "reflect.isMap",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			v := args[0].Type == core.Map
			return core.NewBool(v), nil
		},
	},
	core.NativeFunction{
		Name:      "reflect.typeOf",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			v := args[0]
			return core.NewString(v.TypeName()), nil
		},
	},
	core.NativeFunction{
		Name:      "reflect.call",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if len(args) == 0 {
				return core.NullValue, fmt.Errorf("expected the function name")
			}

			return vm.RunFunc(args[0].ToString(), args[1:]...)
		},
	},
	core.NativeFunction{
		Name:      "reflect.getFunction",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			name := args[0].ToString()
			fn, ok := vm.Program.Function(name)
			if !ok {
				return core.NullValue, nil
			}

			v := core.NewFunction(fn.Index)
			return v, nil
		},
	},
	core.NativeFunction{
		Name:      "reflect.runFunc",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			l := len(args)
			if l < 1 {
				return core.NullValue, fmt.Errorf("expected at least 1 parameter, got %d", l)
			}

			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("argument must be a string, got %s", args[0].TypeName())
			}

			name := args[0].ToString()

			v, err := vm.RunFunc(name, args[1:]...)
			if err != nil {
				return core.NullValue, err
			}

			return v, nil
		},
	},
}
