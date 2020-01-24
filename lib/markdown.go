package lib

import (
	"math/rand"
	"github.com/gtlang/gt/core"
	"time"

	"github.com/russross/blackfriday"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	core.RegisterLib(Markdown, `

declare namespace markdown {

    export function toHTML(n: string | byte[]): string
}

`)
}

var Markdown = []core.NativeFunction{
	core.NativeFunction{
		Name:      "markdown.toHTML",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}
			b := args[0].ToBytes()
			out := blackfriday.MarkdownCommon(b)
			return core.NewString(string(out)), nil
		},
	},
}
