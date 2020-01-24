package lib

import (
	"crypto/md5"
	"crypto/sha256"
	"hash"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(HASH, `

declare namespace hash {
    export function newMD5(): Hash
    export function newSHA256(): Hash

    export interface Hash {
        write(b: byte[]): number
        sum(): byte[]
    }
}


`)
}

var HASH = []core.NativeFunction{
	core.NativeFunction{
		Name: "hash.newMD5",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			hash := md5.New()
			return core.NewObject(hasher{hash}), nil
		},
	},
	core.NativeFunction{
		Name: "hash.newSHA256",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			hash := sha256.New()
			return core.NewObject(hasher{hash}), nil
		},
	},
}

type hasher struct {
	h hash.Hash
}

func (hasher) Type() string {
	return "hash.Hash"
}

func (h hasher) GetMethod(name string) core.NativeMethod {
	switch name {
	case "write":
		return h.write
	case "sum":
		return h.sum
	}
	return nil
}

func (h hasher) Write(p []byte) (n int, err error) {
	return h.h.Write(p)
}

func (h hasher) write(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Bytes); err != nil {
		return core.NullValue, err
	}

	buf := args[0].ToBytes()

	n, err := h.h.Write(buf)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewInt(n), nil
}

func (h hasher) sum(args []core.Value, vm *core.VM) (core.Value, error) {
	b := h.h.Sum(nil)
	return core.NewBytes(b), nil
}
