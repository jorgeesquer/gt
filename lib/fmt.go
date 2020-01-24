package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(libFmt, `
declare namespace fmt {
    export function print(...n: any[]): void
    export function println(...n: any[]): void
    export function printJSON(v: any): void
    export function printf(format: string, ...params: any[]): void
    export function sprintf(format: string, ...params: any[]): string
    export function fprintf(w: io.Writer, format: string, ...params: any[]): void
}	
	`)
}

var libFmt = []core.NativeFunction{
	core.NativeFunction{
		Name:      "fmt.print",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			for _, v := range args {
				switch v.Type {
				case core.Undefined:
					fmt.Print("undefined")
				case core.Float:
					fmt.Print(v.String())
				default:
					o := v.Export(0)
					if o == nil {
						fmt.Print("<null>")
					} else if s, ok := o.(fmt.Stringer); ok {
						fmt.Print(s.String())
					} else {
						fmt.Print(o)
					}
				}
			}
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "fmt.println",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			for i, v := range args {
				if i > 0 {
					fmt.Print(" ")
				}

				switch v.Type {
				case core.Undefined:
					fmt.Print("undefined")
				case core.Float:
					fmt.Print(v.String())
				default:
					o := v.Export(0)
					if o == nil {
						fmt.Print("<null>")
					} else if s, ok := o.(fmt.Stringer); ok {
						fmt.Print(s.String())
					} else {
						fmt.Print(o)
					}
				}
			}
			fmt.Print("\n")
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "fmt.printf",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			l := len(args)
			if l == 0 {
				return core.NullValue, fmt.Errorf("expected at least 1 parameter, got %d", len(args))
			}
			v := args[0]
			if v.Type != core.String {
				return core.NullValue, fmt.Errorf("expected parameter 1 to be a string, got %s", v.Type)
			}

			values := make([]interface{}, l-1)
			for i, v := range args[1:] {
				switch v.Type {
				case core.Null:
					values[i] = "<null>"
				case core.String:
					// need to escape the % to prevent interfering with fmt
					values[i] = core.NewString(strings.Replace(v.ToString(), "%", "%%", -1))
				default:
					o := v.Export(0)
					if o == nil {
						values[i] = "<null>"
					} else {
						values[i] = o
					}
				}
			}

			fmt.Printf(v.ToString(), values...)
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "fmt.fprintf",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			l := len(args)
			if l < 2 {
				return core.NullValue, fmt.Errorf("expected at least 2 parameters, got %d", len(args))
			}

			w, ok := args[0].ToObjectOrNil().(io.Writer)
			if !ok {
				return core.NullValue, fmt.Errorf("expected parameter 1 to be a io.Writer, got %s", args[0].TypeName())
			}

			v := args[1]
			if v.Type != core.String {
				return core.NullValue, fmt.Errorf("expected parameter 2 to be a string, got %s", v.TypeName())
			}

			values := make([]interface{}, l-2)
			for i, v := range args[2:] {
				switch v.Type {
				case core.Null:
					values[i] = "<null>"
				case core.String:
					// need to escape the % to prevent interfering with fmt
					values[i] = core.NewString(strings.Replace(v.ToString(), "%", "%%", -1))
				default:
					o := v.Export(0)
					if o == nil {
						values[i] = "<null>"
					} else {
						values[i] = o
					}
				}
			}

			fmt.Fprintf(w, v.ToString(), values...)
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "fmt.printJSON",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			if a.Type == core.Undefined {
				fmt.Print("undefined")
			} else {
				v := a.Export(0)
				b, err := json.MarshalIndent(v, "", "    ")
				if err != nil {
					return core.NullValue, err
				}
				fmt.Println(string(b))
			}
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "fmt.sprintf",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			l := len(args)
			if l == 0 {
				return core.NullValue, fmt.Errorf("expected at least 1 parameter, got %d", len(args))
			}
			v := args[0]
			if v.Type != core.String {
				return core.NullValue, fmt.Errorf("expected parameter 1 to be a string, got %s", v.Type)
			}

			values := make([]interface{}, l-1)
			for i, v := range args[1:] {
				switch v.Type {
				case core.Null:
					values[i] = "<null>"
				default:
					o := v.Export(0)
					if o == nil {
						values[i] = "<null>"
					} else {
						values[i] = o
					}
				}
			}

			s := fmt.Sprintf(v.ToString(), values...)
			return core.NewString(s), nil
		},
	},
}
