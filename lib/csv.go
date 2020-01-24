package lib

import (
	"encoding/csv"
	"fmt"
	"io"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(CSV, `

declare namespace csv {
    export function newReader(r: io.Reader): Reader
    export interface Reader {
        comma: string
        read(): string[]
    }

    export function newWriter(r: io.Writer): Writer
    export interface Writer {
        comma: string
        write(v: (string | number)[]): void
        flush(): void
    }
}


`)
}

var CSV = []core.NativeFunction{
	core.NativeFunction{
		Name:      "csv.newReader",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Object); err != nil {
				return core.NullValue, err
			}

			r, ok := args[0].ToObject().(io.Reader)
			if !ok {
				return core.NullValue, ErrInvalidType
			}

			reader := csv.NewReader(r)

			return core.NewObject(&csvReader{reader}), nil
		},
	},
	core.NativeFunction{
		Name:      "csv.newWriter",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Object); err != nil {
				return core.NullValue, err
			}

			w, ok := args[0].ToObject().(io.Writer)
			if !ok {
				return core.NullValue, ErrInvalidType
			}

			writer := csv.NewWriter(w)

			return core.NewObject(&csvWriter{writer}), nil
		},
	},
}

type csvReader struct {
	r *csv.Reader
}

func (r *csvReader) Type() string {
	return "csv.Reader"
}

func (r *csvReader) GetProperty(key string, vm *core.VM) (core.Value, error) {
	switch key {
	case "comma":
		return core.NewString(string(r.r.Comma)), nil
	}
	return core.UndefinedValue, nil
}

func (r *csvReader) SetProperty(key string, v core.Value, vm *core.VM) error {
	switch key {
	case "comma":
		if v.Type != core.String {
			return ErrInvalidType
		}
		s := v.ToString()
		if len(s) != 1 {
			return fmt.Errorf("invalid comma: %s", s)
		}
		r.r.Comma = rune(s[0])
		return nil
	}
	return ErrReadOnlyOrUndefined
}

func (r *csvReader) GetMethod(name string) core.NativeMethod {
	switch name {
	case "read":
		return r.read
	}
	return nil
}

func (r *csvReader) read(args []core.Value, vm *core.VM) (core.Value, error) {
	records, err := r.r.Read()
	if err != nil {
		if err == io.EOF {
			return core.NullValue, nil
		}
		return core.NullValue, err
	}

	result := make([]core.Value, len(records))
	for i, v := range records {
		result[i] = core.NewString(v)
	}

	return core.NewArrayValues(result), nil
}

type csvWriter struct {
	w *csv.Writer
}

func (*csvWriter) Type() string {
	return "csv.Writer"
}

func (w *csvWriter) GetProperty(key string, vm *core.VM) (core.Value, error) {
	switch key {
	case "comma":
		return core.NewString(string(w.w.Comma)), nil
	}
	return core.UndefinedValue, nil
}

func (w *csvWriter) SetProperty(key string, v core.Value, vm *core.VM) error {
	switch key {
	case "comma":
		if v.Type != core.String {
			return ErrInvalidType
		}
		s := v.ToString()
		if len(s) != 1 {
			return fmt.Errorf("invalid comma: %s", s)
		}
		w.w.Comma = rune(s[0])
		return nil
	}
	return ErrReadOnlyOrUndefined
}

func (w *csvWriter) GetMethod(name string) core.NativeMethod {
	switch name {
	case "write":
		return w.write
	case "flush":
		return w.flush
	}
	return nil
}

func (w *csvWriter) write(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Array); err != nil {
		return core.NullValue, err
	}

	a := args[0].ToArray()

	values := make([]string, len(a))

	for i, v := range a {
		values[i] = v.ToString()
	}

	if err := w.w.Write(values); err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (w *csvWriter) flush(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	w.w.Flush()

	if err := w.w.Error(); err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}
