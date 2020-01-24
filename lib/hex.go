package lib

import (
	"encoding/hex"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(HEX, `

declare namespace hex {
    export function encodeToString(b: byte[]): number
}


`)
}

var HEX = []core.NativeFunction{
	core.NativeFunction{
		Name:      "hex.encodeToString",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Bytes); err != nil {
				return core.NullValue, err
			}

			b := args[0].ToBytes()
			s := hex.EncodeToString(b)
			return core.NewString(s), nil
		},
	},
}
