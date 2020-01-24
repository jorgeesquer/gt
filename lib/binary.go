package lib

import (
	"encoding/binary"

	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(Binary, `
	
declare namespace binary {
    export function putInt16LittleEndian(v: byte[], n: number): void
    export function putInt32LittleEndian(v: byte[], n: number): void
    export function putInt16BigEndian(v: byte[], n: number): void
    export function putInt32BigEndian(v: byte[], n: number): void

    export function int16LittleEndian(v: byte[]): number
    export function int32LittleEndian(v: byte[]): number
    export function int16BigEndian(v: byte[]): number
    export function int32BigEndian(v: byte[]): number
    export function int64BigEndian(v: byte[]): number
}
 
`)
}

var Binary = []core.NativeFunction{
	core.NativeFunction{
		Name:      "binary.putInt16LittleEndian",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Bytes, core.Int); err != nil {
				return core.NullValue, err
			}

			b := args[0].ToBytes()
			i := args[1].ToInt()
			binary.LittleEndian.PutUint16(b, uint16(i))
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "binary.putInt32LittleEndian",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Bytes, core.Int); err != nil {
				return core.NullValue, err
			}

			b := args[0].ToBytes()
			i := args[1].ToInt()
			binary.LittleEndian.PutUint32(b, uint32(i))
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "binary.putInt16BigEndian",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Bytes, core.Int); err != nil {
				return core.NullValue, err
			}

			b := args[0].ToBytes()
			i := args[1].ToInt()
			binary.BigEndian.PutUint16(b, uint16(i))
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "binary.putInt32BigEndian",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Bytes, core.Int); err != nil {
				return core.NullValue, err
			}

			b := args[0].ToBytes()
			i := args[1].ToInt()
			binary.BigEndian.PutUint32(b, uint32(i))
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "binary.int16LittleEndian",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Bytes); err != nil {
				return core.NullValue, err
			}

			b := args[0].ToBytes()
			i := binary.LittleEndian.Uint16(b)
			return core.NewInt64(int64(i)), nil
		},
	},
	core.NativeFunction{
		Name:      "binary.int32LittleEndian",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Bytes); err != nil {
				return core.NullValue, err
			}

			b := args[0].ToBytes()
			i := binary.LittleEndian.Uint32(b)
			return core.NewInt64(int64(i)), nil
		},
	},
	core.NativeFunction{
		Name:      "binary.int16BigEndian",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Bytes); err != nil {
				return core.NullValue, err
			}

			b := args[0].ToBytes()
			i := binary.BigEndian.Uint16(b)
			return core.NewInt64(int64(i)), nil
		},
	},
	core.NativeFunction{
		Name:      "binary.int32BigEndian",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Bytes); err != nil {
				return core.NullValue, err
			}

			b := args[0].ToBytes()
			i := binary.BigEndian.Uint32(b)
			return core.NewInt64(int64(i)), nil
		},
	},
	core.NativeFunction{
		Name:      "binary.int64BigEndian",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Bytes); err != nil {
				return core.NullValue, err
			}

			b := args[0].ToBytes()
			i := binary.BigEndian.Uint64(b)
			return core.NewInt64(int64(i)), nil
		},
	},
}
