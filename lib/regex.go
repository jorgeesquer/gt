package lib

import (
	"regexp"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(Regex, `

declare namespace regex {
    export function match(pattern: string, value: string): boolean
    export function split(pattern: string, value: string): string[]
    export function findAllStringSubmatch(pattern: string, value: string, count?: number): string[]
    export function findAllStringSubmatchIndex(pattern: string, value: string, count?: number): number[][]
    export function replaceAllString(pattern: string, source: string, replace: string): string
}


`)
}

var Regex = []core.NativeFunction{
	core.NativeFunction{
		Name:      "regex.match",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			ok, err := regexp.MatchString(args[0].ToString(), args[1].ToString())
			if err != nil {
				return core.NullValue, err
			}
			return core.NewBool(ok), nil
		},
	},
	core.NativeFunction{
		Name:      "regex.split",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			r, err := regexp.Compile(args[0].ToString())
			if err != nil {
				return core.NullValue, err
			}

			matches := r.Split(args[1].ToString(), -1)

			ln := len(matches)
			result := make([]core.Value, ln)
			for i := 0; i < ln; i++ {
				result[i] = core.NewString(matches[i])
			}

			return core.NewArrayValues(result), nil
		},
	},
	core.NativeFunction{
		Name:      "regex.findAllStringSubmatchIndex",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgRange(args, 2, 3); err != nil {
				return core.NullValue, err
			}
			if err := ValidateOptionalArgs(args, core.String, core.String, core.Int); err != nil {
				return core.NullValue, err
			}

			r, err := regexp.Compile(args[0].ToString())
			if err != nil {
				return core.NullValue, err
			}

			var i int
			if len(args) == 3 {
				i = int(args[2].ToInt())
			} else {
				i = -1
			}

			matches := r.FindAllStringSubmatchIndex(args[1].ToString(), i)

			var result []core.Value
			for _, v := range matches {
				a := []core.Value{core.NewInt(v[0]), core.NewInt(v[1])}
				result = append(result, core.NewArrayValues(a))
			}

			return core.NewArrayValues(result), nil
		},
	},
	core.NativeFunction{
		Name:      "regex.findAllStringSubmatch",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgRange(args, 2, 3); err != nil {
				return core.NullValue, err
			}
			if err := ValidateOptionalArgs(args, core.String, core.String, core.Int); err != nil {
				return core.NullValue, err
			}

			r, err := regexp.Compile(args[0].ToString())
			if err != nil {
				return core.NullValue, err
			}

			var i int
			if len(args) == 3 {
				i = int(args[2].ToInt())
			} else {
				i = -1
			}

			matches := r.FindAllStringSubmatch(args[1].ToString(), i)

			var result []core.Value

			for _, v := range matches {
				for _, sv := range v[1:] {
					result = append(result, core.NewString(sv))
				}
			}

			return core.NewArrayValues(result), nil
		},
	},
	core.NativeFunction{
		Name:      "regex.replaceAllString",
		Arguments: 3,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			r, err := regexp.Compile(args[0].ToString())
			if err != nil {
				return core.NullValue, err
			}

			result := r.ReplaceAllString(args[1].ToString(), args[2].ToString())

			return core.NewString(result), nil
		},
	},
}
