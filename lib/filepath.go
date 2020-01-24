package lib

import (
	"fmt"
	"path/filepath"
	"github.com/gtlang/gt/core"
	"strings"
)

func init() {
	core.RegisterLib(FilePath, `

declare namespace filepath {
    /**
     * Returns the extension of a path
     */
    export function ext(path: string): string

    /**
     *  Base returns the last element of path.
     *  Trailing path separators are removed before extracting the last element.
     *  If the path is empty, Base returns ".".
     *  If the path consists entirely of separators, Base returns a single separator.
     */
    export function base(path: string): string

    /**
     * Returns name of the file without the directory and extension.
     */
    export function nameWithoutExt(path: string): string

    /**
     * Returns directory part of the path.
     */
    export function dir(path: string): string

    export function join(...parts: string[]): string

    /**
     * joins the elemeents but respecting absolute paths.
     */
    export function joinAbs(...parts: string[]): string
}

`)
}

var FilePath = []core.NativeFunction{
	core.NativeFunction{
		Name:      "filepath.join",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			parts := make([]string, len(args))
			for i, v := range args {
				if v.Type != core.String {
					return core.NullValue, fmt.Errorf("argument %d is not a string (%s)", i, v.TypeName())
				}
				parts[i] = v.ToString()
			}

			path := filepath.Join(parts...)
			return core.NewString(path), nil
		},
	},
	core.NativeFunction{
		Name:      "filepath.joinAbs",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			parts := make([]string, 0, len(args))
			for i, v := range args {
				if v.Type != core.String {
					return core.NullValue, fmt.Errorf("argument %d is not a string (%s)", i, v.TypeName())
				}
				s := v.ToString()
				if strings.HasPrefix(s, "/") {
					parts = nil
				}
				parts = append(parts, s)
			}

			path := filepath.Join(parts...)
			return core.NewString(path), nil
		},
	},
	core.NativeFunction{
		Name:      "filepath.ext",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			path := args[0].ToString()
			ext := filepath.Ext(path)
			return core.NewString(ext), nil
		},
	},
	core.NativeFunction{
		Name:      "filepath.base",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			path := args[0].ToString()
			name := filepath.Base(path)
			return core.NewString(name), nil
		},
	},
	core.NativeFunction{
		Name:      "filepath.nameWithoutExt",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			path := args[0].ToString()
			name := filepath.Base(path)
			if i := strings.LastIndexByte(name, '.'); i != -1 {
				name = name[:i]
			}
			return core.NewString(name), nil
		},
	},
	core.NativeFunction{
		Name:      "filepath.dir",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			path := args[0].ToString()
			name := filepath.Dir(path)
			return core.NewString(name), nil
		},
	},
}
