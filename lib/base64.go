package lib

import (
	"encoding/base64"
	"fmt"

	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(Base64, `
	
declare namespace base64 {
    export function encode(s: any): string
    export function encodeWithPadding(s: any): string
    export function decode(s: any): string
    export function decodeWithPadding(s: any): string
}

`)
}

var Base64 = []core.NativeFunction{
	core.NativeFunction{
		Name:      "base64.encode",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			switch a.Type {
			case core.Null, core.Undefined:
				return core.NullValue, nil
			case core.Bytes, core.String:
				encoder := base64.StdEncoding.WithPadding(base64.NoPadding)
				encoded := encoder.EncodeToString(a.ToBytes())
				return core.NewString(encoded), nil
			default:
				return core.NullValue, fmt.Errorf("expected string, got %v", a.Type)
			}
		},
	},
	core.NativeFunction{
		Name:      "base64.encodeWithPadding",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			switch a.Type {
			case core.Null, core.Undefined:
				return core.NullValue, nil
			case core.Bytes, core.String:
				encoder := base64.StdEncoding.WithPadding(base64.StdPadding)
				encoded := encoder.EncodeToString(a.ToBytes())
				return core.NewString(encoded), nil
			default:
				return core.NullValue, fmt.Errorf("expected string, got %v", a.Type)
			}
		},
	},
	core.NativeFunction{
		Name:      "base64.decode",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			switch a.Type {
			case core.Null, core.Undefined:
				return core.NullValue, nil
			case core.String:
				encoder := base64.StdEncoding.WithPadding(base64.NoPadding)
				encoded, err := encoder.DecodeString(a.ToString())
				if err != nil {
					return core.NullValue, err
				}
				return core.NewBytes(encoded), nil
			default:
				return core.NullValue, fmt.Errorf("expected string, got %v", a.Type)
			}
		},
	},
	core.NativeFunction{
		Name:      "base64.decodeWithPadding",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			switch a.Type {
			case core.Null, core.Undefined:
				return core.NullValue, nil
			case core.String:
				encoder := base64.StdEncoding.WithPadding(base64.StdPadding)
				encoded, err := encoder.DecodeString(a.ToString())
				if err != nil {
					return core.NullValue, err
				}
				return core.NewString(string(encoded)), nil
			default:
				return core.NullValue, fmt.Errorf("expected string, got %v", a.Type)
			}
		},
	},
}
