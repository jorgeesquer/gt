package lib

import (
	"bytes"
	"fmt"
	"io"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(IOUtil, `

declare namespace ioutil {
    export function readAll(r: io.Reader): byte[]
}

`)
}

var IOUtil = []core.NativeFunction{
	core.NativeFunction{
		Name:      "ioutil.readAll",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Object); err != nil {
				return core.NullValue, err
			}

			r := args[0].ToObject()

			reader, ok := r.(io.Reader)
			if !ok {
				return core.NullValue, fmt.Errorf("expected a io.Reader, got %v", args[0])
			}

			b, err := ReadAll(reader, vm)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewBytes(b), nil
		},
	},
}

func ReadAll(reader io.Reader, vm *core.VM) ([]byte, error) {
	const SIZE = 250
	b := make([]byte, SIZE)
	var buf bytes.Buffer

	for {
		n, err := reader.Read(b)

		if err := vm.AddAllocations(n); err != nil {
			return nil, err
		}

		buf.Write(b[:n])

		if n < SIZE || err == io.EOF {
			return buf.Bytes(), nil
		}

		if err != nil {
			return nil, err
		}
	}
}
