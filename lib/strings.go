package lib

import (
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gtlang/gt/core"

	"golang.org/x/exp/utf8string"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	core.RegisterLib(Strings, `
	
declare namespace strings {
    export function newReader(a: string): io.Reader
}

interface String {
    runeAt(i: number): string
}

declare namespace strings {
    export function equalFold(a: string, b: string): boolean
    export function isChar(value: string): boolean
    export function isDigit(value: string): boolean
    export function isIdent(value: string): boolean
    export function isAlphanumeric(value: string): boolean
    export function isAlphanumericIdent(value: string): boolean
    export function isNumeric(value: string): boolean
    export function sort(a: string[]): void
}
	  
interface String {
    [n: number]: string 

    /**
     * Gets the length of the string.
     */
    length: number

    /**
     * The number of bytes oposed to the number of runes returned by length.
     */
    runeCount: number

    toLower(): string

    toUpper(): string

    toTitle(): string

    toUntitle(): string

    replace(oldValue: string, newValue: string, times?: number): string

    hasPrefix(prefix: string): boolean
    hasSuffix(prefix: string): boolean

    trim(cutset?: string): string
    trimLeft(cutset?: string): string
    trimRight(cutset?: string): string
    trimPrefix(prefix: string): string
    trimSuffix(suffix: string): string

    rightPad(pad: string, total: number): string
    leftPad(pad: string, total: number): string

    take(to: number): string
    substring(from: number, to?: number): string
    runeSubstring(from: number, to?: number): string

    split(s: string): string[]
    splitEx(s: string): string[]

    contains(s: string): boolean
    equalFold(s: string): boolean

    indexOf(s: string, start?: number): number
    lastIndexOf(s: string, start?: number): number


	/**
	 * Replace with regular expression.
	 * The syntax is defined: https://golang.org/pkg/regexp/syntax
	 */
    replaceRegex(expr: string, replace: string): string
}

	`)
}

var Strings = []core.NativeFunction{
	core.NativeFunction{
		Name:      "strings.newReader",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			r := strings.NewReader(args[0].ToString())

			return core.NewObject(&reader{r}), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.runeAt",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			if a.Type != core.Int {
				return core.NullValue, fmt.Errorf("expected int, got %s", a.Type)
			}

			i := int(a.ToInt())

			if i < 0 {
				return core.NullValue, vm.NewError("Index out of range in string")
			}

			// TODO: prevent this in every call
			v := utf8string.NewString(this.ToString())

			if int(i) >= v.RuneCount() {
				return core.NullValue, vm.NewError("Index out of range in string")
			}

			return core.NewRune(v.At(int(i))), nil
		},
	},
	core.NativeFunction{
		Name:      "strings.equalFold",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			b := args[1]

			switch a.Type {
			case core.Null, core.Undefined:
				switch b.Type {
				case core.Null, core.Undefined:
					return core.TrueValue, nil
				case core.String:
					return core.FalseValue, nil
				default:
					return core.NullValue, fmt.Errorf("expected argument 2 to be string got %v", b.Type)
				}
			case core.String:
			default:
				return core.NullValue, fmt.Errorf("expected argument 1 to be string got %v", a.Type)
			}

			switch b.Type {
			case core.Null, core.Undefined:
				// a cant be null at this point
				return core.FalseValue, nil
			case core.String:
			default:
				return core.NullValue, fmt.Errorf("expected argument 2 to be string got %v", b.Type)
			}

			eq := strings.EqualFold(a.ToString(), b.ToString())
			return core.NewBool(eq), nil
		},
	},
	core.NativeFunction{
		Name:      "strings.isIdent",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
			}

			b := IsIdent(args[0].ToString())
			return core.NewBool(b), nil
		},
	},
	core.NativeFunction{
		Name:      "strings.isAlphanumeric",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
			}

			b := IsAlphanumeric(args[0].ToString())
			return core.NewBool(b), nil
		},
	},
	core.NativeFunction{
		Name:      "strings.isAlphanumericIdent",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
			}

			b := IsAlphanumericIdent(args[0].ToString())
			return core.NewBool(b), nil
		},
	},
	core.NativeFunction{
		Name:      "strings.isNumeric",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
			}

			b := IsNumeric(args[0].String())
			return core.NewBool(b), nil
		},
	},
	core.NativeFunction{
		Name:      "strings.isChar",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
			}

			s := args[0].ToString()
			if len(s) != 1 {
				return core.FalseValue, nil
			}

			r := rune(s[0])
			if 'A' <= r && r <= 'Z' || 'a' <= r && r <= 'z' {
				return core.TrueValue, nil
			}

			return core.FalseValue, nil
		},
	},
	core.NativeFunction{
		Name:      "strings.isDigit",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
			}

			s := args[0].ToString()
			if len(s) != 1 {
				return core.FalseValue, nil
			}

			r := rune(s[0])
			if '0' <= r && r <= '9' {
				return core.TrueValue, nil
			}

			return core.FalseValue, nil
		},
	},
	core.NativeFunction{
		Name:      "strings.sort",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.Array {
				return core.NullValue, fmt.Errorf("expected arg 1 to be array, got %s", args[0].TypeName())
			}

			a := args[0].ToArray()

			s := make([]string, len(a))

			for i, v := range a {
				s[i] = v.ToString()
			}

			sort.Strings(s)

			for i, v := range s {
				a[i] = core.NewString(v)
			}

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.replaceRegex",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			exp := args[0].ToString()
			repl := args[1].ToString()
			s := this.ToString()
			r, err := regexp.Compile(exp)
			if err != nil {
				return core.NullValue, err
			}
			return core.NewString(r.ReplaceAllString(s, repl)), nil
		},
	},
	core.NativeFunction{
		Name: "String.prototype.toLower",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := this.ToString()
			return core.NewString(strings.ToLower(s)), nil
		},
	},
	core.NativeFunction{
		Name: "String.prototype.toUpper",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := this.ToString()
			return core.NewString(strings.ToUpper(s)), nil
		},
	},
	core.NativeFunction{
		Name: "String.prototype.toTitle",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := this.ToString()
			if len(s) > 0 {
				s = strings.ToUpper(s[:1]) + s[1:]
			}
			return core.NewString(s), nil
		},
	},
	core.NativeFunction{
		Name: "String.prototype.toUntitle",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := this.ToString()
			if len(s) > 0 {
				s = strings.ToLower(s[:1]) + s[1:]
			}
			return core.NewString(s), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.replace",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			l := len(args)
			if l < 2 || l > 3 {
				return core.NullValue, fmt.Errorf("expected 2 or 3 arguments, got %d", len(args))
			}

			oldStr := args[0].ToString()
			newStr := args[1].ToString()

			times := -1
			if l > 2 {
				times = int(args[2].ToInt())
			}

			s := this.ToString()
			return core.NewString(strings.Replace(s, oldStr, newStr, times)), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.split",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			sep := args[0].ToString()

			s := this.ToString()

			parts := Split(s, sep)
			res := make([]core.Value, len(parts))

			for i, v := range parts {
				res[i] = core.NewString(v)
			}
			return core.NewArrayValues(res), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.splitEx",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			sep := args[0].ToString()

			s := this.ToString()

			parts := strings.Split(s, sep)
			res := make([]core.Value, len(parts))

			for i, v := range parts {
				res[i] = core.NewString(v)
			}
			return core.NewArrayValues(res), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.trim",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			var cutset string
			switch len(args) {
			case 0:
				cutset = " \t\r\n"
			case 1:
				cutset = args[0].ToString()
			default:
				return core.NullValue, fmt.Errorf("expected 0 or 1 arguments, got %d", len(args))
			}
			s := this.ToString()
			return core.NewString(strings.Trim(s, cutset)), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.trimLeft",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			var cutset string
			switch len(args) {
			case 0:
				cutset = " \t\r\n"
			case 1:
				cutset = args[0].ToString()
			default:
				return core.NullValue, fmt.Errorf("expected 0 or 1 arguments, got %d", len(args))
			}
			s := this.ToString()
			return core.NewString(strings.TrimLeft(s, cutset)), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.trimRight",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			var cutset string
			switch len(args) {
			case 0:
				cutset = " \t\r\n"
			case 1:
				cutset = args[0].ToString()
			default:
				return core.NullValue, fmt.Errorf("expected 0 or 1 arguments, got %d", len(args))
			}
			s := this.ToString()
			return core.NewString(strings.TrimRight(s, cutset)), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.trimPrefix",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
			}
			s := this.ToString()
			prefix := args[0].ToString()
			s = strings.TrimPrefix(s, prefix)
			return core.NewString(s), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.trimSuffix",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
			}
			s := this.ToString()
			prefix := args[0].ToString()
			s = strings.TrimSuffix(s, prefix)
			return core.NewString(s), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.substring",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := this.ToString()

			switch len(args) {
			case 1:
				v1 := args[0]
				if v1.Type != core.Int {
					return core.NullValue, fmt.Errorf("expected int, got %s", v1.Type)
				}
				a := int(v1.ToInt())
				return core.NewString(s[a:]), nil
			case 2:
				v1 := args[0]
				if v1.Type != core.Int {
					return core.NullValue, fmt.Errorf("expected int, got %s", v1.Type)
				}
				v2 := args[1]
				if v2.Type != core.Int {
					return core.NullValue, fmt.Errorf("expected int, got %s", v2.Type)
				}
				l := len(s)
				a := int(v1.ToInt())
				b := int(v2.ToInt())
				if a < 0 || a > l {
					return core.NullValue, fmt.Errorf("start out of range")
				}
				if b < a || b > l {
					return core.NullValue, fmt.Errorf("end out of range")
				}
				return core.NewString(s[a:b]), nil
			}

			return core.NullValue, fmt.Errorf("expected 1 or 2 parameters")
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.runeSubstring",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := this.ToString()

			switch len(args) {
			case 1:
				v1 := args[0]
				if v1.Type != core.Int {
					return core.NullValue, fmt.Errorf("expected int, got %s", v1.Type)
				}
				a := int(v1.ToInt())
				return core.NewString(substring(s, a, -1)), nil
			case 2:
				v1 := args[0]
				if v1.Type != core.Int {
					return core.NullValue, fmt.Errorf("expected int, got %s", v1.Type)
				}
				v2 := args[1]
				if v2.Type != core.Int {
					return core.NullValue, fmt.Errorf("expected int, got %s", v2.Type)
				}
				l := len(s)
				a := int(v1.ToInt())
				b := int(v2.ToInt())
				if a < 0 || a > l {
					return core.NullValue, fmt.Errorf("start out of range")
				}
				if b < a || b > l {
					return core.NullValue, fmt.Errorf("end out of range")
				}
				return core.NewString(substring(s, a, b)), nil
			}

			return core.NullValue, fmt.Errorf("expected 1 or 2 parameters")
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.take",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.Int {
				return core.NullValue, fmt.Errorf("expected arg 1 to be int, got %s", args[0].TypeName())
			}

			s := this.ToString()
			i := int(args[0].ToInt())

			if len(s) > i {
				s = s[:i]
			}
			return core.NewString(s), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.hasPrefix",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
			}
			v := args[0].ToString()
			s := this.ToString()
			return core.NewBool(strings.HasPrefix(s, v)), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.hasSuffix",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
			}
			v := args[0].ToString()
			s := this.ToString()
			return core.NewBool(strings.HasSuffix(s, v)), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.indexOf",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			ln := len(args)

			if ln > 0 {
				if args[0].Type != core.String {
					return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
				}
			}

			if ln > 1 {
				if args[1].Type != core.Int {
					return core.NullValue, fmt.Errorf("expected arg 2 to be int, got %s", args[1].TypeName())
				}
			}

			sep := args[0].ToString()
			s := this.ToString()

			var i int
			if len(args) > 1 {
				i = int(args[1].ToInt())
				if i > len(s) {
					return core.NullValue, fmt.Errorf("index out of range")
				}
				s = s[i:]
			}
			return core.NewInt(strings.Index(s, sep) + i), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.lastIndexOf",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			ln := len(args)

			if ln > 0 {
				if args[0].Type != core.String {
					return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
				}
			}

			if ln > 1 {
				if args[1].Type != core.Int {
					return core.NullValue, fmt.Errorf("expected arg 2 to be int, got %s", args[1].TypeName())
				}
			}

			sep := args[0].ToString()
			s := this.ToString()

			if len(args) > 1 {
				i := int(args[1].ToInt())
				if i > len(s) {
					return core.NullValue, fmt.Errorf("index out of range")
				}
				s = s[i:]
			}
			return core.NewInt(strings.LastIndex(s, sep)), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.contains",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
			}

			sep := args[0].ToString()
			s := this.ToString()
			return core.NewBool(strings.Contains(s, sep)), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.rightPad",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
			}
			if args[1].Type != core.Int {
				return core.NullValue, fmt.Errorf("expected arg 2 to be int, got %s", args[1].TypeName())
			}
			pad := args[0].ToString()
			if len(pad) != 1 {
				return core.NullValue, fmt.Errorf("invalid pad size. Must be one character")
			}
			total := int(args[1].ToInt())
			s := this.ToString()
			return core.NewString(rightPad(s, rune(pad[0]), total)), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.leftPad",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
			}
			if args[1].Type != core.Int {
				return core.NullValue, fmt.Errorf("expected arg 2 to be int, got %s", args[1].TypeName())
			}

			pad := args[0].ToString()
			if len(pad) != 1 {
				return core.NullValue, fmt.Errorf("invalid pad size. Must be one character")
			}
			total := int(args[1].ToInt())
			s := this.ToString()
			return core.NewString(leftPad(s, rune(pad[0]), total)), nil
		},
	},
	core.NativeFunction{
		Name:      "String.prototype.equalFold",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected arg 1 to be string, got %s", args[0].TypeName())
			}
			eq := strings.EqualFold(this.ToString(), args[0].ToString())
			return core.NewBool(eq), nil
		},
	},
}

func substring(s string, start int, end int) string {
	start_str_idx := 0
	i := 0
	for j := range s {
		if i == start {
			start_str_idx = j
		}
		if i == end {
			return s[start_str_idx:j]
		}
		i++
	}
	return s[start_str_idx:]
}

// IsNumeric returns true if s contains only digits
func IsNumeric(s string) bool {
	for _, r := range s {
		if !IsDecimal(r) {
			return false
		}
	}
	return true
}

// IsDecimal returns true if r is a digit
func IsDecimal(r rune) bool {
	return r >= '0' && r <= '9'
}

// IsIdent returns if s is a valid identifier.
func IsIdent(s string) bool {
	for i, c := range s {
		if !isStrIdent(c, i) {
			return false
		}
	}
	return true
}

func isStrIdent(ch rune, pos int) bool {
	return ch == '_' ||
		'A' <= ch && ch <= 'Z' ||
		'a' <= ch && ch <= 'z' ||
		IsDecimal(ch) && pos > 0
}

func IsAlphanumericIdent(s string) bool {
	for i, c := range s {
		if c == '_' {
			continue
		}
		if !isAlphanumeric(c, i) {
			return false
		}
	}
	return true
}

func IsAlphanumeric(s string) bool {
	for _, c := range s {
		if !isAlphanumeric(c, 1) {
			return false
		}
	}
	return true
}

func isAlphanumeric(ch rune, pos int) bool {
	return 'A' <= ch && ch <= 'Z' ||
		'a' <= ch && ch <= 'z' ||
		IsDecimal(ch) && pos > 0
}

func Split(s, sep string) []string {
	parts := strings.Split(s, sep)
	var result []string
	for _, p := range parts {
		if p != "" {
			// only append non empty values
			result = append(result, p)
		}
	}
	return result
}

func rightPad(s string, pad rune, total int) string {
	l := total - utf8.RuneCountInString(s)
	if l < 1 {
		return s
	}
	return s + strings.Repeat(string(pad), l)
}

func leftPad(s string, pad rune, total int) string {
	l := total - utf8.RuneCountInString(s)
	if l < 1 {
		return s
	}
	return strings.Repeat(string(pad), l) + s
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

// https://stackoverflow.com/a/31832326/4264
func RandString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
