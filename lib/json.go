package lib

import (
	"encoding/json"
	"fmt"
	"github.com/gtlang/gt/core"
	"strings"
)

func init() {
	core.RegisterLib(JSON, `

declare namespace json {
    export function escapeString(str: string): string
    export function marshal(v: any, indent?: boolean): string
    export function unmarshal(str: string | byte[]): any

}
`)
}

var JSON = []core.NativeFunction{
	core.NativeFunction{
		Name:      "json.marshal",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			var format bool

			switch len(args) {
			case 1:

			case 2:
				b := args[1]
				if b.Type != core.Bool {
					return core.NullValue, fmt.Errorf("expected arg 2 to be boolean, got %s", b.TypeName())
				}
				format = b.ToBool()

			default:
				return core.NullValue, fmt.Errorf("expected 1 or 2 arguments, got %d", len(args))
			}

			v := args[0].Export(0)

			var b []byte
			var err error

			if format {
				b, err = json.MarshalIndent(v, "", "    ")
			} else {
				b, err = json.Marshal(v)
			}

			if err != nil {
				return core.NullValue, err
			}

			return core.NewString(string(b)), nil
		},
	},
	core.NativeFunction{
		Name:      "json.unmarshal",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if len(args) != 1 {
				return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
			}

			a := args[0]

			switch a.Type {
			case core.String, core.Bytes:
			default:
				return core.NullValue, fmt.Errorf("expected argument to be string or byte[], got %d", args[0].Type)
			}

			if a.ToString() == "" {
				return core.NullValue, nil
			}

			v, err := unmarshal(a.ToBytes())
			if err != nil {
				return core.NullValue, err
			}

			return v, nil
		},
	},
	core.NativeFunction{
		Name:      "json.escapeString",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}
			s := args[0].ToString()
			r := strings.NewReplacer("\\", "\\\\", "\n", "\\n", "\r", "", "\"", "\\\"", "'", "\\'")
			s = r.Replace(s)
			return core.NewString(s), nil
		},
	},
}

func unmarshal(buf []byte) (core.Value, error) {
	var o interface{}
	err := json.Unmarshal(buf, &o)
	if err != nil {
		return core.NullValue, err
	}

	return unmarshalObject(o)
}

func unmarshalObject(value interface{}) (core.Value, error) {
	switch t := value.(type) {
	case nil:
		return core.NullValue, nil
	case float32: // is this possible?
		i := int(t)
		if t == float32(i) {
			return core.NewInt(i), nil
		}
		return core.NewFloat(float64(t)), nil
	case float64:
		i := int(t)
		if t == float64(i) {
			return core.NewInt(i), nil
		}
		return core.NewFloat(t), nil
	case int, int32, int64, bool, string:
		return core.NewValue(t), nil
	case []interface{}:
		s := make([]core.Value, len(t))
		for i, v := range t {
			o, err := unmarshalObject(v)
			if err != nil {
				return core.NullValue, err
			}
			s[i] = o
		}
		return core.NewArrayValues(s), nil
	case map[string]interface{}:
		m := make(map[string]core.Value, len(t))
		for k, v := range t {
			o, err := unmarshalObject(v)
			if err != nil {
				return core.NullValue, err
			}
			m[k] = o
		}
		return core.NewMapValues(m), nil

	default:
		return core.NullValue, fmt.Errorf("invalid serialized type %T", value)
	}

}
