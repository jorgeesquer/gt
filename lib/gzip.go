package lib

import (
	"compress/gzip"
	"fmt"
	"io"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(GZIP, `

declare namespace gzip {
    export function newWriter(w: io.Writer): io.WriterCloser
    export function newReader(r: io.Reader): io.ReaderCloser
}


`)
}

var GZIP = []core.NativeFunction{
	core.NativeFunction{
		Name:      "gzip.newWriter",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			w, ok := args[0].ToObjectOrNil().(io.Writer)
			if !ok {
				return core.NullValue, fmt.Errorf("exepected a Writer, got %s", args[0].TypeName())
			}

			g := gzip.NewWriter(w)
			v := &gzipWriter{g}
			return core.NewObject(v), nil
		},
	},
	core.NativeFunction{
		Name:      "gzip.newReader",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			r, ok := args[0].ToObjectOrNil().(io.Reader)
			if !ok {
				return core.NullValue, fmt.Errorf("exepected a reader, got %s", args[0].TypeName())
			}

			gr, err := gzip.NewReader(r)
			if err != nil {
				return core.NullValue, err
			}
			v := &gzipReader{gr}
			return core.NewObject(v), nil
		},
	},
}

type gzipWriter struct {
	w *gzip.Writer
}

func (*gzipWriter) Type() string {
	return "gzip.Writer"
}

func (w *gzipWriter) GetMethod(name string) core.NativeMethod {
	switch name {
	case "write":
		return w.write
	case "close":
		return w.close
	}
	return nil
}

func (w *gzipWriter) Write(p []byte) (n int, err error) {
	return w.w.Write(p)
}

func (w *gzipWriter) write(args []core.Value, vm *core.VM) (core.Value, error) {
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

func (w *gzipWriter) close(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	err := w.w.Close()
	if err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

type gzipReader struct {
	r *gzip.Reader
}

func (*gzipReader) Type() string {
	return "gzip.Reader"
}

func (r *gzipReader) GetMethod(name string) core.NativeMethod {
	switch name {
	case "read":
		return r.read
	case "close":
		return r.close
	}
	return nil
}

func (r *gzipReader) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func (r *gzipReader) read(args []core.Value, vm *core.VM) (core.Value, error) {
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

func (r *gzipReader) close(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	err := r.r.Close()
	if err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}
