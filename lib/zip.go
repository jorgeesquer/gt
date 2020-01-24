package lib

import (
	"archive/zip"
	"fmt"
	"io"
	"github.com/gtlang/filesystem"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(ZIP, `


declare namespace zip {
    export function newWriter(w: io.Writer): Writer
    export function newReader(r: io.Reader, size: number): io.ReaderCloser
    export function open(path: string, fs?: io.FileSystem): Reader

    export interface Writer {
        create(name: string): io.Writer
        flush(): void
        close(): void
    }

    export interface Reader {
        files(): File[]
        close(): void
    }

    export interface File {
        name: string
        compressedSize: number
        uncompressedSize: number
        open(): io.ReaderCloser
    }
}


`)
}

var ZIP = []core.NativeFunction{
	core.NativeFunction{
		Name:      "zip.newWriter",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			w, ok := args[0].ToObjectOrNil().(io.Writer)
			if !ok {
				return core.NullValue, fmt.Errorf("exepected a Writer, got %s", args[0].TypeName())
			}

			g := zip.NewWriter(w)
			v := &zipWriter{g}
			return core.NewObject(v), nil
		},
	},
	core.NativeFunction{
		Name:      "zip.newReader",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Object, core.Int); err != nil {
				return core.NullValue, err
			}

			r, ok := args[0].ToObjectOrNil().(io.ReaderAt)
			if !ok {
				return core.NullValue, fmt.Errorf("exepected a reader, got %s", args[0].TypeName())
			}

			size := args[1].ToInt()

			gr, err := zip.NewReader(r, size)
			if err != nil {
				return core.NullValue, err
			}

			v := &zipReader{r: gr}
			return core.NewObject(v), nil
		},
	},
	core.NativeFunction{
		Name:      "zip.open",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateOptionalArgs(args, core.String, core.Object); err != nil {
				return core.NullValue, err
			}

			if err := ValidateArgRange(args, 1, 2); err != nil {
				return core.NullValue, err
			}

			var fs filesystem.FS
			if len(args) == 2 {
				fsObj, ok := args[1].ToObjectOrNil().(*FileSystemObj)
				if !ok {
					return core.NullValue, fmt.Errorf("exepected a FileSystem, got %s", args[1].TypeName())
				}
				fs = fsObj.FS
			} else {
				fs = vm.FileSystem
			}

			f, err := fs.Open(args[0].ToString())
			if err != nil {
				return core.NullValue, err
			}

			fi, err := f.Stat()
			if err != nil {
				return core.NullValue, err
			}

			size := fi.Size()

			gr, err := zip.NewReader(f, size)
			if err != nil {
				return core.NullValue, err
			}

			v := &zipReader{gr, f}
			return core.NewObject(v), nil
		},
	},
}

type zipWriter struct {
	w *zip.Writer
}

func (*zipWriter) Type() string {
	return "zip.Writer"
}

func (w *zipWriter) GetMethod(name string) core.NativeMethod {
	switch name {
	case "create":
		return w.create
	case "flush":
		return w.flush
	case "close":
		return w.close
	}
	return nil
}

func (w *zipWriter) flush(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	if err := w.w.Flush(); err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (w *zipWriter) create(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	f, err := w.w.Create(name)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewObject(&writer{f}), nil
}

func (w *zipWriter) close(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	err := w.w.Close()
	if err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

type zipReader struct {
	r *zip.Reader
	c io.Closer
}

func (*zipReader) Type() string {
	return "zip.Reader"
}
func (r *zipReader) Close() error {
	if r.c == nil {
		return nil
	}
	return r.c.Close()
}

func (r *zipReader) GetMethod(name string) core.NativeMethod {
	switch name {
	case "files":
		return r.files
	case "close":
		return r.close
	}
	return nil
}

func (r *zipReader) close(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	if err := r.Close(); err != nil {
		return core.NullValue, err
	}
	return core.NullValue, nil
}

func (r *zipReader) files(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	files := r.r.File
	a := make([]core.Value, len(files))

	for i, f := range files {
		a[i] = core.NewObject(&zipFile{f})
	}
	return core.NewArrayValues(a), nil
}

type zipFile struct {
	f *zip.File
}

func (*zipFile) Type() string {
	return "zip.File"
}

func (f *zipFile) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "name":
		return core.NewString(f.f.Name), nil
	case "compressedSize":
		return core.NewInt64(int64(f.f.CompressedSize64)), nil
	case "uncompressedSize":
		return core.NewInt64(int64(f.f.UncompressedSize64)), nil
	}

	return core.UndefinedValue, nil
}

func (f *zipFile) GetMethod(name string) core.NativeMethod {
	switch name {
	case "open":
		return f.open
	}
	return nil
}

func (f *zipFile) open(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	rc, err := f.f.Open()
	if err != nil {
		return core.NullValue, err
	}

	return core.NewObject(&readerCloser{rc}), nil
}
