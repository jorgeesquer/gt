package lib

import (
	"io"
	"github.com/gtlang/gt/core"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

func init() {
	core.RegisterLib(Encoding, `
	
declare namespace encoding {
    export interface Decoder {
        reader(r: io.Reader): io.Reader
    }
    export interface Encoder {
        writer(r: io.Writer): io.Writer
    }

    export function newDecoderISO8859_1(): Decoder
    export function newEncoderISO8859_1(): Encoder
    export function newDecoderWindows1252(): Decoder
    export function newEncoderWindows1252(): Encoder
}

`)
}

var Encoding = []core.NativeFunction{
	core.NativeFunction{
		Name:      "encoding.newDecoderISO8859_1",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			d := charmap.ISO8859_1.NewDecoder()
			return core.NewObject(&decoder{d}), nil
		},
	},
	core.NativeFunction{
		Name:      "encoding.newEncoderISO8859_1",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			d := charmap.ISO8859_1.NewEncoder()
			return core.NewObject(&encoder{d}), nil
		},
	},
	core.NativeFunction{
		Name:      "encoding.newDecoderWindows1252",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			d := charmap.Windows1252.NewDecoder()
			return core.NewObject(&decoder{d}), nil
		},
	},
	core.NativeFunction{
		Name:      "encoding.newEncoderWindows1252",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			d := charmap.Windows1252.NewEncoder()
			return core.NewObject(&encoder{d}), nil
		},
	},
}

type decoder struct {
	d *encoding.Decoder
}

func (d *decoder) Type() string {
	return "encoding.Decoder"
}

func (d *decoder) GetMethod(name string) core.NativeMethod {
	switch name {
	case "reader":
		return d.reader
	}
	return nil
}

func (d *decoder) reader(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	r, ok := args[0].ToObject().(io.Reader)
	if !ok {
		return core.NullValue, ErrInvalidType
	}

	rd := &reader{d.d.Reader(r)}

	return core.NewObject(rd), nil
}

type encoder struct {
	d *encoding.Encoder
}

func (d *encoder) Type() string {
	return "encoding.Encoder"
}

func (d *encoder) GetMethod(name string) core.NativeMethod {
	switch name {
	case "writer":
		return d.writer
	}
	return nil
}

func (d *encoder) writer(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	w, ok := args[0].ToObject().(io.Writer)
	if !ok {
		return core.NullValue, ErrInvalidType
	}

	rd := &writer{d.d.Writer(w)}

	return core.NewObject(rd), nil
}
