package lib

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"github.com/gtlang/filesystem"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(IO, `

declare namespace io {
    export interface Reader {
        read(b: byte[]): number
    }

    export interface ReaderAt {
		ReadAt(p: byte[], off: number): number
    }
	
    export interface ReaderCloser extends Reader {
        close(): void
    }

    export interface Writer {
        write(v: string | byte[]): number | void
    }

    export interface WriterCloser extends Writer {
        close(): void
    }

    export function copy(dst: Writer, src: Reader): number

    export function newVirtualFS(): FileSystem

    export function newRootedFS(root: string, baseFS: FileSystem): FileSystem

    /** 
     * Sets the default data file system that will be returned by io.dataFS()
     */
    export function setDataFS(fs: FileSystem): void

    export function newBuffer(): Buffer

    export interface Buffer {
        length: number
        cap: number
        read(b: byte[]): number
        write(v: any): void
        toString(): string
        toBytes(): byte[]
    }

    export interface FileSystem {
        /**
         * The current working directory
         */
        workingDir: string

        abs(path: string): string
        open(path: string): File
        openIfExists(path: string): File
        openForWrite(path: string): File
        openForAppend(path: string): File
        chdir(dir: string): void
        exists(path: string): boolean
        rename(source: string, dest: string): void
        removeAll(path: string): void
        readAll(path: string): byte[]
        readAllIfExists(path: string): byte[]
        readString(path: string): string
        readStringIfExists(path: string): string
        write(path: string, data: string | io.Reader | byte[]): void
        append(path: string, data: string | byte[]): void
        mkdir(path: string): void
        stat(path: string): FileInfo
        readDir(path: string): FileInfo[]
        readNames(path: string, recursive?: boolean): string[]
    }

    export interface File {
        read(b: byte[]): number
        write(v: string | byte[] | io.Reader): number
        writeAt(v: string | byte[] | io.Reader, offset: number): number
        close(): void
    }

    export interface FileInfo {
        name: string
        modTime: time.Time
        isDir: boolean
        size: number
    }
}
`)
}

var IO = []core.NativeFunction{
	core.NativeFunction{
		Name:      "io.newBuffer",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewObject(NewBuffer()), nil
		},
	},
	core.NativeFunction{
		Name:      "io.copy",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Object, core.Object); err != nil {
				return core.NullValue, err
			}

			dst, ok := args[0].ToObject().(io.Writer)
			if !ok {
				return core.NullValue, ErrInvalidType
			}

			src, ok := args[1].ToObject().(io.Reader)
			if !ok {
				return core.NullValue, ErrInvalidType
			}

			i, err := io.Copy(dst, src)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewInt64(i), nil
		},
	},
	core.NativeFunction{
		Name:      "io.newRootedFS",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String, core.Object); err != nil {
				return core.NullValue, err
			}
			fs, ok := args[1].ToObject().(*FileSystemObj)
			if !ok {
				return core.NullValue, fmt.Errorf("invalid filesystem argument, got %v", args[1])
			}
			root := args[0].ToString()
			rFS, err := filesystem.NewRootedFS(root, fs.FS)
			if err != nil {
				return core.NullValue, err
			}
			return core.NewObject(NewFileSystem(rFS)), nil
		},
	},
	// core.NativeFunc{
	// 	Name:      "io.newVirtualFS",
	// 	Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
	// 		if err := ValidateArgs(args, core.StringType, core.ObjectType); err != nil {
	// 			return core.NullValue, err
	// 		}

	// 		fs filesystem.NewVirtualFS()
	// 		rFS, err := filesystem.NewRootedFS(root, fs.FS)
	// 		if err != nil {
	// 			return core.NullValue, err
	// 		}
	// 		return core.Object(NewFileSystem(rFS)), nil
	// 	},
	// },
}

type readerCloser struct {
	r io.ReadCloser
}

func (r *readerCloser) Type() string {
	return "io.Reader"
}

func (r *readerCloser) GetMethod(name string) core.NativeMethod {
	switch name {
	case "read":
		return r.read
	case "close":
		return r.close
	}
	return nil
}

func (r *readerCloser) Close() error {
	return r.r.Close()
}

func (r *readerCloser) close(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	err := r.r.Close()
	if err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (r *readerCloser) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func (r *readerCloser) read(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Bytes); err != nil {
		return core.NullValue, err
	}

	buf := args[0].ToBytes()

	n, err := r.r.Read(buf)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewInt(n), nil
}

func NewReader(r io.Reader) *reader {
	return &reader{r}
}

type reader struct {
	r io.Reader
}

func (r *reader) Type() string {
	return "io.Reader"
}

func (r *reader) GetMethod(name string) core.NativeMethod {
	switch name {
	case "read":
		return r.read
	}
	return nil
}

func (r *reader) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func (r *reader) read(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Bytes); err != nil {
		return core.NullValue, err
	}

	buf := args[0].ToBytes()

	n, err := r.r.Read(buf)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewInt(n), nil
}

func NewWriter(w io.Writer) *writer {
	return &writer{w}
}

type writer struct {
	w io.Writer
}

func (*writer) Type() string {
	return "io.Writer"
}

func (w *writer) GetMethod(name string) core.NativeMethod {
	switch name {
	case "write":
		return w.write
	}
	return nil
}

func (w *writer) Write(p []byte) (n int, err error) {
	return w.w.Write(p)
}

func (w *writer) write(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Bytes); err != nil {
		return core.NullValue, err
	}

	buf := args[0].ToBytes()

	n, err := w.w.Write(buf)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewInt(n), nil
}

func NewBuffer() Buffer {
	var b bytes.Buffer
	return Buffer{&b}
}

type Buffer struct {
	Buf *bytes.Buffer
}

func (b Buffer) Type() string {
	return "io.Buffer"
}

func (b Buffer) Read(p []byte) (n int, err error) {
	return b.Buf.Read(p)
}

func (b Buffer) Write(p []byte) (n int, err error) {
	return b.Buf.Write(p)
}

func (b Buffer) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "length":
		return core.NewInt(b.Buf.Len()), nil
	case "cap":
		return core.NewInt(b.Buf.Cap()), nil
	}

	return core.UndefinedValue, nil
}

func (b Buffer) GetMethod(name string) core.NativeMethod {
	switch name {
	case "write":
		return b.write
	case "toString":
		return b.toString
	case "toBytes":
		return b.toBytes
	}
	return nil
}

func (b Buffer) write(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	if err := Write(b.Buf, args[0], vm); err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func WriteAt(w io.WriterAt, v core.Value, off int64, vm *core.VM) error {
	var d []byte

	switch v.Type {
	case core.Null, core.Undefined:
		return nil
	case core.String, core.Bytes:
		d = v.ToBytes()
	case core.Int:
		i := v.ToInt()
		if i < 0 || i > 255 {
			return fmt.Errorf("invalid byte value %d", i)
		}
		d = []byte{byte(i)}
	case core.Array:
		a := v.ToArray()
		d = make([]byte, len(a))
		for i, b := range a {
			switch b.Type {
			case core.Int:
				x := b.ToInt()
				if x < 0 || x > 255 {
					return fmt.Errorf("invalid byte value %d at %d", x, i)
				}
				d[i] = byte(x)
			}
		}
	case core.Object:
		r, ok := v.ToObject().(io.Reader)
		if !ok {
			return ErrInvalidType
		}
		var err error
		d, err = ioutil.ReadAll(r)
		if err != nil {
			return err
		}
	default:
		return ErrInvalidType
	}

	if err := vm.AddAllocations(len(d)); err != nil {
		return err
	}
	_, err := w.WriteAt(d, off)
	return err
}

func Write(w io.Writer, v core.Value, vm *core.VM) error {
	var d []byte

	switch v.Type {
	case core.Null, core.Undefined:
		return nil
	case core.String, core.Bytes:
		d = v.ToBytes()
	case core.Int:
		i := v.ToInt()
		if i < 0 || i > 255 {
			return fmt.Errorf("invalid byte value %d", i)
		}
		d = []byte{byte(i)}
	case core.Array:
		a := v.ToArray()
		d = make([]byte, len(a))
		for i, b := range a {
			switch b.Type {
			case core.Int:
				x := b.ToInt()
				if x < 0 || x > 255 {
					return fmt.Errorf("invalid byte value %d at %d", x, i)
				}
				d[i] = byte(x)
			}
		}
	case core.Object:
		r, ok := v.ToObject().(io.Reader)
		if !ok {
			return ErrInvalidType
		}
		// we are not worrying here about allocations.
		// TODO: Attack vector?? Make it safe??
		if _, err := io.Copy(w, r); err != nil {
			return err
		}
	default:
		return ErrInvalidType
	}

	if err := vm.AddAllocations(len(d)); err != nil {
		return err
	}

	_, err := w.Write(d)
	return err
}

func (b Buffer) toString(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 argument, got %d", len(args))
	}
	return core.NewString(b.Buf.String()), nil
}

func (b Buffer) toBytes(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 argument, got %d", len(args))
	}
	return core.NewBytes(b.Buf.Bytes()), nil
}
