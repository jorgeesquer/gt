package lib

import (
	"bufio"
	"fmt"
	"io"

	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(Bufio, `
	
declare namespace bufio {
    export function newWriter(w: io.Writer): Writer
    export function newScanner(r: io.Reader): Scanner
    export function newReader(r: io.Reader): Reader

    export interface Writer {
        write(data: byte[]): number
        writeString(s: string): number
        writeByte(b: byte): void
        writeRune(s: string): number
        flush(): void
    }

    export interface Scanner {
        scan(): boolean 
        text(): string
    }

    export interface Reader {
        readString(delim: byte): string
        readBytes(delim: byte): byte[]
        readByte(): byte
        readRune(): number
    }
}

`)
}

var Bufio = []core.NativeFunction{
	core.NativeFunction{
		Name:      "bufio.newScanner",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Object); err != nil {
				return core.NullValue, err
			}

			r := args[0].ToObject()

			reader, ok := r.(io.Reader)
			if !ok {
				return core.NullValue, fmt.Errorf("expected a io.Reader, got %v", args[0])
			}

			s := bufio.NewScanner(reader)

			return core.NewObject(&scanner{s}), nil
		},
	},
	core.NativeFunction{
		Name:      "bufio.newReader",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Object); err != nil {
				return core.NullValue, err
			}

			r := args[0].ToObject()

			reader, ok := r.(io.Reader)
			if !ok {
				return core.NullValue, fmt.Errorf("expected a io.Reader, got %v", args[0])
			}

			s := bufio.NewReader(reader)

			return core.NewObject(&bufioReader{s}), nil
		},
	},
	core.NativeFunction{
		Name:      "bufio.newWriter",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Object); err != nil {
				return core.NullValue, err
			}

			r := args[0].ToObject()

			w, ok := r.(io.Writer)
			if !ok {
				return core.NullValue, fmt.Errorf("expected a io.Writer, got %v", args[0])
			}

			s := bufio.NewWriter(w)

			return core.NewObject(&bufioWriter{s}), nil
		},
	},
}

type bufioWriter struct {
	w *bufio.Writer
}

func (*bufioWriter) Type() string {
	return "bufio.Writer"
}

func (w *bufioWriter) GetMethod(name string) core.NativeMethod {
	switch name {
	case "write":
		return w.write
	case "writeString":
		return w.writeString
	case "writeByte":
		return w.writeByte
	case "writeRune":
		return w.writeRune
	case "flush":
		return w.flush
	}
	return nil
}

func (w *bufioWriter) writeString(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	n, err := w.w.WriteString(args[0].ToString())
	if err != nil {
		return core.NullValue, err
	}

	return core.NewInt(n), nil
}

func (w *bufioWriter) write(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Bytes); err != nil {
		return core.NullValue, err
	}

	n, err := w.w.Write(args[0].ToBytes())
	if err != nil {
		return core.NullValue, err
	}

	return core.NewInt(n), nil
}

func (w *bufioWriter) flush(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}
	err := w.w.Flush()
	if err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (w *bufioWriter) writeByte(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Int); err != nil {
		return core.NullValue, err
	}

	err := w.w.WriteByte(byte(args[0].ToInt()))
	if err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (w *bufioWriter) writeRune(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgRange(args, 1, 1); err != nil {
		return core.NullValue, err
	}

	var r rune
	switch args[0].Type {
	case core.Rune, core.Int:
		r = args[0].ToRune()
	default:
		return core.NullValue, fmt.Errorf("expected rune, got %v", args[0].TypeName())
	}

	n, err := w.w.WriteRune(r)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewInt(n), nil
}

type bufioReader struct {
	s *bufio.Reader
}

func (*bufioReader) Type() string {
	return "bufio.Reader"
}

func (s *bufioReader) GetMethod(name string) core.NativeMethod {
	switch name {
	case "readString":
		return s.readString
	case "readRune":
		return s.readRune
	case "readByte":
		return s.readByte
	case "readBytes":
		return s.readBytes
	}
	return nil
}

func (s *bufioReader) readRune(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	r, _, err := s.s.ReadRune()
	if err != nil {
		if err == io.EOF {
			return core.NullValue, nil
		}
		return core.NullValue, err
	}

	return core.NewRune(r), nil
}

func (s *bufioReader) readByte(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	b, err := s.s.ReadByte()
	if err != nil {
		if err == io.EOF {
			return core.NullValue, nil
		}
		return core.NullValue, err
	}

	return core.NewInt(int(b)), nil
}

func (s *bufioReader) readBytes(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	delim := args[0].ToString()
	if len(delim) != 1 {
		return core.NullValue, fmt.Errorf("invalid delimiter lenght. Must be a byte: %v", delim)
	}

	v, err := s.s.ReadBytes(delim[0])

	if err != nil {
		if err == io.EOF {
			return core.NullValue, nil
		}
		return core.NullValue, err
	}

	return core.NewBytes(v), nil
}

func (s *bufioReader) readString(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	delim := args[0].ToString()
	if len(delim) != 1 {
		return core.NullValue, fmt.Errorf("invalid delimiter lenght. Must be a byte: %v", delim)
	}

	v, err := s.s.ReadString(delim[0])

	if err != nil {
		if err == io.EOF {
			return core.NullValue, nil
		}
		return core.NullValue, err
	}

	return core.NewString(v), nil
}

type scanner struct {
	s *bufio.Scanner
}

func (s *scanner) Type() string {
	return "bufio.Scanner"
}

func (s *scanner) GetMethod(name string) core.NativeMethod {
	switch name {
	case "scan":
		return s.scan
	case "text":
		return s.text
	}
	return nil
}

func (s *scanner) text(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	v := s.s.Text()

	if err := s.s.Err(); err != nil {
		return core.NullValue, err
	}

	return core.NewString(v), nil
}

func (s *scanner) scan(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	v := s.s.Scan()

	if err := s.s.Err(); err != nil {
		return core.NullValue, err
	}

	return core.NewBool(v), nil
}
