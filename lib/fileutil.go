package lib

import (
	"fmt"
	"io"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(FileUtil, `

declare namespace fileutil { 
    export function copy(src: string, dst: string): byte[]
}

`)
}

var FileUtil = []core.NativeFunction{
	core.NativeFunction{
		Name:      "fileutil.copy",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			src := args[0].ToString()
			dst := args[1].ToString()

			fs := vm.FileSystem
			if fs == nil {
				return core.NullValue, fmt.Errorf("no filesystem")
			}

			r, err := fs.Open(src)
			if err != nil {
				return core.NullValue, err
			}

			w, err := fs.OpenForWrite(dst)
			if err != nil {
				return core.NullValue, err
			}

			if _, err := io.Copy(w, r); err != nil {
				return core.NullValue, err
			}

			return core.NullValue, nil
		},
	},
}
