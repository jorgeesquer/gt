package lib

import (
	"fmt"
	"time"

	"github.com/gtlang/gt/core"

	cache "github.com/gtlang/gt/lib/x/go-cache"
)

func init() {
	core.RegisterLib(Caching, `
	
declare namespace caching {
 
    export let global: Cache

    export function newCache(d?: time.Duration | number): Cache

    export interface Cache {
        get(key: string): any | null
        save(key: string, v: any): void
        delete(key: string): void
        keys(): string[]
        items(): Map<any>
        clear(): void
    }
}

`)
}

var global *cacheObj

var Caching = []core.NativeFunction{
	core.NativeFunction{
		Name: "->caching.global",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}
			if global == nil {
				global = newCacheObj(1 * time.Minute)
			}
			return core.NewObject(global), nil
		},
	},
	core.NativeFunction{
		Name:      "caching.newCache",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			l := len(args)

			var d time.Duration

			switch l {
			case 0:
				d = 1 * time.Minute
			case 1:
				var a = args[0]
				switch a.Type {
				case core.Int:
					d = time.Duration(a.ToInt() * 1000000)
				case core.Object:
					dur, ok := a.ToObject().(Duration)
					if !ok {
						return core.NullValue, fmt.Errorf("expected duration, got %s", a.TypeName())
					}
					d = time.Duration(dur)
				}
			default:
				return core.NullValue, fmt.Errorf("expected 0 or 1 arguments, got %d", l)
			}

			return core.NewObject(newCacheObj(d)), nil
		},
	},
}

func newCacheObj(d time.Duration) *cacheObj {
	return &cacheObj{
		cache: cache.New(d, 30*time.Second),
	}
}

type cacheObj struct {
	cache *cache.Cache
}

func (*cacheObj) Type() string {
	return "caching.Cache"
}

func (c *cacheObj) GetMethod(name string) core.NativeMethod {
	switch name {
	case "get":
		return c.get
	case "save":
		return c.save
	case "delete":
		return c.delete
	case "clear":
		return c.clear
	case "keys":
		return c.keys
	case "items":
		return c.items
	}
	return nil
}

func (c *cacheObj) keys(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	keys := c.cache.Keys()

	m := make([]core.Value, len(keys))

	for i, k := range keys {
		m[i] = core.NewString(k)
	}

	return core.NewArrayValues(m), nil
}

func (c *cacheObj) items(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	items := c.cache.Items()

	m := make(map[string]core.Value, len(items))

	for k, v := range items {
		m[k] = v.Object.(core.Value)
	}

	return core.NewMapValues(m), nil
}

func (c *cacheObj) get(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}
	key := args[0].ToString()
	if i, ok := c.cache.Get(key); ok {
		return i.(core.Value), nil
	}
	return core.NullValue, nil
}

func (c *cacheObj) save(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 2 {
		return core.NullValue, fmt.Errorf("expected 2 args, got %d", len(args))
	}
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("invalid argument type, expected string, got %s", args[0].TypeName())
	}
	key := args[0].ToString()
	v := args[1]
	c.cache.Set(key, v, cache.DefaultExpiration)
	return core.NullValue, nil
}

func (c *cacheObj) delete(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}
	key := args[0].ToString()
	c.cache.Delete(key)
	return core.NullValue, nil
}

func (c *cacheObj) clear(args []core.Value, vm *core.VM) (core.Value, error) {
	c.cache.Flush()
	return core.NullValue, nil
}
