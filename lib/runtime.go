package lib

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gtlang/gt/lib/x/i18n"

	"github.com/gtlang/filesystem"
	"github.com/gtlang/gt/binary"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(Runtime, `

declare function panic(message: string): void
declare function defer(f: () => void): void;

declare namespace runtime {	
    export const version: string
    export const build: string
}


declare namespace runtime {
    export interface Finalizable {
        close(): void
	}
	
	export function typeDefs(): string

    export function setFileSystem(fs: io.FileSystem): void

    export function setFinalizer(v: runtime.Finalizable): void
    export function newFinalizable(f: () => void): Finalizable

    export function panic(message: string): void

    export type OSName = "linux" | "windows" | "darwin"

    export const pluginManager: PluginManager
    export const globalPluginManager: PluginManager

	export function setGlobalPluginManager(p: PluginManager): void
    export function initGlobalPluginManager(fs?: io.FileSystem): PluginManager

    export const context: Context
    export const global: any

    export function newPluginManager(fs?: io.FileSystem): PluginManager
    export function newPlugin(name: string, program: Program, trusted?: boolean): Plugin
    export function newContext(): Context

    /**
     * Returns the operating system
     */
    export const OS: OSName

    /**
     * Returns the path of the executable.
     */
    export const executable: string

    /**
     * Returns the path of the native runtime executable.
     */
    export const nativeExecutable: string

    export const vm: VirtualMachine


    export function runFunc(func: string, ...args: any[]): any
    export function exec(func: string, ...args: any[]): any
    export function execIfExists(func: string, ...args: any[]): any

    // export function addHook(name: string, func: Function): void
    // export function execHook(name: string, ...params: any[]): void
    // export function anyHook(name: string): boolean

    export function getItem(name: string): any
    export function setItem(name: string, value: any): void

    export const hasResources: boolean
    export function resources(name: string): string[]
    export function resource(name: string): byte[]

    export function getStackTrace(): string
    export function newVM(p: Program, globals?: any[]): VirtualMachine

    export interface Program {
		build: string
        functions(): FunctionInfo[]
        functionInfo(name: string): FunctionInfo
        resources(): string[]
        resource(key: string): byte[]
        setResource(key: string, value: byte[]): void

		directives(): StringMap
		directive(name: string): string
		directiveValues(name: string): string[]
		addDirective(name: string, value: string): void

        /**
         * Strip sources, not exported functions name and other info.
         */
        strip(): void
        toString(): string
        write(w: io.Writer): void
	}
	
    export interface FunctionInfo {
        name: string
        index: number
        arguments: number
        exported: boolean
        func: Function
        toString(): string
    }

    export interface VirtualMachine {
		maxAllocations: number
		maxFrames: number
		maxSteps: number
		readonly steps: number
		readonly fileSystem: io.FileSystem
		readonly program: Program
		context: Context
		trusted: boolean
		error: errors.Error
		initialize(): void
		run(...args: any[]): any
		runFunc(name: string, ...args: any[]): any
		runFunc(index: number, ...args: any[]): any
		runStaticFunc(name: string, ...args: any[]): any
		runStaticFunc(index: number, ...args: any[]): any
		getValue(name: string): any
		getGlobals(): any[]
		getStackTrace(): string
		setFileSystem(v: io.FileSystem): void
		getItem(name: string): any
		setItem(name: string, v: any): void
		clone(): VirtualMachine
		resetSteps(): void
    }

    export interface Context {
        db: sql.DB
        guid: string
        plugin: Plugin
		pluginManager: PluginManager
        caller: Plugin
        pluginName: string
        userCulture: string
        culture: i18n.Culture
        location: time.Location
        tenant: string
        tenantLabel: string
        tenantIcon: string
        debug: boolean
        monoTenant: string
        test: boolean
        items: { [key: string]: any }
        dataFS: io.FileSystem
        errorLogger: io.File
        addPlugin(name: string): void
        hasPlugin(name: string): boolean
        getPlugins(): string[]
        setPlugins(v: string[]): void
        clone(): Context
        exec(func: string, ...args: any[]): any
		execIfExists(func: string, ...args: any[]): any
    }

    export interface PluginManager {
        debug: boolean
        fileSystem: io.FileSystem
        getPlugin(plugin: string): Plugin
        allPlugins(): Plugin[]
        loadPlugin(plugin: string | Plugin, trusted?: boolean): Plugin
        reloadPlugin(plugin: string, trusted?: boolean): Plugin
		clone(): PluginManager
        clear(name?: string): void
        runFunc(func: string, ...args: any[]): any
        exec(c: Context, func: string, ...args: any[]): any
        execIfExists(c: Context, func: string, ...args: any[]): any
        copy(): PluginManager

        addHook(name: string, func: Function): void
        execHook(name: string, ...params: any[]): void
        anyHook(name: string): boolean
    }

    export interface Plugin {
        name: string
        program: Program
        globals: any[]
        setGlobals(v: any[]): void
    }
}


`)
}

func HasProgramPermission(p *core.Program, name string) bool {
	for _, v := range p.Directives {
		// should check the word only but this a self check.
		// GM only runs its own plugins.
		if strings.Contains(v, "trusted") || strings.Contains(v, name) {
			return true
		}
	}

	return false
}

var globalContext = core.NewMap(10)

var globalPluginManager *PluginManager
var mut = &sync.RWMutex{}

// the path of the excuting binary
var Executable string

var Runtime = []core.NativeFunction{
	core.NativeFunction{
		Name:      "panic",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			panic(args[0].ToString())
		},
	},
	core.NativeFunction{
		Name: "->runtime.version",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewString(core.VERSION), nil
		},
	},
	core.NativeFunction{
		Name: "->runtime.build",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewString(core.BUILD), nil
		},
	},
	core.NativeFunction{
		Name:      "->runtime.pluginManager",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			pm, err := getPluginManager(vm)
			if err != nil {
				return core.NullValue, err
			}

			if pm == nil {
				return core.NullValue, nil
			}

			return core.NewObject(pm), nil
		},
	},
	core.NativeFunction{
		Name:      "->runtime.globalPluginManager",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			var pm *PluginManager
			mut.RLock()
			pm = globalPluginManager
			mut.RUnlock()

			if pm == nil {
				return core.NullValue, nil
			}

			return core.NewObject(pm), nil
		},
	},
	core.NativeFunction{
		Name: "runtime.typeDefs",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args); err != nil {
				return core.NullValue, err
			}
			s := core.TypeDefs()
			return core.NewString(s), nil
		},
	},
	core.NativeFunction{
		Name:      "runtime.setGlobalPluginManager",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}
			if err := ValidateArgs(args, core.Object); err != nil {
				return core.NullValue, err
			}

			pm, ok := args[0].ToObjectOrNil().(*PluginManager)
			if !ok {
				return core.NullValue, fmt.Errorf("expected a PluginManager, got %v", args[0].TypeName())
			}

			mut.Lock()
			globalPluginManager = pm
			mut.Unlock()

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "runtime.initGlobalPluginManager",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if globalPluginManager != nil {
				return core.NullValue, fmt.Errorf("the plugin manager has already been initialized")
			}

			if err := ValidateOptionalArgs(args, core.Object); err != nil {
				return core.NullValue, err
			}

			var fs filesystem.FS

			ln := len(args)

			if ln > 0 {
				a := args[0]
				switch a.Type {
				case core.Null, core.Undefined:
				case core.Object:
					f, ok := a.ToObject().(*FileSystemObj)
					if !ok {
						return core.NullValue, fmt.Errorf("invalid filesystem argument, got %v", a)
					}
					fs = f.FS
				default:
					return core.NullValue, fmt.Errorf("expected a filesystem, got %s", a.TypeName())
				}
			}

			mut.Lock()
			globalPluginManager = newPluginManager(fs)
			mut.Unlock()

			return core.NewObject(globalPluginManager), nil
		},
	},
	core.NativeFunction{
		Name:      "runtime.newContext",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			c := &Context{}
			return core.NewObject(c), nil
		},
	},
	core.NativeFunction{
		Name:      "runtime.runFunc",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if len(args) == 0 {
				return core.NullValue, fmt.Errorf("expected at least the function name")
			}

			a := args[0]
			if a.Type != core.String {
				return core.NullValue, fmt.Errorf("function name must be a string, got %v", a.Type)
			}

			funcName := a.ToString()

			return vm.RunFunc(funcName, args[1:]...)
		},
	},
	core.NativeFunction{
		Name:      "runtime.exec",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			ln := len(args)
			if ln < 1 {
				return core.NullValue, fmt.Errorf("expected at least 2 args, got %d", ln)
			}

			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be a string, got %s", args[0].TypeName())
			}

			pm, err := getPluginManager(vm)
			if err != nil {
				return core.NullValue, err
			}

			if pm == nil {
				return core.NullValue, fmt.Errorf("there is no plugin manager set")
			}

			c := GetContext(vm)
			return pm.execPlugin(c, args[0].ToString(), args[1:], false, vm)
		},
	},
	core.NativeFunction{
		Name:      "runtime.execIfExists",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			ln := len(args)
			if ln < 1 {
				return core.NullValue, fmt.Errorf("expected at least 2 args, got %d", ln)
			}

			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be a string, got %s", args[0].TypeName())
			}

			pm, err := getPluginManager(vm)
			if err != nil {
				return core.NullValue, err
			}

			if pm == nil {
				return core.NullValue, fmt.Errorf("there is no plugin manager set")
			}

			c := GetContext(vm)
			return pm.execPlugin(c, args[0].ToString(), args[1:], true, vm)
		},
	},
	core.NativeFunction{
		Name: "->runtime.resources",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			res := vm.Program.Resources
			if res == nil {
				return core.NewArray(0), nil
			}

			a := make([]core.Value, len(res))

			i := 0
			for k := range res {
				a[i] = core.NewString(k)
				i++
			}

			return core.NewArrayValues(a), nil
		},
	},
	core.NativeFunction{
		Name:      "runtime.resource",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			name := args[0].ToString()

			res := vm.Program.Resources
			if res == nil {
				return core.NullValue, nil
			}

			v, ok := res[name]
			if !ok {
				return core.NullValue, nil
			}

			return core.NewBytes(v), nil
		},
	},
	core.NativeFunction{
		Name: "->runtime.hasResources",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			res := vm.Program.Resources
			if len(res) == 0 {
				return core.FalseValue, nil
			}

			return core.TrueValue, nil
		},
	},
	core.NativeFunction{
		Name: "->runtime.global",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}
			return globalContext, nil
		},
	},
	core.NativeFunction{
		Name: "->runtime.context",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewObject(GetContext(vm)), nil
		},
	},
	core.NativeFunction{
		Name:      "runtime.setFileSystem",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateArgs(args, core.Object); err != nil {
				return core.NullValue, err
			}

			fs, ok := args[0].ToObject().(*FileSystemObj)
			if !ok {
				return core.NullValue, fmt.Errorf("expected a fileSystem, got %s", args[0].TypeName())
			}
			vm.FileSystem = fs.FS
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "runtime.getItem",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}
			s := GetContext(vm).getProtectedItem(args[0].ToString())
			return s, nil
		},
	},
	core.NativeFunction{
		Name:      "runtime.setItem",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected param 1 to be string, got %s", args[0].TypeName())
			}

			GetContext(vm).setProtectedItem(args[0].ToString(), args[1])
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "runtime.newFinalizable",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			v := args[0]

			fin, err := newFinalizable(v, vm)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(fin), nil
		},
	},
	core.NativeFunction{
		Name:      "defer",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			v := args[0]

			fin, err := newFinalizable(v, vm)
			if err != nil {
				return core.NullValue, err
			}

			vm.SetFinalizer(fin)
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "runtime.setFinalizer",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			v := args[0]

			if v.Type != core.Object {
				return core.NullValue, fmt.Errorf("the value is not a finalizer")
			}

			fin, ok := v.ToObject().(core.Finalizable)
			if !ok {
				return core.NullValue, fmt.Errorf("the value is not a finalizer")
			}
			vm.SetFinalizer(fin)
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "errors.wrap",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return wrap(false, args, vm)
		},
	},
	core.NativeFunction{
		Name:      "errors.public",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return wrap(true, args, vm)
		},
	},
	core.NativeFunction{
		Name: "->runtime.OS",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}
			return core.NewString(runtime.GOOS), nil
		},
	},
	core.NativeFunction{
		Name: "->runtime.executable",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}
			return core.NewString(Executable), nil
		},
	},
	core.NativeFunction{
		Name: "->runtime.nativeExecutable",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}
			ex, err := os.Executable()
			if err != nil {
				return core.NullValue, err
			}
			return core.NewString(ex), nil
		},
	},
	core.NativeFunction{
		Name:      "runtime.newVM",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			l := len(args)
			if l == 0 || l > 2 {
				return core.NullValue, fmt.Errorf("expected 1 or 2 params, got %d", l)
			}

			if args[0].Type != core.Object {
				return core.NullValue, fmt.Errorf("argument 1 must be a program, got %s", args[0].TypeName())
			}
			p, ok := args[0].ToObject().(*program)
			if !ok {
				return core.NullValue, fmt.Errorf("argument 1 must be a program, got %s", args[0].TypeName())
			}

			var m *core.VM

			if l == 1 {
				m = core.NewVM(p.prog)
			} else {
				switch args[1].Type {
				case core.Undefined, core.Null:
					m = core.NewVM(p.prog)
				case core.Array:
					m = core.NewInitializedVM(p.prog, args[1].ToArray())
				default:
					return core.NullValue, fmt.Errorf("argument 2 must be an array, got %s", args[1].TypeName())
				}
			}

			m.MaxAllocations = vm.MaxAllocations
			m.MaxFrames = vm.MaxFrames
			m.MaxSteps = vm.MaxSteps

			if err := m.AddSteps(vm.Steps()); err != nil {
				return core.NullValue, err
			}

			return core.NewObject(&libVM{m}), nil
		},
	},
	core.NativeFunction{
		Name: "->runtime.vm",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}
			return core.NewObject(&libVM{vm}), nil
		},
	},
	core.NativeFunction{
		Name: "runtime.resetSteps",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}
			vm.ResetSteps()
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "runtime.getStackTrace",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := strings.Join(vm.Stacktrace(), "\n")
			return core.NewString(s), nil
		},
	},
	core.NativeFunction{
		Name:      "runtime.newPluginManager",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateOptionalArgs(args, core.Object); err != nil {
				return core.NullValue, err
			}

			var fs filesystem.FS

			ln := len(args)

			if ln > 0 {
				a := args[0]
				switch a.Type {
				case core.Null, core.Undefined:
				case core.Object:
					f, ok := a.ToObject().(*FileSystemObj)
					if !ok {
						return core.NullValue, fmt.Errorf("invalid filesystem argument, got %v", a)
					}
					fs = f.FS
				default:
					return core.NullValue, fmt.Errorf("expected a filesystem, got %s", a.TypeName())
				}
			}

			p := newPluginManager(fs)
			return core.NewObject(p), nil
		},
	},
	core.NativeFunction{
		Name:      "runtime.newPlugin",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateArgRange(args, 1, 2); err != nil {
				return core.NullValue, err
			}

			a := args[0]
			if a.Type != core.String {
				return core.NullValue, fmt.Errorf("expected argument 1 to be string, got %v", a.TypeName())
			}

			name := a.ToString()

			plg := &plugin{name: name}

			l := len(args)

			if l > 1 {
				b := args[1]
				switch b.Type {
				case core.Null, core.Undefined:
				case core.Object:
					p, ok := args[1].ToObject().(*program)
					if !ok {
						return core.NullValue, fmt.Errorf("expected argument 2 to be program, got %v", b.TypeName())
					}
					plg.program = p.prog
				default:
					return core.NullValue, fmt.Errorf("expected argument 2 to be program, got %v", b.TypeName())
				}
			}

			return core.NewObject(plg), nil
		},
	},
}

func wrap(public bool, args []core.Value, vm *core.VM) (core.Value, error) {
	ln := len(args)
	if ln < 1 || ln > 2 {
		return core.NullValue, fmt.Errorf("expected 1 or 2 parameters, got %d", ln)
	}

	v := args[0]
	if v.Type != core.String {
		return core.NullValue, fmt.Errorf("expected parameter 1 to be a string, got %s", v.Type)
	}

	e := vm.NewError(v.ToString())

	if public {
		e.SetPublic(true)
	}

	if ln > 1 {
		innerObj := args[1]
		switch innerObj.Type {

		case core.Null, core.Undefined:

		case core.String:
			innerEx := vm.NewError(innerObj.ToString())
			e.Wrap(innerEx)

		case core.Object:
			if innerObj.Type != core.Object {
				return core.NullValue, fmt.Errorf("expected parameter 2 to be a Exception, got %s", innerObj.Type)
			}
			innerEx, ok := innerObj.ToObject().(core.Error)
			if !ok {
				return core.NullValue, fmt.Errorf("expected parameter 2 to be a Exception, got %s", innerEx.Type())
			}
			e.Wrap(innerEx)

		default:
			return core.NullValue, fmt.Errorf("expected parameter 2 to be a Exception, got %s", innerObj.Type)
		}
	}

	return core.NewObject(e), nil
}

func getPluginManager(vm *core.VM) (*PluginManager, error) {
	c := GetContext(vm)
	if c.PluginManager != nil {
		return c.PluginManager, nil
	}

	var pm *PluginManager
	mut.RLock()
	pm = globalPluginManager
	mut.RUnlock()

	return pm, nil
}

type plugin struct {
	name    string
	program *core.Program
	globals []core.Value
}

func (*plugin) Type() string {
	return "Plugin"
}

func (p *plugin) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "name":
		return core.NewString(p.name), nil
	case "program":
		return core.NewObject(&program{prog: p.program}), nil
	case "globals":
		return core.NewArrayValues(p.globals), nil
	}

	return core.UndefinedValue, nil
}

func (p *plugin) GetMethod(name string) core.NativeMethod {
	switch name {
	case "setGlobals":
		return p.setGlobals
	}
	return nil
}

func (p *plugin) setGlobals(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgs(args, core.Array); err != nil {
		return core.NullValue, err
	}

	p.globals = args[0].ToArray()

	return core.NullValue, nil
}

func newPluginManager(fs filesystem.FS) *PluginManager {
	return &PluginManager{
		fs:      fs,
		plugins: make(map[string]*plugin),
		hooks:   make(map[string][]HookFunction),
	}
}

type PluginManager struct {
	sync.Mutex
	fs      filesystem.FS
	plugins map[string]*plugin

	debug      bool
	pluginsDir string

	// allow to suscribe to events between plugins
	hooks map[string][]HookFunction
}

type HookFunction struct {
	Plugin   string
	Function int
}

func (*PluginManager) Type() string {
	return "PluginManager"
}

func (m *PluginManager) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "debug":
		return core.NewBool(m.debug), nil
	case "pluginsDir":
		return core.NewString(m.pluginsDir), nil
	case "fileSystem":
		if !vm.HasPermission("trusted") {
			return core.NullValue, ErrUnauthorized
		}
		fs := NewFileSystem(m.fs)
		return core.NewObject(fs), nil
	}
	return core.UndefinedValue, nil
}

func (m *PluginManager) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "debug":
		if !vm.HasPermission("trusted") {
			return ErrUnauthorized
		}
		switch v.Type {
		case core.Bool:
			m.debug = v.ToBool()
			return nil
		case core.Undefined, core.Null:
			m.debug = false
			return nil
		default:
			return ErrInvalidType
		}

	case "pluginsDir":
		switch v.Type {
		case core.String:
			if !vm.HasPermission("trusted") {
				return ErrUnauthorized
			}
			m.pluginsDir = v.ToString()
			return nil
		default:
			return ErrInvalidType
		}
	}

	return ErrReadOnlyOrUndefined
}

func (m *PluginManager) GetMethod(name string) core.NativeMethod {
	switch name {
	case "allPlugins":
		return m.allPlugins
	case "getPlugin":
		return m.getPlugin
	case "loadPlugin":
		return m.loadPlugin
	case "reloadPlugin":
		return m.reloadPlugin
	case "clone":
		return m.clone
	case "clear":
		return m.clear
	case "exec":
		return m.exec
	case "execIfExists":
		return m.execIfExists
	case "addHook":
		return m.addHook
	case "execHook":
		return m.execHook
	case "anyHook":
		return m.anyHook
	case "copy":
		return m.copy
	case "lock":
		return m.lock
	case "unlock":
		return m.unlock
	case "runFunc":
		return m.runFunc
	}
	return nil
}

func (m *PluginManager) lock(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		// Important or it could run arbitrary code by creating a system.xxxx
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgRange(args, 0, 0); err != nil {
		return core.NullValue, err
	}

	m.Lock()
	return core.NullValue, nil
}

func (m *PluginManager) unlock(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		// Important or it could run arbitrary code by creating a system.xxxx
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgRange(args, 0, 0); err != nil {
		return core.NullValue, err
	}

	m.Unlock()
	return core.NullValue, nil
}

func (m *PluginManager) getPluginPath(name string) (string, error) {
	path := name + ".gt"
	return path, nil
}

func getPluginName(path string) (string, error) {
	name := strings.TrimSuffix(path, ".gt")

	if !IsAlphanumericIdent(name) {
		return "", fmt.Errorf("invalid plugin name %s", path)
	}

	return name, nil
}

func validateHookName(name string, vm *core.VM) error {
	c := GetContext(vm)
	p := c.Plugin
	if p != nil {
		if !strings.HasPrefix(name, p.name+".") {
			return fmt.Errorf("invalid hook name. Must start with %s. Got %s", p.name, name)
		}
	}
	return nil
}

func (m *PluginManager) copy(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		// Important or it could run arbitrary code by creating a system.xxxx
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	cp := &PluginManager{
		fs:         m.fs,
		debug:      m.debug,
		pluginsDir: m.pluginsDir,
		plugins:    make(map[string]*plugin),
		hooks:      make(map[string][]HookFunction),
	}

	for k, v := range m.plugins {
		cp.plugins[k] = v
	}

	for k, v := range m.hooks {
		cp.hooks[k] = v
	}

	return core.NewObject(cp), nil

}

func (m *PluginManager) anyHook(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) == 0 {
		return core.NullValue, fmt.Errorf("you must provide a hook name")
	}

	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected a hook name, got %s", args[0].TypeName())
	}

	name := args[0].ToString()

	hooks := m.hooks[name]

	if hooks == nil {
		return core.FalseValue, nil
	}

	if len(hooks) == 0 {
		return core.FalseValue, nil

	}

	return core.TrueValue, nil
}

func (m *PluginManager) addHook(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String, core.Func); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	c := GetContext(vm)
	p := c.Plugin
	if p == nil {
		return core.NullValue, fmt.Errorf("the current context is not a plugin")
	}

	i := args[1].ToFunction()
	f := HookFunction{
		Plugin:   p.name,
		Function: i,
	}

	m.hooks[name] = append(m.hooks[name], f)
	return core.NullValue, nil
}

func (m *PluginManager) execHook(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) == 0 {
		return core.NullValue, fmt.Errorf("you must provide a hook name")
	}

	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected a hook name, got %s", args[0].TypeName())
	}

	name := args[0].ToString()

	if err := validateHookName(name, vm); err != nil {
		return core.NullValue, err
	}

	hooks := m.hooks[name]
	if hooks == nil {
		// nobody is suscribed
		return core.NullValue, nil
	}

	// Remove the hook name: execHook(name, ...)
	args = args[1:]

	c := GetContext(vm)
	for _, hook := range hooks {
		pluginName := hook.Plugin
		if !hasPluginActive(c, pluginName) {
			continue
		}

		path, err := m.getPluginPath(pluginName)
		if err != nil {
			return core.NullValue, err
		}

		m.Lock()
		p := m.plugins[path]
		m.Unlock()

		f := p.program.Functions[hook.Function]
		if _, err := doExecFunction(c, f, p, args, vm); err != nil {
			return core.NullValue, err
		}
	}

	return core.NullValue, nil
}

func hasPluginActive(c *Context, name string) bool {
	for _, p := range c.Plugins {
		if p == name {
			return true
		}
	}
	return false
}

func parsePluginFunctionName(name string) (string, string, error) {
	parts := Split(name, ".")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("the format is plugin.function, got %s", strings.Join(parts, "."))
	}

	return parts[0], parts[1], nil
}

func (m *PluginManager) exec(args []core.Value, vm *core.VM) (core.Value, error) {
	ln := len(args)
	if ln < 2 {
		return core.NullValue, fmt.Errorf("expected at least 2 args, got %d", ln)
	}

	c, ok := args[0].ToObjectOrNil().(*Context)
	if !ok {
		return core.NullValue, fmt.Errorf("expected arg 1 to be a Context, got %s", args[0].TypeName())
	}

	if args[1].Type != core.String {
		return core.NullValue, fmt.Errorf("expected arg 2 to be a string, got %s", args[1].TypeName())
	}
	return m.execPlugin(c, args[1].ToString(), args[2:], false, vm)
}

func (m *PluginManager) execIfExists(args []core.Value, vm *core.VM) (core.Value, error) {
	ln := len(args)
	if ln < 2 {
		return core.NullValue, fmt.Errorf("expected at least 2 args, got %d", ln)
	}

	c, ok := args[0].ToObjectOrNil().(*Context)
	if !ok {
		return core.NullValue, fmt.Errorf("expected arg 1 to be a Context, got %s", args[0].TypeName())
	}

	if args[1].Type != core.String {
		return core.NullValue, fmt.Errorf("expected arg 2 to be a string, got %s", args[1].TypeName())
	}

	return m.execPlugin(c, args[1].ToString(), args[2:], true, vm)
}

func (m *PluginManager) execPlugin(c *Context, fullName string, args []core.Value, onlyIfExists bool, vm *core.VM) (core.Value, error) {
	plugin, funcName, err := parsePluginFunctionName(fullName)
	if err != nil {
		return core.NullValue, err
	}

	if !hasPluginActive(c, plugin) {
		if onlyIfExists {
			return core.NullValue, nil
		}
		return core.NullValue, fmt.Errorf("the plugin is not installed: %s", plugin)
	}

	path, err := m.getPluginPath(plugin)
	if err != nil {
		return core.NullValue, err
	}

	return m.doExec(c, path, funcName, onlyIfExists, true, args, vm)
}

func (m *PluginManager) runFunc(args []core.Value, vm *core.VM) (core.Value, error) {
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected arg 1 to be a string, got %s", args[0].TypeName())
	}

	pluginName, funcName, err := parsePluginFunctionName(args[0].ToString())
	if err != nil {
		return core.NullValue, err
	}

	path, err := m.getPluginPath(pluginName)
	if err != nil {
		return core.NullValue, err
	}

	p := m.plugins[path]
	if p == nil {
		// don't lock here. Plugins must be loaded before calling to prevent
		// races in client code (this can be called from init funcs)
		return core.NullValue, fmt.Errorf("plugin not loaded %s", path)
	}

	f, ok := p.program.Function(funcName)
	if !ok {
		return core.NullValue, fmt.Errorf("the function doesn't exist: %s", funcName)
	}

	if !f.Exported {
		return core.NullValue, fmt.Errorf("invalid function: %s is not exported", funcName)
	}

	cloneVM := vm.Clone(p.program, p.globals)
	v, err := cloneVM.RunFuncIndex(f.Index, args[1:]...)
	if err != nil {
		return core.NullValue, err
	}

	return v, nil
}

func (m *PluginManager) doExec(c *Context, path, function string, onlyIfExists bool, onlyExported bool, args []core.Value, vm *core.VM) (core.Value, error) {
	m.Lock()
	p := m.plugins[path]

	if p == nil {
		// if the plugin is not loaded already, try to load it now
		var err error
		p, err = m.doLoadPlugin(path, vm)
		if err != nil {
			m.Unlock()
			if onlyIfExists && os.IsNotExist(err) {
				return core.NullValue, nil
			}
			return core.NullValue, fmt.Errorf("error loading plugin %s: %v", path, err)
		}

		if p == nil {
			m.Unlock()
			if onlyIfExists {
				return core.NullValue, nil
			}
			return core.NullValue, fmt.Errorf("plugin not loaded %s", path)
		}
	}
	m.Unlock()

	f, ok := p.program.Function(function)
	if !ok {
		if onlyIfExists {
			return core.NullValue, nil
		}
		return core.NullValue, fmt.Errorf("the function doesn't exist: %s", function)
	}

	if onlyExported && !f.Exported {
		return core.NullValue, fmt.Errorf("invalid function: %s is not exported", function)
	}

	return doExecFunction(c, f, p, args, vm)
}

func findFunc(name string, p *core.Program) *core.Function {
	f, ok := p.Function(name)
	if ok {
		return f
	}

	for _, v := range p.Functions {
		vName := v.Name
		i := strings.LastIndex(vName, "/")
		if i != -1 {
			vName = strings.Replace(vName[:i], "/", ".", -1) + "." + vName[i+1:]
		}

		if strings.HasSuffix(vName, name) {
			f = v
		}
	}

	return f
}

func doExecFunction(c *Context, f *core.Function, p *plugin, args []core.Value, vm *core.VM) (core.Value, error) {
	cvm := vm.Clone(p.program, p.globals)
	c = c.Clone()
	if c.DB != nil {
		db := c.DB.db
		origAnyNamespace := db.WriteAnyNamespace
		origOpenAnyDB := db.OpenAnyDatabase
		origNamespace := db.Namespace
		db.Namespace = strings.Replace(p.name, ".", ":", -1)
		defer func() {
			db.Namespace = origNamespace
			db.WriteAnyNamespace = origAnyNamespace
			db.OpenAnyDatabase = origOpenAnyDB
		}()
	}

	c.Plugin = p

	if HasProgramPermission(p.program, "trusted") {
		if c.DB != nil {
			c.DB.db.WriteAnyNamespace = true
			c.DB.db.OpenAnyDatabase = true
		}
	} else {
		if c.DB != nil {
			if HasProgramPermission(p.program, "c") {
				c.DB.db.WriteAnyNamespace = true
			}
			if HasProgramPermission(p.program, "openAnyDatabase") {
				c.DB.db.OpenAnyDatabase = true
			}
		}
	}

	cvm.Context = c

	v, err := cvm.RunFuncIndex(f.Index, args...)
	if err != nil {
		return core.NullValue, err
	}

	return v, nil
}

func (m *PluginManager) Clone(vm *core.VM) (*PluginManager, error) {
	clone := &PluginManager{
		fs:         m.fs,
		debug:      m.debug,
		pluginsDir: m.pluginsDir,
		plugins:    make(map[string]*plugin),
		hooks:      make(map[string][]HookFunction),
	}

	// initialize a new copy
	for path, p := range m.plugins {
		plg := &plugin{
			name:    p.name,
			program: p.program,
		}

		if err := clone.doLoadPluginProgram(plg, path, vm); err != nil {
			return nil, err
		}
	}

	return clone, nil
}

func (m *PluginManager) allPlugins(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	a := make([]core.Value, len(m.plugins))

	i := 0
	for _, v := range m.plugins {
		a[i] = core.NewObject(v)
		i++
	}

	return core.NewArrayValues(a), nil
}

func (m *PluginManager) getPlugin(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	path, err := m.getPluginPath(args[0].ToString())
	if err != nil {
		return core.NullValue, err
	}

	m.Lock()
	p := m.plugins[path]
	m.Unlock()

	if p == nil {
		return core.NullValue, nil
	}

	return core.NewObject(p), nil
}

func (m *PluginManager) loadPlugin(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", l)
	}

	a := args[0]
	switch a.Type {
	case core.Object:
		return m.loadObjectPlugin(a, vm)
	case core.String:
	default:
		return core.NullValue, fmt.Errorf("expected a string or a plugin, got %s", a.TypeName())
	}

	plugin := a.ToString()

	if !vm.HasPermission("trusted") {
		// only allow to load installed plugins
		c := GetContext(vm)
		if !hasPluginActive(c, plugin) {
			return core.NullValue, fmt.Errorf("unauthorized: the plugin is not installed: %s", plugin)
		}
	}

	path, err := m.getPluginPath(plugin)
	if err != nil {
		return core.NullValue, err
	}

	m.Lock()
	defer m.Unlock()
	p, err := m.doLoadPlugin(path, vm)
	if err != nil {
		return core.NullValue, fmt.Errorf("error loading plugin %s: %v", path, err)
	}

	return core.NewObject(p), nil
}

func (m *PluginManager) reloadPlugin(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	path, err := m.getPluginPath(name)
	if err != nil {
		return core.NullValue, err
	}

	if !vm.HasPermission("trusted") {
		// only allow to load installed plugins
		c := GetContext(vm)
		if !hasPluginActive(c, path) {
			return core.NullValue, ErrUnauthorized
		}
	}

	m.Lock()
	defer m.Unlock()

	delete(m.plugins, path)

	// remove the hook subscriptions (they will be added again in init)
	for k, hooks := range m.hooks {
		for i := len(hooks) - 1; i >= 0; i-- {
			if hooks[i].Plugin == name {
				hooks = append(hooks[:i], hooks[i+1:]...)
			}
		}
		m.hooks[k] = hooks
	}

	p, err := m.doLoadPlugin(path, vm)
	if err != nil {
		return core.NullValue, fmt.Errorf("error loading plugin %s: %v", path, err)
	}

	return core.NewObject(p), nil
}

func (m *PluginManager) loadObjectPlugin(a core.Value, vm *core.VM) (core.Value, error) {
	plg, ok := a.ToObjectOrNil().(*plugin)
	if !ok {
		return core.NullValue, fmt.Errorf("expected a plugin, got %s", a.TypeName())
	}

	path := fmt.Sprintf("%s.gt", plg.name)

	m.Lock()
	defer m.Unlock()
	if err := m.doLoadPluginProgram(plg, path, vm); err != nil {
		return core.NullValue, err
	}

	return core.NewObject(plg), nil
}

func (m *PluginManager) doLoadPlugin(path string, vm *core.VM) (*plugin, error) {
	pg, ok := m.plugins[path]
	if ok {
		return pg, nil
	}

	if m.fs == nil {
		return nil, fmt.Errorf("no filesystem")
	}

	var p *core.Program
	var err error

	if m.debug {
		src := filepath.Join("plugins", strings.TrimSuffix(path, ".gt"), "server", "main.ts")
		p, err = core.Compile(m.fs, src)
		if err != nil {
			return nil, err
		}

	} else {
		f, err := m.fs.Open(path)
		if err != nil {
			return nil, err
		}
		p, err = binary.Read(f)
		if err != nil {
			return nil, err
		}
		f.Close()
	}

	if err != nil {
		return nil, err
	}

	name, err := getPluginName(path)
	if err != nil {
		return nil, err
	}

	plg := &plugin{
		name:    name,
		program: p,
	}

	if err := m.doLoadPluginProgram(plg, path, vm); err != nil {
		return nil, err
	}

	return plg, nil
}

func (m *PluginManager) doLoadPluginProgram(p *plugin, path string, vm *core.VM) error {
	cvm := core.NewVM(p.program)
	c := GetContext(vm).Clone()
	c.PluginManager = m

	if c.DB != nil {
		db := c.DB.db
		origWriteAll := db.WriteAnyNamespace
		origOpenAnyDB := db.OpenAnyDatabase
		origNamespace := db.Namespace
		db.Namespace = strings.Replace(p.name, ".", ":", -1)
		defer func() {
			db.Namespace = origNamespace
			db.WriteAnyNamespace = origWriteAll
			db.OpenAnyDatabase = origOpenAnyDB
		}()
	}

	c.Plugin = p

	if HasProgramPermission(p.program, "trusted") {
		if c.DB != nil {
			c.DB.db.WriteAnyNamespace = true
		}
	} else {
		if c.DB != nil {
			if HasProgramPermission(p.program, "writeAnyDatabaseNamespace") {
				c.DB.db.WriteAnyNamespace = true
			}
			if HasProgramPermission(p.program, "openAnyDatabase") {
				c.DB.db.OpenAnyDatabase = true
			}
		}
	}

	cvm.Context = c
	cvm.MaxAllocations = vm.MaxAllocations
	cvm.MaxFrames = vm.MaxFrames
	cvm.MaxSteps = vm.MaxSteps

	cvm.AddSteps(vm.Steps())

	if err := cvm.Initialize(); err != nil {
		return fmt.Errorf(err.Error())
	}

	p.globals = cvm.Globals()
	m.plugins[path] = p

	return nil
}

func (m *PluginManager) clone(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	clone, err := m.Clone(vm)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewObject(clone), nil
}

func (m *PluginManager) clear(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateOptionalArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	m.hooks = make(map[string][]HookFunction)

	if len(args) == 0 {
		m.plugins = make(map[string]*plugin)
		m.hooks = make(map[string][]HookFunction)
	} else {
		path, err := m.getPluginPath(args[0].ToString())
		if err != nil {
			return core.NullValue, err
		}
		delete(m.plugins, path)
		delete(m.hooks, path)
	}

	return core.NullValue, nil
}

func newFinalizable(v core.Value, vm *core.VM) (finalizable, error) {
	switch v.Type {
	case core.Func:

	case core.NativeFunc:

	case core.Object:
		if _, ok := v.ToObject().(core.Closure); !ok {
			return finalizable{}, fmt.Errorf("expected a function, got: %s", v.TypeName())
		}

	default:
		return finalizable{}, fmt.Errorf("expected a function, got %v", v.TypeName())
	}

	f := finalizable{v: v, vm: vm}
	return f, nil
}

type finalizable struct {
	v  core.Value
	vm *core.VM
}

func (finalizable) Type() string {
	return "[Finalizable]"
}

func (f finalizable) Close() error {
	v := f.v
	vm := f.vm

	var lastErr = vm.Error
	vm.Error = nil
	switch v.Type {

	case core.NativeFunc:
		i := v.ToNativeFunction()
		f := core.NativeFuncFromIndex(i)
		if f.Arguments != 0 {
			return fmt.Errorf("function '%s' expects %d parameters", f.Name, f.Arguments)
		}
		_, err := f.Function(core.NullValue, nil, vm)
		return err

	case core.Func:
		i := v.ToFunction()
		if _, err := vm.RunFuncIndex(i); err != nil {
			return err
		}

	case core.Object:
		c, ok := v.ToObject().(core.Closure)
		if !ok {
			panic("should be a closure")
		}
		if _, err := vm.RunClosure(c); err != nil {
			return err
		}

	default:
		panic("should be a function or a closure")

	}

	vm.Error = lastErr
	return nil
}

func (f finalizable) GetMethod(name string) core.NativeMethod {
	switch name {
	case "close":
		return f.close
	}
	return nil
}

func (f finalizable) close(args []core.Value, vm *core.VM) (core.Value, error) {
	return core.NullValue, nil
}

type program struct {
	prog *core.Program
}

func (p *program) Type() string {
	return "runtime.Program"
}

// func (p *program) GetProperty(name string, vm *core.VM) (core.Value, error) {
// 	switch name {
// 	case "build":
// 		return core.String(p.prog.Build), nil
// 	}

// 	return core.Undefined(), nil
// }

// func (p *program) SetProperty(key string, v core.Value, vm *core.VM) error {
// 	if !vm.HasPermission("trusted") {
// 		return ErrUnauthorized
// 	}

// 	switch key {
// 	case "build":
// 		p.prog.Build = v.ToString()
// 		return nil
// 	}
// 	return ErrReadOnlyOrUndefined
// }

func (p *program) GetMethod(name string) core.NativeMethod {
	switch name {
	case "functions":
		return p.functions
	case "functionInfo":
		return p.functionInfo
	case "toString":
		return p.toString
	case "toBytes":
		return p.toBytes
	case "resources":
		return p.resources
	case "setResource":
		return p.setResource
	case "resource":
		return p.resource
	case "strip":
		return p.strip
	case "write":
		return p.write
	case "directives":
		return p.directives
	case "directive":
		return p.directive
	case "directiveValues":
		return p.directiveValues
	case "addDirective":
		return p.addDirective
	}
	return nil
}

func (p *program) directives(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	directives := p.prog.Directives

	ret := make(map[string]core.Value, len(directives))

	for k, v := range directives {
		ret[k] = core.NewString(v)
	}

	v := core.NewMapValues(ret)

	return v, nil
}

func (p *program) directive(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()
	v, ok := p.prog.Directives[name]

	if ok {
		v = strings.TrimPrefix(v, "\"")

		if strings.HasSuffix(v, "\"") {
			v = v[:len(v)-1]
		}
		return core.NewString(v), nil
	}

	return core.NullValue, nil
}

func (p *program) directiveValues(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()
	v, ok := p.prog.Directives[name]

	if ok {
		items := strings.Split(v, " ")
		result := make([]core.Value, len(items))
		for i, item := range items {
			result[i] = core.NewString(item)
		}
		return core.NewArrayValues(result), nil
	}

	return core.NullValue, nil
}

func (p *program) addDirective(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgs(args, core.String, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()
	value := args[1].ToString()

	p.prog.Directives[name] = value
	return core.NullValue, nil
}

func (p *program) write(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	w, ok := args[0].ToObjectOrNil().(io.Writer)
	if !ok {
		return core.NullValue, fmt.Errorf("exepected a Writer, got %s", args[0].TypeName())
	}

	if err := binary.Write(w, p.prog); err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (p *program) strip(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	p.prog.Strip()

	return core.NullValue, nil
}

func (p *program) setResource(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String, core.Bytes); err != nil {
		return core.NullValue, err
	}

	if p.prog.Resources == nil {
		p.prog.Resources = make(map[string][]byte)
	}

	p.prog.Resources[args[0].ToString()] = args[1].ToBytes()
	return core.NullValue, nil
}

func (p *program) resources(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	res := p.prog.Resources

	if res == nil {
		return core.NewArray(0), nil
	}

	a := make([]core.Value, len(res))

	i := 0
	for k := range res {
		a[i] = core.NewString(k)
		i++
	}

	return core.NewArrayValues(a), nil
}

func (p *program) resource(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	if p.prog.Resources == nil {
		return core.NullValue, nil
	}

	v, ok := p.prog.Resources[name]
	if !ok {
		return core.NullValue, nil
	}

	return core.NewBytes(v), nil
}

func (p *program) functions(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected no args")
	}

	var funcs []core.Value
	for _, f := range p.prog.Functions {
		fi := functionInfo{f, *p}
		funcs = append(funcs, core.NewObject(fi))
	}
	return core.NewArrayValues(funcs), nil
}

func (p *program) functionInfo(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	f, ok := p.prog.Function(name)
	if !ok {
		return core.NullValue, nil
	}

	return core.NewObject(functionInfo{f, *p}), nil
}

func (p *program) toBytes(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	w, ok := args[0].ToObject().(io.Writer)
	if !ok {
		return core.NullValue, fmt.Errorf("expected parameter 1 to be io.Writer, got %T", args[0].ToObject())
	}

	err := binary.Write(w, p.prog)
	return core.NullValue, err
}

func (p *program) toString(args []core.Value, vm *core.VM) (core.Value, error) {
	var b bytes.Buffer
	core.Fprint(&b, p.prog)
	return core.NewString(b.String()), nil
}

type functionInfo struct {
	fn *core.Function
	p  program
}

func (functionInfo) Type() string {
	return "runtime.FunctionInfo"
}

func (f functionInfo) GetMethod(name string) core.NativeMethod {
	switch name {
	case "toString":
		return f.toString
	}
	return nil
}

func (f functionInfo) toString(args []core.Value, vm *core.VM) (core.Value, error) {
	var b bytes.Buffer
	core.FprintFunction(&b, f.fn, f.p.prog)
	return core.NewString(b.String()), nil
}

func (f functionInfo) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "name":
		return core.NewString(f.fn.Name), nil
	case "arguments":
		return core.NewInt(f.fn.Arguments), nil
	case "index":
		return core.NewInt(f.fn.Index), nil
	case "exported":
		return core.NewBool(f.fn.Exported), nil
	case "func":
		return core.NewFunction(f.fn.Index), nil
	}
	return core.UndefinedValue, nil
}

type libVM struct {
	vm *core.VM
}

func (m *libVM) Type() string {
	return "runtime.VirtualMachine"
}

func (m *libVM) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "context":
		return core.NewObject(GetContext(m.vm)), nil
	case "error":
		e := m.vm.Error
		if e != nil {
			return core.NewObject(e), nil
		}
		return core.NullValue, nil
	case "program":
		if !vm.HasPermission("trusted") {
			return core.NullValue, ErrUnauthorized
		}
		return core.NewObject(&program{prog: m.vm.Program}), nil
	case "fileSystem":
		return core.NewObject(NewFileSystem(m.vm.FileSystem)), nil
	case "maxAllocations":
		return core.NewInt64(m.vm.MaxAllocations), nil
	case "maxFrames":
		return core.NewInt(m.vm.MaxFrames), nil
	case "maxSteps":
		return core.NewInt64(m.vm.MaxSteps), nil
	case "steps":
		return core.NewInt64(m.vm.Steps()), nil
	case "trusted":
		return core.NewBool(m.vm.Trusted), nil
	}
	return core.UndefinedValue, nil
}

func (m *libVM) SetProperty(name string, v core.Value, vm *core.VM) error {
	if !vm.HasPermission("trusted") {
		return ErrUnauthorized
	}

	switch name {
	case "trusted":
		if v.Type != core.Bool {
			return ErrInvalidType
		}
		m.vm.Trusted = v.ToBool()
		return nil

	case "error":
		switch v.Type {
		case core.Null:
			m.vm.Error = nil
			return nil

		case core.Object:
			e, ok := v.ToObject().(error)
			if !ok {
				return ErrInvalidType
			}
			m.vm.Error = e
			return nil

		default:
			return ErrInvalidType
		}

	case "context":
		c, ok := v.ToObjectOrNil().(*Context)
		if !ok {
			return ErrInvalidType
		}
		m.vm.Context = c
		return nil

	case "maxAllocations":
		if v.Type != core.Int {
			return ErrInvalidType
		}
		m.vm.MaxAllocations = v.ToInt()
		return nil

	case "maxFrames":
		if v.Type != core.Int {
			return ErrInvalidType
		}
		m.vm.MaxFrames = int(v.ToInt())
		return nil

	case "maxSteps":
		if v.Type != core.Int {
			return ErrInvalidType
		}
		m.vm.MaxSteps = v.ToInt()
		return nil
	}

	return ErrReadOnlyOrUndefined
}

func (m *libVM) GetMethod(name string) core.NativeMethod {
	switch name {
	case "initialize":
		return m.initialize
	case "run":
		return m.run
	case "runFunc":
		return m.runFunc
	case "runStaticFunc":
		return m.runStaticFunc
	case "clone":
		return m.clone
	case "getValue":
		return m.getValue
	case "getGlobals":
		return m.getGlobals
	case "getStackTrace":
		return m.getStackTrace
	case "setFileSystem":
		return m.setFileSystem
	case "getItem":
		return m.getItem
	case "setItem":
		return m.setItem
	case "resetSteps":
		return m.resetSteps
	}
	return nil
}

func (m *libVM) clone(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	c := m.vm.Clone(m.vm.Program, m.vm.Globals())
	return core.NewObject(&libVM{c}), nil
}

func (m *libVM) getItem(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}
	s := GetContext(m.vm).getProtectedItem(args[0].ToString())
	return s, nil
}

func (m *libVM) resetSteps(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	m.vm.ResetSteps()

	return core.NullValue, nil
}

func (m *libVM) setItem(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if len(args) != 2 {
		return core.NullValue, fmt.Errorf("expected 2 params, got %d", len(args))
	}

	a := args[0]
	if a.Type != core.String {
		return core.NullValue, fmt.Errorf("expected param 1 to be string, got %s", a.TypeName())
	}

	GetContext(m.vm).setProtectedItem(a.ToString(), args[1])

	return core.NullValue, nil
}

func (m *libVM) setFileSystem(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	fs, ok := args[0].ToObject().(*FileSystemObj)
	if !ok {
		return core.NullValue, fmt.Errorf("expected a fileSystem, got %s", args[0].TypeName())
	}
	m.vm.FileSystem = fs.FS
	return core.NullValue, nil
}

func (m *libVM) getStackTrace(args []core.Value, vm *core.VM) (core.Value, error) {
	s := strings.Join(m.vm.Stacktrace(), "\n")
	return core.NewString(s), nil
}

func (m *libVM) getGlobals(args []core.Value, vm *core.VM) (core.Value, error) {
	return core.NewArrayValues(m.vm.Globals()), nil
}

func (m *libVM) initialize(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := m.vm.Initialize(); err != nil {
		// return the error with the stacktrace included in the message
		// because the caller in the program will have it's own stacktrace.
		return core.NullValue, errors.New(err.Error())
	}
	return core.NullValue, nil
}

func (m *libVM) run(args []core.Value, vm *core.VM) (core.Value, error) {
	v, err := m.vm.Run(args...)
	if err != nil {
		// return the error with the stacktrace included in the message
		// because the caller in the program will have it's own stacktrace.
		return core.NullValue, errors.New(err.Error())
	}
	return v, nil
}

func (m *libVM) runStaticFunc(args []core.Value, vm *core.VM) (core.Value, error) {
	return m.runFunction(args, vm, false)
}

func (m *libVM) runFunc(args []core.Value, vm *core.VM) (core.Value, error) {
	return m.runFunction(args, vm, true)
}

func (m *libVM) runFunction(args []core.Value, vm *core.VM, initialize bool) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected at least 1 parameter, got %d", l)
	}

	var index int

	switch args[0].Type {
	case core.String:
		name := args[0].ToString()
		f, ok := m.vm.Program.Function(name)
		if !ok {
			return core.NullValue, fmt.Errorf("function %s not found", name)
		}
		index = f.Index
	case core.Int:
		index = int(args[0].ToInt())
		if len(m.vm.Program.Functions) <= index {
			return core.NullValue, fmt.Errorf("index out of range")
		}
	default:
		return core.NullValue, fmt.Errorf("argument 1 must be a string (function name), got %s", args[0].TypeName())
	}

	if initialize && !m.vm.Initialized() {
		if err := m.vm.Initialize(); err != nil {
			return core.NullValue, m.vm.WrapError(err)
		}
		if err := vm.AddSteps(m.vm.Steps()); err != nil {
			return core.NullValue, m.vm.WrapError(err)
		}
	}

	v, err := m.vm.RunFuncIndex(index, args[1:]...)
	if err != nil {
		// return the error with the stacktrace included in the message
		// because the caller in the program will have it's own stacktrace.
		return core.NullValue, m.vm.WrapError(err)
	}
	if err := vm.AddSteps(m.vm.Steps()); err != nil {
		return core.NullValue, m.vm.WrapError(err)
	}
	return v, nil
}

func (m *libVM) getValue(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l != 1 {
		return core.NullValue, fmt.Errorf("expected 1 parameter, got %d", l)
	}

	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("argument 1 must be a string (var name), got %s", args[0].TypeName())
	}

	name := args[0].ToString()
	v, _ := m.vm.RegisterValue(name)
	return v, nil
}

func GetContext(vm *core.VM) *Context {
	var c *Context

	if vm.Context == nil {
		c = &Context{}
		vm.Context = c
	} else {
		var ok bool
		c, ok = vm.Context.(*Context)
		if !ok {
			panic(fmt.Sprintf("Invalid Context type: %T", vm.Context))
		}
	}

	return c
}

type Context struct {
	sync.RWMutex
	GUID           string
	culture        culture
	UserCulture    string
	location       location
	Tenant         string
	TenantLabel    string
	TenantIcon     string
	Debug          bool
	MonoTenant     string
	Now            time.Time // to set the current time fixed
	Test           bool
	DB             *libDB
	Plugin         *plugin
	PluginManager  *PluginManager
	Caller         *plugin
	Plugins        []string
	Items          core.Value
	DataFS         *FileSystemObj
	ErrorLogger    *file
	protectedItems map[string]core.Value
}

func (c *Context) Type() string {
	return "Context"
}

func (c *Context) Clone() *Context {
	return &Context{
		GUID:           c.GUID,
		culture:        c.culture,
		UserCulture:    c.UserCulture,
		location:       c.location,
		Tenant:         c.Tenant,
		TenantLabel:    c.TenantLabel,
		TenantIcon:     c.TenantIcon,
		Debug:          c.Debug,
		MonoTenant:     c.MonoTenant,
		Now:            c.Now,
		Test:           c.Test,
		DB:             c.DB,
		Plugin:         c.Plugin,
		PluginManager:  c.PluginManager,
		Caller:         c.Plugin,
		Plugins:        c.Plugins,
		Items:          c.Items,
		protectedItems: c.protectedItems,
		DataFS:         c.DataFS,
		ErrorLogger:    c.ErrorLogger,
	}
}

func (c *Context) GetLocation() *time.Location {
	l := c.location
	if l.l != nil {
		return l.l
	}
	return time.Local
}

func (c *Context) GetCulture() culture {
	l := c.culture
	if l.culture.Name != "" {
		return culture{l.culture}
	}
	return culture{i18n.DefaultCulture}
}

func (c *Context) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "guid":
		return core.NewString(c.GUID), nil
	case "culture":
		cl := c.GetCulture()
		return core.NewObject(cl), nil
	case "userCulture":
		return core.NewString(c.UserCulture), nil
	case "location":
		return core.NewObject(location{c.GetLocation()}), nil
	case "tenant":
		return core.NewString(c.Tenant), nil
	case "tenantLabel":
		return core.NewString(c.TenantLabel), nil
	case "tenantIcon":
		return core.NewString(c.TenantIcon), nil
	case "debug":
		return core.NewBool(c.Debug), nil
	case "monoTenant":
		return core.NewString(c.MonoTenant), nil
	case "test":
		return core.NewBool(c.Test), nil
	case "db":
		if c.DB == nil {
			return core.NullValue, nil
		}
		return core.NewObject(c.DB), nil
	case "dataFS":
		return core.NewObject(c.DataFS), nil
	case "items":
		if c.Items.Type == core.Null {
			c.Items = core.NewMap(0)
		}
		return c.Items, nil
	case "plugin":
		if c.Plugin != nil {
			return core.NewObject(c.Plugin), nil
		}
		return core.NullValue, nil
	case "pluginManager":
		pm := c.getPluginManager()
		if pm != nil {
			return core.NewObject(pm), nil
		}
		return core.NullValue, nil
	case "caller":
		if c.Caller != nil {
			return core.NewObject(c.Caller), nil
		}
		return core.NullValue, nil
	case "pluginName":
		if c.Plugin != nil {
			return core.NewString(c.Plugin.name), nil
		}
		return core.NewString(""), nil
	case "errorLogger":
		if c.ErrorLogger == nil {
			return core.NullValue, nil
		}
		return core.NewObject(c.ErrorLogger), nil
	}
	return core.UndefinedValue, nil
}

func (c *Context) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "guid":
		if v.Type != core.String {
			return fmt.Errorf("invalid value: %s", v.TypeName())
		}
		c.GUID = v.String()
		return nil

	case "culture":
		l, ok := v.ToObjectOrNil().(culture)
		if !ok {
			return fmt.Errorf("invalid value: %s", v.TypeName())
		}
		c.culture = l
		return nil

	case "userCulture":
		if v.Type != core.String {
			return fmt.Errorf("invalid value: %s", v.TypeName())
		}
		c.UserCulture = v.String()
		return nil

	case "location":
		l, ok := v.ToObjectOrNil().(location)
		if !ok || l.l == nil {
			return fmt.Errorf("invalid value: %s", v.TypeName())
		}
		c.location = l
		return nil

	case "tenant":
		if !vm.HasPermission("trusted") {
			return ErrUnauthorized
		}
		if v.Type != core.String {
			return fmt.Errorf("invalid value: %s", v.TypeName())
		}
		c.Tenant = v.ToString()
		return nil

	case "tenantLabel":
		if !vm.HasPermission("trusted") {
			return ErrUnauthorized
		}
		if v.Type != core.String {
			if v.IsNil() {
				c.TenantLabel = ""
				return nil
			}
			return fmt.Errorf("invalid value: %s", v.TypeName())
		}
		c.TenantLabel = v.ToString()
		return nil

	case "tenantIcon":
		if !vm.HasPermission("trusted") {
			return ErrUnauthorized
		}
		if v.Type != core.String {
			if v.IsNil() {
				c.TenantIcon = ""
				return nil
			}
			return fmt.Errorf("invalid value: %s", v.TypeName())
		}
		c.TenantIcon = v.ToString()
		return nil

	case "debug":
		if !vm.HasPermission("trusted") {
			return ErrUnauthorized
		}
		switch v.Type {
		case core.Null, core.Undefined:
			return nil
		case core.Bool:
			c.Debug = v.ToBool()
			return nil
		default:
			return fmt.Errorf("invalid value: %s", v.TypeName())
		}

	case "monoTenant":
		if !vm.HasPermission("trusted") {
			return ErrUnauthorized
		}
		switch v.Type {
		case core.String:
			c.MonoTenant = v.ToString()
			return nil
		case core.Undefined, core.Null:
			c.MonoTenant = ""
			return nil
		default:
			return ErrInvalidType
		}

	case "test":
		if !vm.HasPermission("trusted") {
			return ErrUnauthorized
		}
		switch v.Type {
		case core.Null, core.Undefined:
			return nil
		case core.Bool:
			c.Test = v.ToBool()
			return nil
		default:
			return fmt.Errorf("invalid value: %s", v.TypeName())
		}

	case "db":
		// if !vm.HasPermission("trusted") {
		// 	return ErrUnauthorized
		// }
		db, ok := v.ToObjectOrNil().(*libDB)
		if !ok {
			return fmt.Errorf("invalid value: %s", v.TypeName())
		}
		c.DB = db
		return nil

	case "dataFS":
		if !vm.HasPermission("trusted") {
			return ErrUnauthorized
		}
		fs, ok := v.ToObjectOrNil().(*FileSystemObj)
		if !ok {
			return fmt.Errorf("invalid value: %s", v.TypeName())
		}
		c.DataFS = fs
		return nil

	case "items":
		switch v.Type {
		case core.Map:
		default:
			return errors.New("invalid type. Expected an object (map)")
		}
		c.Items = v
		return nil

	case "plugin":
		if !vm.HasPermission("trusted") {
			return ErrUnauthorized
		}
		p, ok := v.ToObjectOrNil().(*plugin)
		if !ok {
			return fmt.Errorf("invalid value: %s", v.TypeName())
		}
		c.Plugin = p
		return nil

	case "pluginManager":
		if !vm.HasPermission("trusted") {
			return ErrUnauthorized
		}
		p, ok := v.ToObjectOrNil().(*PluginManager)
		if !ok {
			return fmt.Errorf("invalid value: %s", v.TypeName())
		}
		c.PluginManager = p
		return nil

	case "errorLogger":
		if !vm.HasPermission("trusted") {
			return ErrUnauthorized
		}
		f, ok := v.ToObjectOrNil().(*file)
		if !ok {
			return fmt.Errorf("expected a file, got %s", v.TypeName())
		}
		c.ErrorLogger = f
		return nil

	default:
		return ErrReadOnlyOrUndefined
	}
}

func (c *Context) GetMethod(name string) core.NativeMethod {
	switch name {
	case "clone":
		return c.clone
	case "exec":
		return c.exec
	case "execIfExists":
		return c.execIfExists
	case "addPlugin":
		return c.addPlugin
	case "hasPlugin":
		return c.hasPlugin
	case "getPlugins":
		return c.getPlugins
	case "setPlugins":
		return c.setPlugins
	}
	return nil
}

func (c *Context) getProtectedItem(name string) core.Value {
	if c.protectedItems == nil {
		return core.NewString("")
	}

	c.RLock()
	v := c.protectedItems[name]
	c.RUnlock()

	if v == core.NullValue {
		return core.NewString("")
	}

	return v
}

func (c *Context) setProtectedItem(name string, v core.Value) {
	c.Lock()
	if c.protectedItems == nil {
		c.protectedItems = make(map[string]core.Value)
	}
	c.protectedItems[name] = v
	c.Unlock()
}

func (c *Context) clone(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	return core.NewObject(c.Clone()), nil
}

func (c *Context) hasPlugin(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	for _, p := range c.Plugins {
		if p == name {
			return core.TrueValue, nil
		}
	}

	return core.FalseValue, nil
}

func (c *Context) getPlugins(args []core.Value, vm *core.VM) (core.Value, error) {
	ps := make([]core.Value, len(c.Plugins))
	for i, p := range c.Plugins {
		ps[i] = core.NewString(p)
	}
	return core.NewArrayValues(ps), nil
}

func (c *Context) setPlugins(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgRange(args, 1, 1); err != nil {
		return core.NullValue, err
	}

	v := args[0]

	if v.IsNil() {
		c.Plugins = nil
		return core.NullValue, nil
	}
	if v.Type != core.Array {
		return core.NullValue, ErrInvalidType
	}

	a := v.ToArray()
	c.Plugins = make([]string, len(a))

	for i, j := range a {
		if j.Type != core.String {
			return core.NullValue, fmt.Errorf("invalid value in index %d, expected string, got %s", i, j.TypeName())
		}
		c.Plugins[i] = j.ToString()
	}
	return core.NullValue, nil
}

func (c *Context) addPlugin(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	p := args[0].ToString()

	for _, v := range c.Plugins {
		if v == p {
			return core.NullValue, nil
		}
	}

	c.Plugins = append(c.Plugins, p)
	return core.NullValue, nil
}

func (c *Context) getPluginManager() *PluginManager {
	if c.PluginManager != nil {
		return c.PluginManager
	}

	var pm *PluginManager
	mut.RLock()
	pm = globalPluginManager
	mut.RUnlock()

	return pm
}

func (c *Context) exec(args []core.Value, vm *core.VM) (core.Value, error) {
	ln := len(args)
	if ln < 1 {
		return core.NullValue, fmt.Errorf("expected at least 1 args, got %d", ln)
	}

	pm := c.getPluginManager()
	if pm == nil {
		return core.NullValue, fmt.Errorf("there is no plugin manager set")
	}

	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected arg 1 to be a string, got %s", args[0].TypeName())
	}

	return pm.execPlugin(c, args[0].ToString(), args[1:], false, vm)
}

func (c *Context) execIfExists(args []core.Value, vm *core.VM) (core.Value, error) {
	ln := len(args)
	if ln < 1 {
		return core.NullValue, fmt.Errorf("expected at least 1 args, got %d", ln)
	}

	pm := c.getPluginManager()
	if pm == nil {
		return core.NullValue, fmt.Errorf("there is no plugin manager set")
	}

	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected arg 1 to be a string, got %s", args[0].TypeName())
	}

	return pm.execPlugin(c, args[0].ToString(), args[1:], true, vm)
}
