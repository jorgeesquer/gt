package lib

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(libArray, `
 
interface Array<T> {
    [n: number]: T
    slice(start?: number, count?: number): Array<T>
    range(start?: number, end?: number): Array<T>
    append(v: T[]): T[]
    push(...v: T[]): void
    pushRange(v: T[]): void
    copyAt(i: number, v: T[]): void
    length: number
    insertAt(i: number, v: T): void
    removeAt(i: number): void
    removeAt(from: number, to: number): void
    indexOf(v: T): number
    join(sep: string): T
    sort(comprarer: (a: T, b: T) => boolean): void
    equals(other: Array<T>): boolean;
    any(func: (t: T) => any): boolean;
    all(func: (t: T) => any): boolean;
    contains<T>(t: T): boolean;
    remove<T>(t: T): void;
    first(): T;
    last(): T;
    first(func?: (t: T) => any): T;
    last(func?: (t: T) => any): T;
    firstIndex(func: (t: T) => any): number;
    select<K>(func: (t: T) => K): Array<K>;
    selectMany<K>(func: (t: T) => K): K;
    distinct<K>(func?: (t: K) => any): Array<K>;
    where(func: (t: T) => any): Array<T>;
    groupBy(func: (t: T) => string | number): KeyIndexer<T[]>;
    sum<K extends number>(): number;
    sum<K extends number>(func: (t: T) => K): number;
    min(func: (t: T) => number): number;
    max(func: (t: T) => number): number;
    count(func: (t: T) => any): number;
}

declare namespace array {
    /**
     * Create a new array with size.
     */
    export function make<T>(size: number, capacity?: number): Array<T>

    /**
     * Create a new array of bytes with size.
     */
    export function bytes(size: number, capacity?: number): byte[]
}
	`)
}

var libArray = []core.NativeFunction{
	core.NativeFunction{
		Name:      "array.make",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			ln := len(args)

			if ln > 0 {
				if args[0].Type != core.Int {
					return core.NullValue, fmt.Errorf("expected arg 1 to be int, got %s", args[0].TypeName())
				}
			}

			if ln > 1 {
				if args[1].Type != core.Int {
					return core.NullValue, fmt.Errorf("expected arg 2 to be int, got %s", args[1].TypeName())
				}
			}
			var size, cap int64
			switch len(args) {
			case 1:
				size = args[0].ToInt()
				return core.NewArray(int(size)), nil

			case 2:
				size = args[0].ToInt()
				cap = args[1].ToInt()
				a := make([]core.Value, size, cap)
				return core.NewArrayValues(a), nil

			default:
				return core.NullValue, fmt.Errorf("expected 1 or 2 arguments, got %d", len(args))
			}
		},
	},
	core.NativeFunction{
		Name:      "array.bytes",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			ln := len(args)

			if ln > 0 {
				if args[0].Type != core.Int {
					return core.NullValue, fmt.Errorf("expected arg 1 to be int, got %s", args[0].TypeName())
				}
			}

			if ln > 1 {
				if args[1].Type != core.Int {
					return core.NullValue, fmt.Errorf("expected arg 2 to be int, got %s", args[1].TypeName())
				}
			}

			var size, cap int64
			switch len(args) {
			case 1:
				size = args[0].ToInt()
				return core.NewBytes(make([]byte, size)), nil

			case 2:
				size = args[0].ToInt()
				cap = args[1].ToInt()
				return core.NewBytes(make([]byte, size, cap)), nil

			default:
				return core.NullValue, fmt.Errorf("expected 1 or 2 arguments, got %d", len(args))
			}
		},
	},
	core.NativeFunction{
		Name:      "Array.prototype.copyAt",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.Int {
				return core.NullValue, fmt.Errorf("expected arg 1 to be int, got %s", args[0].TypeName())
			}
			if args[1].Type != core.Array {
				return core.NullValue, fmt.Errorf("expected arg 2 to be array, got %s", args[1].TypeName())
			}

			a := this.ToArray()
			start := int(args[0].ToInt())
			b := args[1].ToArray()

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
		Name:      "Array.prototype.any",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}

			funcIndex := -1
			var closure core.Closure

			b := args[0]
			switch b.Type {
			case core.Func:
				funcIndex = b.ToFunction()

			case core.Object:
				c, ok := b.ToObject().(core.Closure)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
				}
				closure = c

			default:
				return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
			}

			for _, v := range a {
				var r core.Value
				if funcIndex != -1 {
					r, err = vm.RunFuncIndex(funcIndex, v)
				} else {
					r, err = vm.RunClosure(closure, v)
				}
				if err != nil {
					return core.NullValue, err
				}

				var any bool

				switch r.Type {
				case core.Int:
					any = r.ToInt() != 0

				case core.Float:
					any = r.ToFloat() != 0

				case core.Bool:
					any = r.ToBool()

				case core.Null, core.Undefined:
					any = false

				default:
					any = true
				}

				if any {
					return core.TrueValue, nil
				}
			}

			return core.FalseValue, nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.all",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}

			if len(a) == 0 {
				return core.FalseValue, nil
			}

			funcIndex := -1
			var closure core.Closure

			b := args[0]
			switch b.Type {
			case core.Func:
				funcIndex = b.ToFunction()

			case core.Object:
				c, ok := b.ToObject().(core.Closure)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
				}
				closure = c

			default:
				return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
			}

			for _, v := range a {
				var r core.Value
				if funcIndex != -1 {
					r, err = vm.RunFuncIndex(funcIndex, v)
				} else {
					r, err = vm.RunClosure(closure, v)
				}
				if err != nil {
					return core.NullValue, err
				}

				switch r.Type {
				case core.Int:
					if r.ToInt() == 0 {
						return core.FalseValue, nil
					}

				case core.Float:
					if r.ToFloat() == 0 {
						return core.FalseValue, nil
					}

				case core.Bool:
					if !r.ToBool() {
						return core.FalseValue, nil
					}

				case core.Null, core.Undefined:
					return core.FalseValue, nil

				default:
					return core.FalseValue, nil
				}
			}

			return core.TrueValue, nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.contains",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}
			b := args[0]

			for _, v := range a {
				if v.Equals(b) {
					return core.TrueValue, nil
				}
			}

			return core.FalseValue, nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.remove",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := this.ToArray()
			b := args[0]

			for i, v := range a {
				if v.Equals(b) {
					obj := this.ToArrayObject()
					a := obj.Array
					copy(a[i:], a[i+1:])
					a[len(a)-1] = core.NullValue
					obj.Array = a[:len(a)-1]
					break
				}
			}

			return core.NullValue, nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.firstIndex",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}

			funcIndex := -1
			var closure core.Closure

			b := args[0]
			switch b.Type {
			case core.Func:
				funcIndex = b.ToFunction()

			case core.Object:
				c, ok := b.ToObject().(core.Closure)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
				}
				closure = c

			default:
				return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
			}

			for i, v := range a {
				var r core.Value
				if funcIndex != -1 {
					r, err = vm.RunFuncIndex(funcIndex, v)
				} else {
					r, err = vm.RunClosure(closure, v)
				}
				if err != nil {
					return core.NullValue, err
				}

				var found bool

				switch r.Type {
				case core.Int:
					found = r.ToInt() != 0

				case core.Float:
					found = r.ToFloat() != 0

				case core.Bool:
					found = r.ToBool()

				case core.Null, core.Undefined:
					found = false

				default:
					found = true
				}

				if found {
					return core.NewInt(i), nil
				}
			}

			return core.NewInt(-1), nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.first",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			items, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}

			if len(args) == 0 {
				if len(items) > 0 {
					return items[0], nil
				} else {
					return core.NullValue, nil
				}
			}

			funcIndex := -1
			var closure core.Closure

			b := args[0]
			switch b.Type {
			case core.Func:
				funcIndex = b.ToFunction()

			case core.Object:
				c, ok := b.ToObject().(core.Closure)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
				}
				closure = c

			default:
				return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
			}

			for i, item := range items {
				var r core.Value
				if funcIndex != -1 {
					r, err = vm.RunFuncIndex(funcIndex, item)
				} else {
					r, err = vm.RunClosure(closure, item)
				}
				if err != nil {
					return core.NullValue, err
				}

				var found bool

				switch r.Type {
				case core.Int:
					found = r.ToInt() != 0

				case core.Float:
					found = r.ToFloat() != 0

				case core.Bool:
					found = r.ToBool()

				case core.Null, core.Undefined:
					found = false

				default:
					// any non null value is considered a match
					found = true
				}

				if found {
					return items[i], nil
				}
			}

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "Array.prototype.last",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}

			if len(args) == 0 {
				if l := len(a); l > 0 {
					return a[l-1], nil
				} else {
					return core.NullValue, nil
				}
			}

			funcIndex := -1
			var closure core.Closure

			b := args[0]
			switch b.Type {
			case core.Func:
				funcIndex = b.ToFunction()

			case core.Object:
				c, ok := b.ToObject().(core.Closure)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
				}
				closure = c

			default:
				return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
			}

			last := core.NullValue

			for i, v := range a {
				var r core.Value
				if funcIndex != -1 {
					r, err = vm.RunFuncIndex(funcIndex, v)
				} else {
					r, err = vm.RunClosure(closure, v)
				}
				if err != nil {
					return core.NullValue, err
				}

				var matches bool

				switch r.Type {
				case core.Int:
					matches = r.ToInt() != 0

				case core.Float:
					matches = r.ToFloat() != 0

				case core.Bool:
					matches = r.ToBool()

				case core.Null, core.Undefined:
					matches = false

				default:
					matches = true
				}

				if matches {
					last = a[i]
				}
			}

			return last, nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.where",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}

			funcIndex := -1
			var closure core.Closure

			b := args[0]
			switch b.Type {
			case core.Func:
				funcIndex = b.ToFunction()

			case core.Object:
				c, ok := b.ToObject().(core.Closure)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
				}
				closure = c

			default:
				return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
			}

			filtered := make([]core.Value, 0)

			for _, v := range a {
				var r core.Value
				if funcIndex != -1 {
					r, err = vm.RunFuncIndex(funcIndex, v)
				} else {
					r, err = vm.RunClosure(closure, v)
				}
				if err != nil {
					return core.NullValue, err
				}

				var matches bool

				switch r.Type {
				case core.Int:
					matches = r.ToInt() != 0

				case core.Float:
					matches = r.ToFloat() != 0

				case core.Bool:
					matches = r.ToBool()

				case core.Null, core.Undefined:
					matches = false

				default:
					matches = true
				}

				if matches {
					filtered = append(filtered, v)
				}
			}

			return core.NewArrayValues(filtered), nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.sum",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}

			if len(args) == 0 {
				var ret float64
				var anyFloat bool

				for i, v := range a {
					switch v.Type {
					case core.Int:
						ret += v.ToFloat()

					case core.Float:
						ret += v.ToFloat()
						anyFloat = true

					default:
						return core.NullValue, fmt.Errorf("invalid array value at index %d", i)
					}
				}

				if anyFloat {
					return core.NewFloat(ret), nil
				}
				return core.NewInt(int(ret)), nil
			}

			funcIndex := -1
			var closure core.Closure

			b := args[0]
			switch b.Type {
			case core.Func:
				funcIndex = b.ToFunction()

			case core.Object:
				c, ok := b.ToObject().(core.Closure)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
				}
				closure = c

			default:
				return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
			}

			var ret float64
			var anyFloat bool

			for i, v := range a {
				var r core.Value
				if funcIndex != -1 {
					r, err = vm.RunFuncIndex(funcIndex, v)
				} else {
					r, err = vm.RunClosure(closure, v)
				}
				if err != nil {
					return core.NullValue, err
				}

				switch r.Type {
				case core.Int:
					ret += r.ToFloat()

				case core.Float:
					ret += r.ToFloat()
					anyFloat = true

				default:
					return core.NullValue, fmt.Errorf("invalid array value at index %d", i)
				}
			}

			if anyFloat {
				return core.NewFloat(ret), nil
			}
			return core.NewInt(int(ret)), nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.min",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}

			if len(a) == 0 {
				return core.UndefinedValue, err
			}

			var min, tmp float64
			var anyFloat bool

			if len(args) == 0 {
				for i, v := range a {
					switch v.Type {
					case core.Int:
						tmp = v.ToFloat()

					case core.Float:
						tmp = v.ToFloat()
						anyFloat = true

					default:
						return core.NullValue, fmt.Errorf("invalid array value at index %d", i)
					}

					if i == 0 || tmp < min {
						min = tmp
					}
				}

				if anyFloat {
					return core.NewFloat(min), nil
				}
				return core.NewInt(int(min)), nil
			}

			funcIndex := -1
			var closure core.Closure

			b := args[0]
			switch b.Type {
			case core.Func:
				funcIndex = b.ToFunction()

			case core.Object:
				c, ok := b.ToObject().(core.Closure)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
				}
				closure = c

			default:
				return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
			}

			for i, v := range a {
				var r core.Value
				if funcIndex != -1 {
					r, err = vm.RunFuncIndex(funcIndex, v)
				} else {
					r, err = vm.RunClosure(closure, v)
				}
				if err != nil {
					return core.NullValue, err
				}

				switch r.Type {
				case core.Int:
					tmp = r.ToFloat()

				case core.Float:
					tmp = r.ToFloat()
					anyFloat = true

				default:
					return core.NullValue, fmt.Errorf("invalid array value at index %d", i)
				}

				if i == 0 || tmp < min {
					min = tmp
				}
			}

			if anyFloat {
				return core.NewFloat(min), nil
			}
			return core.NewInt(int(min)), nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.max",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}

			if len(a) == 0 {
				return core.UndefinedValue, err
			}

			var max, tmp float64
			var anyFloat bool

			if len(args) == 0 {
				for i, v := range a {
					switch v.Type {
					case core.Int:
						tmp = v.ToFloat()

					case core.Float:
						tmp = v.ToFloat()
						anyFloat = true

					default:
						return core.NullValue, fmt.Errorf("invalid array value at index %d", i)
					}

					if i == 0 || tmp > max {
						max = tmp
					}
				}

				if anyFloat {
					return core.NewFloat(max), nil
				}
				return core.NewInt(int(max)), nil
			}

			funcIndex := -1
			var closure core.Closure

			b := args[0]
			switch b.Type {
			case core.Func:
				funcIndex = b.ToFunction()

			case core.Object:
				c, ok := b.ToObject().(core.Closure)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
				}
				closure = c

			default:
				return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
			}

			for i, v := range a {
				var r core.Value
				if funcIndex != -1 {
					r, err = vm.RunFuncIndex(funcIndex, v)
				} else {
					r, err = vm.RunClosure(closure, v)
				}
				if err != nil {
					return core.NullValue, err
				}

				switch r.Type {
				case core.Int:
					tmp = r.ToFloat()

				case core.Float:
					tmp = r.ToFloat()
					anyFloat = true

				default:
					return core.NullValue, fmt.Errorf("invalid array value at index %d", i)
				}

				if i == 0 || tmp > max {
					max = tmp
				}
			}

			if anyFloat {
				return core.NewFloat(max), nil
			}
			return core.NewInt(int(max)), nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.count",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}

			funcIndex := -1
			var closure core.Closure

			b := args[0]
			switch b.Type {
			case core.Func:
				funcIndex = b.ToFunction()

			case core.Object:
				c, ok := b.ToObject().(core.Closure)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
				}
				closure = c

			default:
				return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
			}

			var c int

			for _, v := range a {
				var r core.Value
				if funcIndex != -1 {
					r, err = vm.RunFuncIndex(funcIndex, v)
				} else {
					r, err = vm.RunClosure(closure, v)
				}
				if err != nil {
					return core.NullValue, err
				}

				var matches bool

				switch r.Type {
				case core.Int:
					matches = r.ToInt() != 0

				case core.Float:
					matches = r.ToFloat() != 0

				case core.Bool:
					matches = r.ToBool()

				case core.Null, core.Undefined:
					matches = false

				default:
					matches = true
				}

				if matches {
					c++
				}
			}

			return core.NewInt(c), nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.select",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}

			l := len(a)

			funcIndex := -1
			var closure core.Closure

			b := args[0]
			switch b.Type {
			case core.Func:
				funcIndex = b.ToFunction()

			case core.Object:
				c, ok := b.ToObject().(core.Closure)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
				}
				closure = c

			default:
				return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
			}

			items := make([]core.Value, l)

			for i, v := range a {
				var r core.Value
				if funcIndex != -1 {
					r, err = vm.RunFuncIndex(funcIndex, v)
				} else {
					r, err = vm.RunClosure(closure, v)
				}
				if err != nil {
					return core.NullValue, err
				}

				items[i] = r
			}

			return core.NewArrayValues(items), nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.selectMany",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}

			funcIndex := -1
			var closure core.Closure

			b := args[0]
			switch b.Type {
			case core.Func:
				funcIndex = b.ToFunction()

			case core.Object:
				c, ok := b.ToObject().(core.Closure)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
				}
				closure = c

			default:
				return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
			}

			items := make([]core.Value, 0)

			for i, v := range a {
				var r core.Value
				if funcIndex != -1 {
					r, err = vm.RunFuncIndex(funcIndex, v)
				} else {
					r, err = vm.RunClosure(closure, v)
				}
				if err != nil {
					return core.NullValue, err
				}
				if r.Type != core.Array {
					return core.NullValue, fmt.Errorf("the element in index %d is not an array", i)
				}
				items = append(items, r.ToArray()...)
			}

			return core.NewArrayValues(items), nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.distinct",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			thisItems, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}

			items := make([]core.Value, 0)

			if len(args) == 0 {
				for _, v := range thisItems {
					exists := false
					for _, w := range items {
						if v == w {
							exists = true
							break
						}
					}
					if !exists {
						items = append(items, v)
					}
				}
				return core.NewArrayValues(items), nil
			}

			funcIndex := -1
			var closure core.Closure

			b := args[0]
			switch b.Type {
			case core.Func:
				funcIndex = b.ToFunction()

			case core.Object:
				c, ok := b.ToObject().(core.Closure)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
				}
				closure = c

			default:
				return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
			}

			for _, v := range thisItems {
				var vKey core.Value
				if funcIndex != -1 {
					vKey, err = vm.RunFuncIndex(funcIndex, v)
				} else {
					vKey, err = vm.RunClosure(closure, v)
				}
				if err != nil {
					return core.NullValue, err
				}

				exists := false
				for _, w := range items {
					var existingKey core.Value
					if funcIndex != -1 {
						existingKey, err = vm.RunFuncIndex(funcIndex, w)
					} else {
						existingKey, err = vm.RunClosure(closure, w)
					}
					if err != nil {
						return core.NullValue, err
					}

					if existingKey == vKey {
						exists = true
						break
					}
				}

				if !exists {
					items = append(items, v)
				}
			}

			return core.NewArrayValues(items), nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.groupBy",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}
			groups := make(map[string]core.Value)

			funcIndex := -1
			var closure core.Closure

			b := args[0]
			switch b.Type {
			case core.Func:
				funcIndex = b.ToFunction()

			case core.Object:
				c, ok := b.ToObject().(core.Closure)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
				}
				closure = c

			default:
				return core.NullValue, fmt.Errorf("expected a function, got %s", b.TypeName())
			}

			for _, v := range a {
				var r core.Value
				if funcIndex != -1 {
					r, err = vm.RunFuncIndex(funcIndex, v)
				} else {
					r, err = vm.RunClosure(closure, v)
				}
				if err != nil {
					return core.NullValue, err
				}

				var key string
				if r.Type == core.Null {
					key = ""
				} else {
					key = r.ToString()
				}

				tmp, ok := groups[key]

				if ok {
					g := tmp.ToArrayObject().Array
					g = append(g, v)
					groups[key] = core.NewArrayValues(g)
				} else {
					g := []core.Value{v}
					groups[key] = core.NewArrayValues(g)
				}
			}

			return core.NewMapValues(groups), nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.equals",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if len(args) == 0 || args[0].Type != core.Array {
				return core.FalseValue, nil
			}

			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}
			b := args[0].ToArray()

			if len(a) != len(b) {
				return core.FalseValue, nil
			}

			for i := range a {
				if !a[i].Equals(b[i]) {
					return core.FalseValue, nil
				}
			}

			return core.TrueValue, nil
		},
	},

	core.NativeFunction{
		Name:      "Array.prototype.sort",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			switch this.Type {
			case core.Null:
				return args[0], nil
			case core.Array, core.Bytes:
			default:
				return core.NullValue, fmt.Errorf("expected array, called on %s", this.TypeName())
			}

			a := this.ToArray()

			b := args[0]
			switch b.Type {
			case core.Func:
				c := &comparer{items: a, compFunc: b.ToFunction(), vm: vm}
				sort.Sort(c)
				return core.NullValue, nil

			case core.Object:
				cl, ok := b.ToObjectOrNil().(core.Closure)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a function, got: %s", b.TypeName())
				}
				c := &closureComparer{items: a, comp: cl, vm: vm}
				sort.Sort(c)
				return core.NullValue, nil

			default:
				return core.NullValue, fmt.Errorf("expected a function, got: %s", b.TypeName())
			}
		},
	},
	core.NativeFunction{
		Name:      "Array.prototype.append",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			switch this.Type {
			case core.Array, core.Bytes:
			default:
				return core.NullValue, fmt.Errorf("expected array, called on %s", this.TypeName())
			}

			a := this.ToArray()

			b := args[0]
			switch b.Type {
			case core.Null:
				return this, nil
			case core.Array, core.Bytes:
			default:
				return core.NullValue, fmt.Errorf("expected array, called on %s", b.TypeName())
			}

			c := append(a, b.ToArray()...)

			return core.NewArrayValues(c), nil
		},
	},
	core.NativeFunction{
		Name:      "Array.prototype.pushRange",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			switch this.Type {
			case core.Array, core.Bytes:
			default:
				return core.NullValue, fmt.Errorf("expected array, called on %s", this.TypeName())
			}

			b := args[0]
			switch b.Type {
			case core.Null:
			case core.Array, core.Bytes:
				a := this.ToArrayObject()
				items := b.ToArray()
				a.Array = append(a.Array, items...)

				if vm.MaxAllocations > 0 {
					var allocs int
					for _, v := range items {
						allocs += v.Size()
					}
					if err := vm.AddAllocations(allocs); err != nil {
						return core.NullValue, err
					}
				}

			default:
				return core.NullValue, fmt.Errorf("expected array, called on %s", b.TypeName())
			}

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "Array.prototype.push",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			switch this.Type {
			case core.Array:
				a := this.ToArrayObject()
				a.Array = append(a.Array, args...)
				if vm.MaxAllocations > 0 {
					var allocs int
					for _, v := range a.Array {
						allocs += v.Size()
					}
					if err := vm.AddAllocations(allocs); err != nil {
						return core.NullValue, err
					}
				}

			default:
				return core.NullValue, fmt.Errorf("expected array, got %s", this.TypeName())
			}

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "Array.prototype.insertAt",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if this.Type != core.Array {
				return core.NullValue, fmt.Errorf("expected string array, got %s", this.TypeName())
			}
			if args[0].Type != core.Int {
				return core.NullValue, fmt.Errorf("expected arg 1 to be int, got %s", this.TypeName())
			}
			obj := this.ToArrayObject()
			i := int(args[0].ToInt())

			a := obj.Array
			a = append(a, core.NullValue)
			copy(a[i+1:], a[i:])
			a[i] = args[1]
			obj.Array = a

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "Array.prototype.removeAt",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.Int {
				return core.NullValue, fmt.Errorf("expected arg 1 to be int, got %s", args[0].TypeName())
			}

			if this.Type != core.Array {
				return core.NullValue, fmt.Errorf("expected string array, got %s", this.TypeName())
			}
			obj := this.ToArrayObject()
			i := int(args[0].ToInt())

			a := obj.Array
			copy(a[i:], a[i+1:])
			a[len(a)-1] = core.NullValue
			obj.Array = a[:len(a)-1]

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "Array.prototype.removeRange",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if this.Type != core.Array {
				return core.NullValue, fmt.Errorf("expected string array, got %s", this.TypeName())
			}
			obj := this.ToArrayObject()
			i := int(args[0].ToInt())
			j := int(args[0].ToInt())

			a := obj.Array
			copy(a[i:], a[j:])
			for k, n := len(a)-j+i, len(a); k < n; k++ {
				a[k] = core.NullValue
			}
			obj.Array = a[:len(a)-j+i]

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "Array.prototype.indexOf",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if this.Type != core.Array {
				return core.NullValue, fmt.Errorf("expected string array, got %s", this.TypeName())
			}
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}
			v := args[0]

			for i, j := range a {
				if j.Equals(v) {
					return core.NewInt(i), nil
				}
			}

			return core.NewInt(-1), nil
		},
	},
	core.NativeFunction{
		Name: "Array.prototype.reverse",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if this.Type != core.Array {
				return core.NullValue, fmt.Errorf("expected string array, got %s", this.TypeName())
			}
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}
			l := len(a) - 1

			for i, k := 0, l/2; i <= k; i++ {
				a[i], a[l-i] = a[l-i], a[i]
			}

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "Array.prototype.join",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if this.Type != core.Array {
				return core.NullValue, fmt.Errorf("expected string array, got %s", this.TypeName())
			}
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}

			sep := args[0]
			if sep.Type != core.String {
				return core.NullValue, fmt.Errorf("expected string, got %s", sep.TypeName())
			}

			s := make([]string, len(a))
			for i, v := range a {
				switch v.Type {
				case core.String, core.Rune, core.Int, core.Float,
					core.Bool, core.Null, core.Undefined:
					s[i] = v.ToString()

				default:
					return core.NullValue, fmt.Errorf("invalid type at index %d, expected string, got %s", i, v.TypeName())
				}
			}
			r := strings.Join(s, sep.ToString())

			return core.NewString(r), nil
		},
	},
	core.NativeFunction{
		Name:      "Array.prototype.slice",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if this.Type != core.Array {
				return core.NullValue, fmt.Errorf("expected array, called on %s", this.TypeName())
			}
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}
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

			return core.NewArrayValues(a), nil
		},
	},
	core.NativeFunction{
		Name:      "Array.prototype.range",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if this.Type != core.Array {
				return core.NullValue, fmt.Errorf("expected array, called on %s", this.TypeName())
			}
			a, err := toArray(this)
			if err != nil {
				return core.NullValue, err
			}
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

			return core.NewArrayValues(a), nil
		},
	},
}

func toArray(v core.Value) ([]core.Value, error) {
	switch v.Type {
	case core.Array:
		return v.ToArray(), nil

	case core.Object:
		e, ok := v.ToObject().(core.Enumerable)
		if !ok {
			return nil, fmt.Errorf("expected an enumerable, got %s", v.TypeName())
		}

		a, err := e.Values()
		if err != nil {
			return nil, fmt.Errorf("error enumerating values: %v", err)
		}

		return a, nil

	default:
		return nil, fmt.Errorf("expected an enumerable, got %s", v.TypeName())
	}
}

type comparer struct {
	items    []core.Value
	compFunc int
	vm       *core.VM
	err      error
}

func (c *comparer) Len() int {
	return len(c.items)
}
func (c *comparer) Swap(i, j int) {
	c.items[i], c.items[j] = c.items[j], c.items[i]
}
func (c *comparer) Less(i, j int) bool {
	if c.err != nil {
		return false
	}
	v, err := c.vm.RunFuncIndex(c.compFunc, c.items[i], c.items[j])
	if err != nil {
		c.err = err
		return false
	}

	if v.Type != core.Bool {
		c.err = fmt.Errorf("the comparer function must return a boolean")
	}
	return v.ToBool()
}

type closureComparer struct {
	items []core.Value
	comp  core.Closure
	vm    *core.VM
	err   error
}

func (c *closureComparer) Len() int {
	return len(c.items)
}
func (c *closureComparer) Swap(i, j int) {
	c.items[i], c.items[j] = c.items[j], c.items[i]
}
func (c *closureComparer) Less(i, j int) bool {
	if c.err != nil {
		return false
	}
	v, err := c.vm.RunClosure(c.comp, c.items[i], c.items[j])
	if err != nil {
		c.err = err
		return false
	}

	if v.Type != core.Bool {
		c.err = fmt.Errorf("the comparer function must return a boolean")
	}
	return v.ToBool()
}
