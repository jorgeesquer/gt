package lib

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"github.com/gtlang/gt/core"

	"github.com/google/uuid"
)

func init() {
	core.RegisterLib(OS, `

declare namespace os {
    export const stdin: io.File
    export const stdout: io.File
    export const stderr: io.File
    export const fileSystem: io.FileSystem

    export function readLine(): string

    export function exec(name: string, ...params: string[]): string

    /**
     * Reads an environment variable.
     */
    export function getEnv(key: string): string
    /**
     * Sets an environment variable.
     */
    export function setEnv(key: string, value: string): void

    export function exit(code?: number): void

    export const homeDir: string
    export const pathSeparator: string
	
    export function mapPath(path: string): string

    export function newUUID(): string

    export function newCommand(name: string, ...params: any[]): Command

    export interface Command {
        dir: string
        env: string[]
        stdin: io.File
        stdout: io.File
        stderr: io.File

        run(): void
        start(): void
        output(): void
        combinedOutput(): void
    }
}


`)
}

var OS = []core.NativeFunction{
	core.NativeFunction{
		Name: "os.newUUID",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			id, err := uuid.NewUUID()
			if err != nil {
				return core.NullValue, err
			}
			return core.NewString(id.String()), nil
		},
	},
	core.NativeFunction{
		Name: "->os.pathSeparator",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewString(string(os.PathSeparator)), nil
		},
	},
	core.NativeFunction{
		Name: "->os.homeDir",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}
			u, err := user.Current()
			if err != nil {
				return core.NullValue, ErrUnauthorized
			}
			return core.NewString(u.HomeDir), nil
		},
	},
	core.NativeFunction{
		Name: "->os.stdout",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			f := &file{f: os.Stdout}
			return core.NewObject(f), nil
		},
	},
	core.NativeFunction{
		Name: "->os.stdin",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			f := &file{f: os.Stdin}
			return core.NewObject(f), nil
		},
	},
	core.NativeFunction{
		Name: "->os.stderr",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			f := &file{f: os.Stderr}
			return core.NewObject(f), nil
		},
	},
	core.NativeFunction{
		Name:      "os.mapPath",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			p := args[0].ToString()

			if len(p) > 0 && p[0] == '~' {
				usr, err := user.Current()
				if err != nil {
					return core.NullValue, err
				}
				p = filepath.Join(usr.HomeDir, p[1:])
			}

			return core.NewString(p), nil
		},
	},
	core.NativeFunction{
		Name:      "os.exit",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}
			if err := ValidateOptionalArgs(args, core.Int); err != nil {
				return core.NullValue, err
			}

			var exitCode int
			if len(args) > 0 {
				exitCode = int(args[0].ToInt())
			}

			os.Exit(exitCode)
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "os.exec",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			l := len(args)
			if l == 0 {
				return core.NullValue, fmt.Errorf("expected at least 1 argument")
			}

			values := make([]string, l)
			for i, v := range args {
				values[i] = v.ToString()
			}

			cmd := exec.Command(values[0], values[1:]...)
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout

			if err := cmd.Run(); err != nil {
				return core.NullValue, err
			}

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "os.newCommand",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			l := len(args)
			if l == 0 {
				return core.NullValue, fmt.Errorf("expected at least 1 argument")
			}

			values := make([]string, l)
			for i, v := range args {
				values[i] = v.ToString()
			}

			cmd := newCommand(values[0], values[1:]...)

			return core.NewObject(cmd), nil
		},
	},
	core.NativeFunction{
		Name:      "os.readLine",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			r := bufio.NewReader(os.Stdin)
			s, err := r.ReadString('\n')
			if err != nil {
				return core.NullValue, err
			}

			// trim the \n
			s = s[:len(s)-1]

			return core.NewString(s), nil
		},
	},
	core.NativeFunction{
		Name: "->os.fileSystem",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if vm.FileSystem == nil {
				return core.NullValue, nil
			}
			return core.NewObject(NewFileSystem(vm.FileSystem)), nil
		},
	},
	core.NativeFunction{
		Name:      "os.getEnv",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}
			s := os.Getenv(args[0].ToString())
			return core.NewString(s), nil
		},
	},
	core.NativeFunction{
		Name:      "os.setEnv",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}
			if err := ValidateArgs(args, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			if err := os.Setenv(args[0].ToString(), args[1].ToString()); err != nil {
				return core.NullValue, err
			}
			return core.NullValue, nil
		},
	},
}

func newCommand(name string, arg ...string) *command {
	return &command{
		command: exec.Command(name, arg...),
	}
}

type command struct {
	command *exec.Cmd
}

func (*command) Type() string {
	return "os.command"
}

func (c *command) GetMethod(name string) core.NativeMethod {
	switch name {
	case "run":
		return c.run
	case "start":
		return c.start
	case "output":
		return c.output
	case "combinedOutput":
		return c.combinedOutput
	}
	return nil
}

func (c *command) GetProperty(key string, vm *core.VM) (core.Value, error) {
	switch key {
	case "stdin":
		return core.NewObject(c.command.Stdin), nil
	case "stdout":
		return core.NewObject(c.command.Stdout), nil
	case "stderr":
		return core.NewObject(c.command.Stderr), nil
	case "dir":
		return core.NewString(c.command.Dir), nil
	case "env":
		items := c.command.Env
		a := make([]core.Value, len(items))
		for i, v := range items {
			a[i] = core.NewString(v)
		}
		return core.NewArrayValues(a), nil
	}
	return core.UndefinedValue, nil
}

func (c *command) SetProperty(key string, v core.Value, vm *core.VM) error {
	switch key {
	case "stdin":
		if v.Type != core.Object {
			return ErrInvalidType
		}
		o, ok := v.ToObject().(io.Reader)
		if !ok {
			return fmt.Errorf("expected a Reader")
		}
		c.command.Stdin = o
		return nil

	case "stdout":
		if v.Type != core.Object {
			return ErrInvalidType
		}
		o, ok := v.ToObject().(io.Writer)
		if !ok {
			return fmt.Errorf("expected a Writer")
		}
		c.command.Stdout = o
		return nil

	case "stderr":
		if v.Type != core.Object {
			return ErrInvalidType
		}
		o, ok := v.ToObject().(io.Writer)
		if !ok {
			return fmt.Errorf("expected a Writer")
		}
		c.command.Stderr = o
		return nil

	case "dir":
		if v.Type != core.String {
			return ErrInvalidType
		}
		c.command.Dir = v.ToString()
		return nil

	case "env":
		if v.Type != core.Array {
			return ErrInvalidType
		}
		a := v.ToArray()
		b := make([]string, len(a))
		for i, v := range a {
			b[i] = v.ToString()
		}
		c.command.Env = b
		return nil
	}
	return ErrReadOnlyOrUndefined
}

func (c *command) run(args []core.Value, vm *core.VM) (core.Value, error) {
	err := c.command.Run()
	return core.NullValue, err
}

func (c *command) start(args []core.Value, vm *core.VM) (core.Value, error) {
	err := c.command.Start()
	return core.NullValue, err
}

func (c *command) output(args []core.Value, vm *core.VM) (core.Value, error) {
	b, err := c.command.Output()
	if err != nil {
		return core.NullValue, err
	}
	return core.NewString(string(b)), nil
}

func (c *command) combinedOutput(args []core.Value, vm *core.VM) (core.Value, error) {
	b, err := c.command.CombinedOutput()
	if err != nil {
		return core.NullValue, err
	}
	return core.NewString(string(b)), nil
}
