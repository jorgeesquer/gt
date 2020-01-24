package lib

import (
	"errors"
	"fmt"
	"io"

	"github.com/gtlang/filesystem"
	"github.com/gtlang/gt/binary"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(Bytecode, `
	
declare namespace bytecode {
    /**
     * 
     * @param path 
     * @param fileSystem 
     * @param scriptMode if statements outside of functions are allowed.
     */
    export function compile(path: string, fileSystem?: io.FileSystem): runtime.Program

    export function compileStr(code: string): runtime.Program

    /**
     * Load a binary program from the file system
     * @param path the path to the main binary.
     * @param fs the file trusted. If empty it will use the current fs.
     */
    export function load(path: string, fs?: io.FileSystem): runtime.Program

    export function loadProgram(b: byte[]): runtime.Program
    export function readProgram(r: io.Reader): runtime.Program
    export function writeProgram(w: io.Writer, p: runtime.Program): void
}

`)
}

var Bytecode = []core.NativeFunction{
	core.NativeFunction{
		Name:      "bytecode.compile",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			return compile(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "bytecode.compileStr",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}
			code := args[0].ToString()

			p, err := core.CompileStr(code)
			if err != nil {
				return core.NullValue, errors.New(err.Error())
			}

			return core.NewObject(&program{prog: p}), nil
		},
	},
	core.NativeFunction{
		Name:      "bytecode.loadProgram",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateArgs(args, core.Bytes); err != nil {
				return core.NullValue, err
			}

			p, err := binary.Load(args[0].ToBytes())
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(&program{prog: p}), nil
		},
	},
	core.NativeFunction{
		Name:      "bytecode.load",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateOptionalArgs(args, core.String, core.Bool, core.Object); err != nil {
				return core.NullValue, err
			}

			l := len(args)
			if l < 1 || l > 3 {
				return core.NullValue, fmt.Errorf("expected 1 to 3 args, got %d", l)
			}

			path := args[0].ToString()
			var fs filesystem.FS

			if l > 1 {
				vfs, ok := args[1].ToObjectOrNil().(*FileSystemObj)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a filesystem, got %v", args[1])
				}
				fs = vfs.FS
			} else {
				fs = vm.FileSystem
			}

			f, err := fs.Open(path)
			if err != nil {
				return core.NullValue, err
			}
			defer f.Close()

			p, err := binary.Read(f)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(&program{prog: p}), nil
		},
	},
	core.NativeFunction{
		Name:      "bytecode.readProgram",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateArgs(args, core.Object); err != nil {
				return core.NullValue, err
			}

			r, ok := args[0].ToObjectOrNil().(io.Reader)
			if !ok {
				return core.NullValue, fmt.Errorf("expected parameter 1 to be io.Reader, got %T", args[0].ToObjectOrNil())
			}

			p, err := binary.Read(r)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(&program{prog: p}), nil
		},
	},
	core.NativeFunction{
		Name:      "bytecode.writeProgram",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateArgs(args, core.Object, core.Object); err != nil {
				return core.NullValue, err
			}

			w, ok := args[0].ToObjectOrNil().(io.Writer)
			if !ok {
				return core.NullValue, fmt.Errorf("expected parameter 1 to be io.Reader, got %T", args[0].ToObjectOrNil())
			}

			p, ok := args[1].ToObjectOrNil().(*program)
			if !ok {
				return core.NullValue, fmt.Errorf("expected parameter 2 to be a program, got %T", args[0].ToObjectOrNil())
			}

			if err := binary.Write(w, p.prog); err != nil {
				return core.NullValue, err
			}

			return core.NullValue, nil
		},
	},
}

func compile(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateOptionalArgs(args, core.String, core.Bool, core.Bool, core.Object); err != nil {
		return core.NullValue, err
	}

	path := args[0].ToString()

	var fs filesystem.FS

	l := len(args)

	if l > 1 {
		vfs, ok := args[1].ToObjectOrNil().(*FileSystemObj)
		if !ok {
			return core.NullValue, fmt.Errorf("expected a filesystem, got %v", args[1])
		}
		fs = vfs.FS
	} else {
		fs = vm.FileSystem
	}

	p, err := core.Compile(fs, path)
	if err != nil {
		return core.NullValue, fmt.Errorf("compiling %s: %v", path, err)
	}

	return core.NewObject(&program{prog: p}), nil
}
