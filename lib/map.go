package lib

import (
	"fmt"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(libMap, `	
declare interface StringMap {
    [key: string]: string
}

declare interface KeyIndexer<T> {
    [key: string]: T
}

declare type Map<T> = KeyIndexer<T>
 
declare namespace map {
    export function len(v: any): number
    export function keys(v: any): string[]
    export function values<T>(v: Map<T>): T[]
    export function values<T>(v: any): T[]
    export function deleteKey(v: any, key: string | number): void
    export function deleteKeys(v: any): void
    export function hasKey(v: any, key: any): boolean
    export function clone<T>(v: T): T
}
	`)
}

var libMap = []core.NativeFunction{
	core.NativeFunction{
		Name:      "map.isMap",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0].Type == core.Map
			return core.NewBool(a), nil
		},
	},
	core.NativeFunction{
		Name:      "map.clone",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			if a.Type != core.Map {
				return core.NullValue, fmt.Errorf("expected a map or object, got %s", a.TypeName())
			}

			m := a.ToMap().Map

			clone := make(map[string]core.Value, len(m))
			for k, v := range m {
				clone[k] = v
			}

			return core.NewMapValues(clone), nil
		},
	},
	core.NativeFunction{
		Name:      "map.len",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			if a.Type != core.Map {
				return core.NullValue, fmt.Errorf("expected a map or object, got %s", a.TypeName())
			}
			m := a.ToMap()
			m.Mutex.RLock()
			l := len(m.Map)
			m.Mutex.RUnlock()
			return core.NewInt(l), nil
		},
	},
	core.NativeFunction{
		Name:      "map.keys",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]

			switch a.Type {
			case core.Map:
			default:
				return core.NullValue, fmt.Errorf("expected a map or object, got %s", a.TypeName())
			}

			m := a.ToMap()
			m.Mutex.RLock()
			keys := make([]core.Value, len(m.Map))
			var i int
			for k := range m.Map {
				keys[i] = core.NewString(k)
				i++
			}
			m.Mutex.RUnlock()
			return core.NewArrayValues(keys), nil
		},
	},
	core.NativeFunction{
		Name:      "map.values",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			if a.Type != core.Map {
				return core.NullValue, fmt.Errorf("expected a map or object, got %s", a.TypeName())
			}

			m := a.ToMap()
			m.Mutex.RLock()
			values := make([]core.Value, len(m.Map))
			var i int
			for k := range m.Map {
				values[i] = m.Map[k]
				i++
			}
			m.Mutex.RUnlock()
			return core.NewArrayValues(values), nil
		},
	},
	core.NativeFunction{
		Name:      "map.deleteKey",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			if a.Type != core.Map {
				return core.NullValue, fmt.Errorf("expected a map or object, got %s", a.TypeName())
			}

			b := args[1]
			switch b.Type {
			case core.String, core.Int:
			default:
				return core.NullValue, fmt.Errorf("invalid key type: %s", b.TypeName())
			}

			m := a.ToMap()
			m.Mutex.Lock()
			delete(m.Map, b.ToString())
			m.Mutex.Unlock()
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "map.deleteKeys",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			if a.Type != core.Map {
				return core.NullValue, fmt.Errorf("expected a map or object, got %s", a.TypeName())
			}

			m := a.ToMap()
			m.Mutex.Lock()
			for k:= range m.Map {
				delete(m.Map, k)
			}
			m.Mutex.Unlock()
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "map.hasKey",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			b := args[1]

			a := args[0]
			switch a.Type {
			case core.Map:
				m := a.ToMap()
				m.Mutex.RLock()
				_, ok := m.Map[b.ToString()]
				m.Mutex.RUnlock()
				return core.NewBool(ok), nil

			case core.Object:
				if o, ok := a.ToObject().(core.PropertyGetter); ok {
					v, err := o.GetProperty(b.ToString(), vm)
					if err != nil {
						return core.NullValue, err
					}
					return core.NewBool(v.Type != core.Undefined), nil
				}
			}

			return core.NullValue, fmt.Errorf("expected a map or object, got %s", a.TypeName())
		},
	},
}
