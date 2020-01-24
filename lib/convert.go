package lib

import (
	"fmt"
	"github.com/gtlang/gt/lib/x/i18n"
	"github.com/gtlang/gt/core"
	"strconv"
	"strings"
)

func init() {
	core.RegisterLib(Convert, `

declare namespace convert {
    export function toInt(v: string | number): number
    export function toFloat(v: string | number): number
    export function parseCurrency(v: string | number): number
    export function toString(v: any): string
    export function toRune(v: any): string
    export function toBool(v: string | number | boolean): boolean
    export function toBytes(v: string | byte[]): byte[]
}

`)
}

var Convert = []core.NativeFunction{
	core.NativeFunction{
		Name:      "convert.toByte",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			var r core.Value

			switch a.Type {
			case core.String:
			default:
				return core.NullValue, fmt.Errorf("can't convert %v to byte", a.Type)
			}

			s := a.ToString()
			if len(s) != 1 {
				return core.NullValue, fmt.Errorf("can't convert %v to int", a.Type)
			}

			return r, nil
		},
	},
	core.NativeFunction{
		Name:      "convert.toRune",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]

			switch a.Type {
			case core.String:
				s := a.ToString()
				if len(s) != 1 {
					return core.NullValue, fmt.Errorf("can't convert %v to rune", s)
				}
				return core.NewRune(rune(s[0])), nil
			case core.Int:
				i := a.ToInt()
				if i > 255 {
					return core.NullValue, fmt.Errorf("can't convert %v to rune", i)
				}
				return core.NewRune(rune(i)), nil
			default:
				return core.NullValue, fmt.Errorf("can't convert %v to byte", a.Type)
			}
		},
	},
	core.NativeFunction{
		Name:      "convert.toInt",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			var r core.Value

			switch a.Type {
			case core.Int:
				r = a
			case core.Float:
				r = core.NewInt64(a.ToInt())
			case core.Rune:
				r = core.NewInt64(a.ToInt())
			case core.String:
				s, err := trimZeros(a.ToString())
				if err != nil {
					return core.NullValue, err
				}
				i, err := strconv.ParseInt(s, 0, 64)
				if err != nil {
					return core.NullValue, err
				}
				r = core.NewInt64(i)
			default:
				return core.NullValue, fmt.Errorf("can't convert %v to int", a.Type)
			}

			return r, nil
		},
	},
	core.NativeFunction{
		Name:      "convert.toFloat",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			switch a.Type {
			case core.Int:
				return core.NewFloat(a.ToFloat()), nil
			case core.Float:
				return a, nil
			case core.String:
				c := GetContext(vm).GetCulture()
				s, err := trimZeros(a.ToString())
				if err != nil {
					return core.NullValue, err
				}

				i, err := parseFloat(s, c.culture)
				if err != nil {
					return core.NullValue, err
				}
				return core.NewFloat(i), nil
			default:
				return core.NullValue, fmt.Errorf("can't convert %v to int", a.Type)
			}
		},
	},
	core.NativeFunction{
		Name:      "convert.parseCurrency",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			switch a.Type {
			case core.Int:
				return core.NewFloat(a.ToFloat()), nil
			case core.Float:
				return a, nil
			case core.String:
				c := GetContext(vm).GetCulture()
				s, err := trimZeros(a.ToString())
				if err != nil {
					return core.NullValue, err
				}
				s = strings.Replace(s, c.culture.CurrencySymbol, "", 1)

				i, err := parseFloat(s, c.culture)
				if err != nil {
					return core.NullValue, err
				}
				return core.NewFloat(i), nil
			default:
				return core.NullValue, fmt.Errorf("can't convert %v to int", a.Type)
			}
		},
	},
	core.NativeFunction{
		Name:      "convert.toBool",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			var r core.Value

			switch a.Type {

			case core.Bool:
				r = a

			case core.Int:
				switch a.ToInt() {
				case 0:
					r = core.FalseValue
				case 1:
					r = core.TrueValue
				default:
					return core.NullValue, fmt.Errorf("can't convert %v to bool", a.Type)
				}

			case core.String:
				s := a.ToString()
				s = strings.Trim(s, " ")
				switch s {
				case "true", "1":
					r = core.TrueValue
				case "false", "0":
					r = core.FalseValue
				default:
					return core.NullValue, fmt.Errorf("can't convert %v to bool", s)
				}

			default:
				return core.NullValue, fmt.Errorf("can't convert %v to bool", a.Type)

			}

			return r, nil
		},
	},
	core.NativeFunction{
		Name:      "convert.toString",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			var r core.Value
			switch a.Type {
			case core.Int, core.Float, core.Bool, core.Bytes, core.Rune:
				r = core.NewString(a.ToString())
			case core.String:
				r = a
			case core.Object:
				o := a.ToObject()
				switch t := o.(type) {
				case TimeObj:
					r = core.NewString(t.String())
				default:
					r = core.NewString(a.TypeName())
				}
			default:
				r = core.NewString(a.TypeName())
			}

			return r, nil
		},
	},
	core.NativeFunction{
		Name:      "convert.toBytes",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			var r core.Value

			switch a.Type {
			case core.String:
				r = core.NewBytes(a.ToBytes())
			case core.Bytes:
				r = a
			default:
				return core.NullValue, fmt.Errorf("can't convert %v to int", a.Type)
			}

			return r, nil
		},
	},
}

func parseFloat(s string, c i18n.Culture) (float64, error) {
	if s == "NaN" {
		return 0, fmt.Errorf("invalid format: NaN")
	}

	// if the string only contains a . accept it always as the decimal separator
	// for any culture.
	if c.DecimalSeparator == '.' || !strings.Contains(s, string(c.DecimalSeparator)) {
		if strings.Count(s, ".") == 1 {
			return strconv.ParseFloat(s, 64)
		}
	}

	// if strings.ContainsRune(s, c.ThousandSeparator) {
	// 	return 0, fmt.Errorf("invalid format (thousands separator): %v", s)
	// }

	if strings.ContainsRune(s, c.ThousandSeparator) {
		s = strings.Replace(s, string(c.ThousandSeparator), "", -1)
	}

	if c.DecimalSeparator != '.' {
		s = strings.Replace(s, string(c.DecimalSeparator), ".", -1)
	}

	return strconv.ParseFloat(s, 64)
}

func trimZeros(s string) (string, error) {
	s = strings.Trim(s, " ")

	if len(s) == 0 {
		return "", fmt.Errorf("can't parse number: emtpy string")
	}

	s = strings.TrimLeft(s, "0")
	if s == "" {
		s = "0"
	}
	return s, nil

}
