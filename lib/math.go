package lib

import (
	"fmt"
	"math"
	"math/rand"
	"github.com/gtlang/gt/core"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	core.RegisterLib(Math, `

declare namespace math {
    /**
     * returns, as an int, a non-negative pseudo-random number in (0,n)
     */
    export function rand(n: number): number

    export function abs(n: number): number

    export function min(nums: number[]): number

    export function floor(n: number): number
    export function ceil(n: number): number

    export function round(n: number, decimals?: number): number

    export function median(nums: number[]): number

    export function standardDev(nums: number[]): number
}

`)
}

var Math = []core.NativeFunction{
	core.NativeFunction{
		Name:      "math.pow",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Float, core.Float); err != nil {
				return core.NullValue, err
			}
			v := math.Pow(args[0].ToFloat(), args[1].ToFloat())
			return core.NewFloat(v), nil
		},
	},
	core.NativeFunction{
		Name:      "math.abs",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Int); err != nil {
				return core.NullValue, err
			}
			v := math.Abs(args[0].ToFloat())
			return core.NewFloat(v), nil
		},
	},
	core.NativeFunction{
		Name:      "math.floor",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Int); err != nil {
				return core.NullValue, err
			}
			v := math.Floor(args[0].ToFloat())
			return core.NewInt64(int64(v)), nil
		},
	},
	core.NativeFunction{
		Name:      "math.ceil",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Int); err != nil {
				return core.NullValue, err
			}
			v := math.Ceil(args[0].ToFloat())
			return core.NewInt64(int64(v)), nil
		},
	},
	core.NativeFunction{
		Name:      "math.round",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {

			l := len(args)
			if l > 2 {
				return core.NullValue, fmt.Errorf("expected 1 or 2 params, got %d", l)
			}

			f := args[0]
			switch f.Type {
			case core.Float, core.Int:
			default:
				return core.NullValue, fmt.Errorf("expected parameter 1 to be a number, got %s", f.TypeName())
			}

			if l == 1 {
				v := math.Round(f.ToFloat())
				return core.NewInt64(int64(v)), nil
			}

			d := args[1]
			if d.Type != core.Int {
				return core.NullValue, fmt.Errorf("expected parameter 2 to be int, got %s", d.TypeName())
			}

			i := math.Pow10(int(d.ToInt()))
			v := math.Round(f.ToFloat()*i) / i
			return core.NewFloat(v), nil
		},
	},
	core.NativeFunction{
		Name:      "math.rand",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Int); err != nil {
				return core.NullValue, err
			}
			v := rand.Intn(int(args[0].ToInt()))
			return core.NewInt(v), nil
		},
	},
	core.NativeFunction{
		Name:      "math.median",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Array); err != nil {
				return core.NullValue, err
			}

			a := args[0].ToArray()

			values := make([]float64, len(a))

			for i, v := range a {
				switch v.Type {
				case core.Int, core.Float:
					values[i] = v.ToFloat()
				default:
					return core.NullValue, fmt.Errorf("element at %d is not a number: %s", i, v.TypeName())
				}
			}

			r := median(values)
			return core.NewFloat(r), nil
		},
	},
	core.NativeFunction{
		Name:      "math.min",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Array); err != nil {
				return core.NullValue, err
			}

			a := args[0].ToArray()
			var min float64

			for i, v := range a {
				switch v.Type {
				case core.Int, core.Float:
					k := v.ToFloat()
					if i == 0 {
						min = k
					} else if k < min {
						min = k
					}
				default:
					return core.NullValue, fmt.Errorf("element at %d is not a number: %s", i, v.TypeName())
				}
			}

			return core.NewFloat(min), nil
		},
	},
	core.NativeFunction{
		Name:      "math.standardDev",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Array); err != nil {
				return core.NullValue, err
			}

			a := args[0].ToArray()

			values := make([]float64, len(a))
			for i, v := range a {
				switch v.Type {
				case core.Int, core.Float:
					values[i] = v.ToFloat()
				default:
					return core.NullValue, fmt.Errorf("element at %d is not a number: %s", i, v.TypeName())
				}
			}

			m := median(values)
			d := stdDev(values, m)
			return core.NewFloat(d), nil
		},
	},
}

func median(numbers []float64) float64 {
	middle := len(numbers) / 2
	result := numbers[middle]
	if len(numbers)%2 == 0 {
		result = (result + numbers[middle-1]) / 2
	}
	return result
}

func stdDev(numbers []float64, mean float64) float64 {
	total := 0.0
	for _, number := range numbers {
		total += math.Pow(number-mean, 2)
	}
	variance := total / float64(len(numbers)-1)
	return math.Sqrt(variance)
}
