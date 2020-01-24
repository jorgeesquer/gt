package core

import (
	"strings"
)

var allNativeFuncs []NativeFunction
var allNativeMap map[string]NativeFunction = make(map[string]NativeFunction)
var typeDefs = []string{header}

type NativeFunction struct {
	Name      string
	Arguments int
	Index     int
	Function  func(this Value, args []Value, vm *VM) (Value, error)
}

type NativeMethod func(args []Value, vm *VM) (Value, error)

func (NativeMethod) Type() string {
	return "[native method]"
}

type nativePrototype struct {
	this Value
	fn   int
}

func (nativePrototype) Type() string {
	return "[native prototype]"
}

func AddNativeFunc(f NativeFunction) {
	// replace if it already exists
	if existingFunc, ok := allNativeMap[f.Name]; ok {
		f.Index = existingFunc.Index
		allNativeMap[f.Name] = f
		return
	}

	f.Index = len(allNativeFuncs)
	allNativeFuncs = append(allNativeFuncs, f)
	allNativeMap[f.Name] = f
}

func RegisterLib(funcs []NativeFunction, dts string) {
	for _, f := range funcs {
		AddNativeFunc(f)
	}

	if dts != "" {
		typeDefs = append(typeDefs, dts)
	}
}

func NativeFuncFromIndex(i int) NativeFunction {
	return allNativeFuncs[i]
}

func NativeFuncFromName(name string) (NativeFunction, bool) {
	f, ok := allNativeMap[name]
	return f, ok
}

func All() []NativeFunction {
	return allNativeFuncs
}

func TypeDefs() string {
	return strings.Join(typeDefs, "\n\n")
}

const header = `/**
 * ------------------------------------------------------------------
 * GT Native type definitions.
 * ------------------------------------------------------------------
 */

// for the ts compiler
interface Boolean { }
interface Function { }
interface IArguments { }
interface Number { }
interface Object { }
interface RegExp { }
interface byte { }

declare const int: any
declare const float: any
declare const Array: any

interface Array<T> {
    [n: number]: T
    slice(start?: number, count?: number): Array<T>
    range(start?: number, end?: number): Array<T>
    append(v: T[]): T[]
    push(...v: T[]): void
    pushRange(v: T[]): void
    length: number
    insertAt(i: number, v: T): void
    removeAt(i: number): void
    removeAt(from: number, to: number): void
    indexOf(v: T): number
    join(sep: string): T
    sort(comprarer: (a: T, b: T) => boolean): void
}

// translate a value.
declare function T(key: string, ...params: any[]): string

declare namespace errors {
    export function wrap(msg: string, inner: Error): Error
    export function public(msg: string, inner?: Error | string): Error

    export interface Error {
        type: string
        public: boolean
        message: string
        pc: number
        stackTrace: string
        toString(): string
    }
}

`
