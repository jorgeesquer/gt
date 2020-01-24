package lib

import (
	"fmt"
	"html"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(HTML, `

declare namespace html {
    export function encode(s: any): string
    export function decode(s: any): string
}


`)
}

var HTML = []core.NativeFunction{
	core.NativeFunction{
		Name:      "html.encode",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			switch a.Type {
			case core.Null, core.Undefined:
				return core.NullValue, nil
			case core.String:
				return core.NewString(html.EscapeString(a.ToString())), nil
			default:
				return core.NewString(a.String()), nil
			}
		},
	},
	core.NativeFunction{
		Name:      "html.decode",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			switch a.Type {
			case core.Null, core.Undefined:
				return core.NullValue, nil
			case core.String:
				return core.NewString(html.UnescapeString(a.ToString())), nil
			default:
				return core.NullValue, fmt.Errorf("expected string, got %v", a.Type)
			}
		},
	},
}
