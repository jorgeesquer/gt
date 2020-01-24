package lib

import (
	"image"
	"image/png"
	"io"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(Png, `

declare namespace png {

    export function encode(w: io.Writer, img: Image): void

    export function decode(buf: byte[] | io.Reader): Image

    export interface Image { }
}


`)
}

var Png = []core.NativeFunction{
	core.NativeFunction{
		Name:      "png.decode",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Object); err != nil {
				return core.NullValue, err
			}

			r, ok := args[0].ToObject().(io.Reader)
			if !ok {
				return core.NullValue, ErrInvalidType
			}

			img, err := png.Decode(r)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(imageObj{img}), nil
		},
	},
	core.NativeFunction{
		Name:      "png.encode",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Object, core.Object); err != nil {
				return core.NullValue, err
			}

			w, ok := args[0].ToObject().(io.Writer)
			if !ok {
				return core.NullValue, ErrInvalidType
			}

			i, ok := args[1].ToObject().(imageObj)
			if !ok {
				return core.NullValue, ErrInvalidType
			}

			err := png.Encode(w, i.img)
			if err != nil {
				return core.NullValue, err
			}

			return core.NullValue, nil
		},
	},
}

type imageObj struct {
	img image.Image
}

func (i imageObj) Type() string {
	return "image"
}
