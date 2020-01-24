package lib

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"github.com/gtlang/filesystem"
	"github.com/gtlang/gt/core"
	"strconv"
	"strings"

	"github.com/gtlang/gt/lib/x/templates"
)

func init() {
	core.RegisterLib(Templates, `


declare namespace templates {
    /**
     * Reads the file and processes includes
     */
    export function exec(code: string, model?: any): string
    export function preprocess(path: string, fs?: io.FileSystem): string
    export function render(text: string, model?: any): string
    export function renderHTML(text: string, model?: any): string
    /**
     * 
     * @param headerFunc By defauult is: function render(w: io.Writer, model: any)
     */
    export function compile(text: string, headerFunc?: string): string
    /**
     * 
     * @param headerFunc By defauult is: function render(w: io.Writer, model: any)
     */
    export function compileHTML(text: string, headerFunc?: string): string

    /**
     * 
     * @param headerFunc By defauult is: function render(w: io.Writer, model: any)
     */
    export function writeHTML(w: io.Writer, path: string, model?: any, fs?: io.FileSystem, headerFunc?: string): void

    /**
     * 
     * @param headerFunc By defauult is: function render(w: io.Writer, model: any)
     */
    export function writeHTMLTemplate(w: io.Writer, template: string, model?: any, headerFunc?: string): void
}

`)
}

var includesRegex = regexp.MustCompile(`<!-- include "(.*?)" -->`)

var Templates = []core.NativeFunction{
	core.NativeFunction{
		Name:      "templates.exec",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			var buf []byte
			var model core.Value

			l := len(args)
			if l == 0 || l > 2 {
				return core.NullValue, fmt.Errorf("expected one or two arguments, got %d", l)
			}

			a := args[0]
			switch a.Type {
			case core.String, core.Bytes:
				buf = a.ToBytes()
			default:
				return core.NullValue, ErrInvalidType
			}

			if l == 2 {
				model = args[1]
			}

			return execTemplate(buf, nil, model, vm)
		},
	},
	core.NativeFunction{
		Name:      "templates.render",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return render(false, this, args, vm)
		},
	},
	core.NativeFunction{
		Name:      "templates.renderHTML",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return render(true, this, args, vm)
		},
	},
	core.NativeFunction{
		Name:      "templates.writeHTML",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return writeHTML(true, this, args, vm)
		},
	},
	core.NativeFunction{
		Name:      "templates.writeHTMLTemplate",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return writeHTMLTemplate(true, this, args, vm)
		},
	},
	core.NativeFunction{
		Name:      "templates.compile",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return compileTemplate(false, this, args, vm)
		},
	},
	core.NativeFunction{
		Name:      "templates.compileHTML",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return compileTemplate(true, this, args, vm)
		},
	},
	core.NativeFunction{
		Name:      "templates.preprocess",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			var path string
			var fs filesystem.FS

			l := len(args)
			switch l {
			case 1:
				path = args[0].ToString()
				fs = vm.FileSystem
			case 2:
				path = args[0].ToString()
				fo, ok := args[1].ToObjectOrNil().(*FileSystemObj)
				if !ok {
					return core.NullValue, fmt.Errorf("expected a fileSystem, got %s", args[0].TypeName())
				}
				fs = fo.FS
			default:
				return core.NullValue, fmt.Errorf("expected one or two arguments, got %d", l)
			}

			buf, err := readFile(path, fs, vm)
			if err != nil {
				if os.IsNotExist(err) {
					return core.NullValue, nil
				}
				return core.NullValue, fmt.Errorf("error reading template '%s':_ %v", path, err)
			}

			includes := includesRegex.FindAllSubmatchIndex(buf, -1)
			for i := len(includes) - 1; i >= 0; i-- {
				loc := includes[i]
				start := loc[0]
				end := loc[1]
				include := string(buf[loc[2]:loc[3]])
				b, err := readFile(include, fs, vm)
				if err != nil {
					if os.IsNotExist(err) {
						// try the path relative to the template dir
						localPath := filepath.Join(filepath.Dir(path), include)
						b, err = readFile(localPath, fs, vm)
						if err != nil {
							return core.NullValue, fmt.Errorf("error reading include '%s':_ %v", include, err)
						}
					} else {
						return core.NullValue, fmt.Errorf("error reading include '%s':_ %v", include, err)
					}
				}
				buf = append(buf[:start], append(b, buf[end:]...)...)
			}

			return core.NewString(string(buf)), nil
		},
	},
}

func readFile(path string, fs filesystem.FS, vm *core.VM) ([]byte, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}

	b, err := ReadAll(f, vm)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func compileTemplate(html bool, this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
	var funcWrapper string
	var code string

	l := len(args)
	switch l {
	case 1:
		code = args[0].ToString()
		funcWrapper = "function render(w: io.Writer, model: any)"
	case 2:
		code = args[0].ToString()
		funcWrapper = args[1].ToString()
	default:
		return core.NullValue, fmt.Errorf("expected one or two arguments, got %d", l)
	}

	var b []byte
	var sourceMap []int
	var err error

	if html {
		b, sourceMap, err = templates.CompileHtml(code, funcWrapper)
	} else {
		b, sourceMap, err = templates.Compile(code, funcWrapper)
	}

	if err != nil {
		return core.NullValue, mapError(err, sourceMap)
	}

	return core.NewString(string(b)), nil
}

func writeHTML(html bool, this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
	ln := len(args)

	if ln < 2 || ln > 5 {
		return core.NullValue, fmt.Errorf("expected at 2, 3 or 5 arguments, got %d", ln)
	}

	if _, ok := args[0].ToObjectOrNil().(io.Writer); !ok {
		return core.NullValue, fmt.Errorf("expected arg 1 to be a io.Writer, got %s", args[0].TypeName())
	}

	vPath := args[1]
	if vPath.Type != core.String {
		return core.NullValue, fmt.Errorf("expected arg 2 to be a string, got %s", vPath.TypeName())
	}

	var model core.Value
	if ln > 2 {
		model = args[2]
	}

	var fs filesystem.FS
	if ln == 4 {
		vFS, ok := args[3].ToObjectOrNil().(*FileSystemObj)
		if !ok {
			return core.NullValue, fmt.Errorf("expected arg 4 to be a io.FileSystem, got %s", args[3].TypeName())
		}
		fs = vFS.FS
	} else {
		fs = vm.FileSystem
	}

	if fs == nil {
		return core.NullValue, fmt.Errorf("there is no filesystem")
	}

	src, err := filesystem.ReadAll(fs, vPath.ToString())
	if err != nil {
		return core.NullValue, err
	}

	var headerFunc string
	if ln == 5 {
		h := args[4]
		if h.Type != core.String {
			return core.NullValue, fmt.Errorf("expected arg 5 to be a string, got %s", h.TypeName())
		}
		headerFunc = h.ToString()
	} else {
		headerFunc = "function render(w: io.Writer, model: any)"
	}

	var sourceMap []int
	var b []byte
	if html {
		b, sourceMap, err = templates.CompileHtml(string(src), headerFunc)
	} else {
		b, sourceMap, err = templates.Compile(string(src), headerFunc)
	}

	if err != nil {
		return core.NullValue, mapError(err, sourceMap)
	}

	return write(args[0], b, sourceMap, model, vm)
}

func writeHTMLTemplate(html bool, this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
	ln := len(args)

	if ln < 2 || ln > 4 {
		return core.NullValue, fmt.Errorf("expected at 2, 3 or 4 arguments, got %d", ln)
	}

	if _, ok := args[0].ToObjectOrNil().(io.Writer); !ok {
		return core.NullValue, fmt.Errorf("expected arg 1 to be a io.Writer, got %s", args[0].TypeName())
	}

	template := args[1]
	if template.Type != core.String {
		return core.NullValue, fmt.Errorf("expected arg 2 to be a string, got %s", template.TypeName())
	}

	var model core.Value
	if ln > 2 {
		model = args[2]
	}

	var headerFunc string
	if ln == 4 {
		h := args[3]
		if h.Type != core.String {
			return core.NullValue, fmt.Errorf("expected arg 5 to be a string, got %s", h.TypeName())
		}
		headerFunc = h.ToString()
	} else {
		headerFunc = "function render(w: io.Writer, model: any)"
	}

	var sourceMap []int
	var b []byte
	var err error
	if html {
		b, sourceMap, err = templates.CompileHtml(template.ToString(), headerFunc)
	} else {
		b, sourceMap, err = templates.Compile(template.ToString(), headerFunc)
	}

	if err != nil {
		return core.NullValue, mapError(err, sourceMap)
	}

	return write(args[0], b, sourceMap, model, vm)
}

func render(html bool, this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
	var template core.Value
	var model core.Value

	l := len(args)
	switch l {
	case 0:
		return core.NullValue, fmt.Errorf("expected at least one argument, got %d", l)
	case 1:
		template = args[0]
		switch template.Type {
		case core.String, core.Bytes:
		default:
			return core.NullValue, ErrInvalidType
		}
	case 2:
		template = args[0]
		switch template.Type {
		case core.String, core.Bytes:
		default:
			return core.NullValue, ErrInvalidType
		}
		model = args[1]
	default:
		return core.NullValue, fmt.Errorf("expected one or two arguments, got %d", l)
	}

	var b []byte
	var sourceMap []int
	var err error

	if html {
		b, sourceMap, err = templates.CompileHtml(template.ToString(), "function render(w: io.Writer, model: any)")
	} else {
		b, sourceMap, err = templates.Compile(template.ToString(), "function render(w: io.Writer, model: any)")
	}

	if err != nil {
		return core.NullValue, errors.New(err.Error())
	}

	return execTemplate(b, sourceMap, model, vm)
}

func getVM(b []byte, vm *core.VM) (*core.VM, error) {
	p, err := core.CompileStr(string(b))
	if err != nil {
		return nil, err
	}

	m := core.NewVM(p)

	m.MaxAllocations = 10000000000 // vm.MaxAllocations
	m.MaxFrames = 10               // vm.MaxFrames
	m.MaxSteps = 10000000          // vm.MaxSteps
	m.Context = vm.Context
	// m.AddSteps(vm.Steps())

	return m, nil
}

func write(w core.Value, b []byte, sourceMap []int, model core.Value, vm *core.VM) (core.Value, error) {
	m, err := getVM(b, vm)
	if err != nil {
		return core.NullValue, mapError(err, sourceMap)
	}

	if _, err := m.RunFunc("render", w, model); err != nil {
		// return the error with the stacktrace included in the message
		// because the caller in the program will have it's own stacktrace.
		return core.NullValue, mapError(err, sourceMap)
	}

	if err := vm.AddSteps(m.Steps() - vm.Steps()); err != nil {
		return core.NullValue, mapError(err, sourceMap)
	}

	return core.NullValue, nil
}

func execTemplate(b []byte, sourceMap []int, model core.Value, vm *core.VM) (core.Value, error) {
	m, err := getVM(b, vm)
	if err != nil {
		return core.NullValue, mapError(err, sourceMap)
	}

	buf := &buffer{buf: &bytes.Buffer{}}

	if _, err := m.RunFunc("render", core.NewObject(buf), model); err != nil {
		// return the error with the stacktrace included in the message
		// because the caller in the program will have it's own stacktrace.
		return core.NullValue, mapError(err, sourceMap)
	}

	if err := vm.AddSteps(m.Steps() - vm.Steps()); err != nil {
		return core.NullValue, mapError(err, sourceMap)
	}

	return core.NewString(buf.buf.String()), nil
}

func mapError(e error, sourceMap []int) error {
	ln := len(sourceMap)

	if ln == 0 {
		return e
	}

	var lines []string
	r := strings.NewReader(e.Error())
	s := bufio.NewScanner(r)

	for s.Scan() {
		line := s.Text()
		if !strings.HasPrefix(line, " -> ") {
			lines = append(lines, line)
			continue
		}

		i := strings.LastIndexByte(line, ':')
		if i == -1 {
			i = strings.LastIndex(line, " line ")
			if i == -1 {
				lines = append(lines, line)
				continue
			}
			i += 5
		}

		n, err := strconv.Atoi(line[i+1:])
		if err != nil {
			return fmt.Errorf("error mapping error: %v. Original Errror: %v", err, e)
		}

		// error lines are reported in base 1 and the map is also in base 0
		n -= 2

		if n > ln {
			return fmt.Errorf("error mapping error: %v. Original Errror: %v", err, e)
		}

		// now show the mapped line in base 1 again
		m := sourceMap[n] + 1

		line = line[:i] + ":" + strconv.Itoa(m)
		lines = append(lines, line)
	}

	if err := s.Err(); err != nil {
		return fmt.Errorf("error mapping error: %v. Original Errror: %v", err, e)
	}

	sErr := strings.Join(lines, "\n")

	return fmt.Errorf("template error: %s", sErr)
}

type buffer struct {
	buf *bytes.Buffer
}

func (b buffer) Type() string {
	return "templates.Buffer"
}

func (b buffer) GetMethod(name string) core.NativeMethod {
	switch name {
	case "write":
		return b.write
	}
	return nil
}

func (b buffer) write(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	v := args[0]

	switch v.Type {
	case core.Null, core.Undefined:
		return core.NullValue, nil
	case core.String:
		b.buf.WriteString(v.ToString())
	case core.Bytes:
		b.buf.Write(v.ToBytes())
	case core.Int:
		fmt.Fprintf(b.buf, "%d", v.ToInt())
	case core.Float:
		fmt.Fprintf(b.buf, "%f", v.ToFloat())
	case core.Array:
		b.buf.WriteString("[array]")
	case core.Object:
		b.buf.WriteString("[object]")
	case core.Map:
		b.buf.WriteString("[object]")
	default:
		return core.NullValue, ErrInvalidType
	}

	return core.NullValue, nil
}
