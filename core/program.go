//go:generate stringer -type=Opcode,AddressKind

package core

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

type AddressKind byte

const (
	AddrVoid AddressKind = iota
	AddrLocal
	AddrGlobal
	AddrConstant
	AddrClosure
	AddrFunc
	AddrNativeFunc
	AddrData
	AddrUnresolved
)

type Address struct {
	Kind  AddressKind
	Value int32
}

func NewAddress(kind AddressKind, value int) *Address {
	return &Address{kind, int32(value)}
}

func (r *Address) Equal(b *Address) bool {
	return r.Kind == b.Kind && r.Value == b.Value
}

func (r *Address) String() string {
	switch r.Kind {
	case AddrFunc:
		return fmt.Sprintf("%dF", r.Value)
	case AddrNativeFunc:
		return fmt.Sprintf("%dN", r.Value)
	case AddrConstant:
		return fmt.Sprintf("%dK", r.Value)
	case AddrGlobal:
		return fmt.Sprintf("%dG", r.Value)
	case AddrLocal:
		return fmt.Sprintf("%dL", r.Value)
	case AddrClosure:
		return fmt.Sprintf("%dC", r.Value)
	case AddrData:
		return fmt.Sprintf("%dD", r.Value)
	case AddrUnresolved:
		return fmt.Sprintf("%dU", r.Value)
	case AddrVoid:
		return "--"
	default:
		return fmt.Sprintf("%d-%d?", r.Kind, r.Value)
	}
}

var Void = NewAddress(AddrVoid, 0)

type Instruction struct {
	Opcode Opcode
	A      *Address
	B      *Address
	C      *Address
}

func NewInstruction(op Opcode, a, b, c *Address) *Instruction {
	return &Instruction{op, a, b, c}
}

func (i *Instruction) String() string {
	return i.Format(false)
}

func (i *Instruction) Format(padd bool) string {
	op := strings.ToUpper(i.Opcode.String()[3:])
	return fmt.Sprintf("%s %6v %6v %6v", op, i.A, i.B, i.C)
}

type Register struct {
	Name     string
	Index    int
	StartPC  int
	EndPC    int
	Exported bool
	Module   string
}

func (r *Register) Equals(b *Register) bool {
	return r.Name == b.Name &&
		r.StartPC == b.StartPC &&
		r.EndPC == b.EndPC
}

type Class struct {
	Name      string
	Fields    []*Field
	Functions []int
	Exported  bool
}

type Field struct {
	Name     string
	Exported bool
}

type FunctionKind byte

const (
	User FunctionKind = iota
	Init
	Main
	Global
)

type Function struct {
	Name         string
	Variadic     bool
	Exported     bool
	IsClass      bool
	IsGlobal     bool
	Index        int
	Arguments    int
	MaxRegIndex  int
	Kind         FunctionKind
	Registers    []*Register
	Closures     []*Register
	Instructions []*Instruction
	Positions    []Position
}

type Program struct {
	sync.Mutex
	Functions   []*Function
	Classes     []*Class
	Constants   []Value
	Files       []string
	Directives  map[string]string
	Permissions map[string]bool
	Resources   map[string][]byte

	kSize   int // the memory for all constants
	funcMap map[string]*Function
}

func (p *Program) HasPermission(name string) bool {
	var value bool

	p.Lock()

	if p.Permissions == nil {
		p.initPermissions()
	}

	if p.Permissions[name] {
		value = true
	}

	if name != "trusted" && p.Permissions["trusted"] {
		value = true
	}

	p.Unlock()

	return value
}

func (p *Program) initPermissions() {
	d, ok := p.Directives["permissions"]
	if ok {
		values := split(d, " ")
		p.Permissions = make(map[string]bool, len(values))
		for _, v := range values {
			p.Permissions[v] = true
		}

	} else {
		p.Permissions = make(map[string]bool)
	}
}

func (p *Program) addConstant(v Value) *Address {
	for i, k := range p.Constants {
		if k.Type == v.Type && k.object == v.object {
			return NewAddress(AddrConstant, i)
		}
	}

	i := len(p.Constants)
	p.Constants = append(p.Constants, v)
	return NewAddress(AddrConstant, i)
}

func (p *Program) Strip() {
	for i := range p.Functions {
		f := p.Functions[i]
		if strings.Contains(f.Name, ".prototype.") {
			continue
		}
		if !f.Exported && f.Name != "main" {
			f.Name = ""
		}
		for j := range f.Registers {
			r := f.Registers[j]
			if !r.Exported {
				r.Name = ""
			}
		}
	}
}

func (p *Program) FileIndex(file string) int {
	for i, f := range p.Files {
		if file == f {
			return i
		}
	}
	return -1
}

func (p *Program) ToTraceLine(f *Function, pc int) TraceLine {
	ln := len(f.Positions)

	if ln == 0 || ln <= pc {
		return TraceLine{Function: f.Name}
	}

	// Instructions that have an empty position belong to the last source line in code.
	var file string
	var pos Position
	for {
		pos = f.Positions[pc]
		if pc > 0 && pos.Line == 0 { // pos.Line is in base 1 so this is an empty position
			pc--
			continue
		}

		if len(p.Files) > 0 {
			file = p.Files[pos.File]
		}
		break
	}

	return TraceLine{Function: f.Name, File: file, Line: pos.Line}
}

func (p *Program) Function(name string) (*Function, bool) {
	p.Lock()
	if p.funcMap == nil {
		funcMap := make(map[string]*Function, len(p.Functions))
		p.funcMap = funcMap
		for _, f := range p.Functions {
			funcMap[f.Name] = f
		}
	}
	f, ok := p.funcMap[name]
	p.Unlock()
	return f, ok
}

func (p *Program) AddDirective(name string, value string) {
	v, ok := p.Directives[name]
	if ok {
		p.Directives[name] = v + " " + value
	} else {
		p.Directives[name] = value
	}
}

type TraceLine struct {
	Function string
	File     string
	Line     int
}

func (p TraceLine) String() string {
	var buf bytes.Buffer

	switch p.File {
	case "", ".":
		if p.Line > 0 {
			fmt.Fprintf(&buf, "line %d", p.Line)
		} else {
			fmt.Fprint(&buf, p.Function)
		}
	default:
		fmt.Fprintf(&buf, "%s:%d", p.File, p.Line)
	}

	return buf.String()
}

func (p TraceLine) SameLine(o TraceLine) bool {
	return p.File == o.File && p.Line == o.Line
}

type Position struct {
	File   int
	Line   int
	Column int
}

func Print(p *Program) {
	Fprint(os.Stdout, p)
}

func PrintFunction(f *Function, p *Program) {
	FprintFunction(os.Stdout, f, p)
}

func Sprint(p *Program) (string, error) {
	var b bytes.Buffer
	Fprint(&b, p)
	return b.String(), nil
}

func Fprint(w io.Writer, p *Program) {
	if len(p.Classes) > 0 {
		fmt.Fprint(w, "\n")
		for i, c := range p.Classes {
			fmt.Fprintf(w, "\n%dC %s", i, c.Name)
		}
	}

	if len(p.Functions) > 0 {
		fmt.Fprint(w, "\n")
		for _, f := range p.Functions {
			FprintFunction(w, f, p)
		}
	}

	if len(p.Constants) > 0 {
		fmt.Fprint(w, "\n")
		FprintConstants(w, p)
	}

	fmt.Fprint(w, "\n")
}

func FprintFunction(w io.Writer, f *Function, p *Program) {
	// if len(f.Instructions) == 0 && strings.ContainsRune(f.Name, '@') {
	// 	// ignore autogenerated empty funcs
	// 	return
	// }

	fmt.Fprintf(w, "\n%dF %s", f.Index, f.Name)

	for i, v := range f.Instructions {
		printInstruction(w, p, f, i, v)
	}

	var regType string
	if f.Index == 0 {
		regType = "G"
	} else {
		regType = "L"
	}

	fmt.Fprintf(w, "\n  MaxRegIndex %d", f.MaxRegIndex)
	for i, r := range f.Registers {
		fmt.Fprintf(w, "\n  %d%s %s %d-%d", i, regType, r.Name, r.StartPC, r.EndPC)
	}

	fmt.Fprint(w, "\n")
}

func printInstruction(w io.Writer, p *Program, f *Function, i int, instr *Instruction) {
	fmt.Fprintf(w, "\n  %-5d %s", i, instr.Format(true))

	if len(f.Positions) > i {
		t := p.ToTraceLine(f, i)
		fmt.Fprintf(w, "   ;   %s", t.String())
	}

	fmt.Fprint(w)
}

func FprintConstants(w io.Writer, p *Program) {
	for i, k := range p.Constants {
		switch k.Type {
		case String:
			s := k.ToString()
			if len(s) > 50 {
				s = s[:50]
			}
			s = strings.Replace(s, "\n", "\\n", -1)
			fmt.Fprintf(w, "%dK string %v\n", i, s)
		default:
			fmt.Fprintf(w, "%dK %v %v\n", i, k.Type, k.ToString())
		}
	}
}

func SprintNames(p *Program, registers bool) (string, error) {
	var b bytes.Buffer
	FprintNames(&b, p, registers)
	return b.String(), nil
}

func PrintNames(p *Program, registers bool) {
	FprintNames(os.Stdout, p, registers)
}

func FprintNames(w io.Writer, p *Program, registers bool) {
	if len(p.Classes) > 0 {
		for i, c := range p.Classes {
			fmt.Fprintf(w, "\n%dC %s", i, c.Name)

			functions := make([]*Function, len(c.Functions))
			for i, fIndex := range c.Functions {
				functions[i] = p.Functions[fIndex]
			}
			fprintFunctionNames(w, p, true, 1, functions, registers)
		}
		fmt.Fprint(w, "\n")
	}

	fprintFunctionNames(w, p, false, 0, p.Functions, registers)
	fmt.Fprint(w, "\n")
}

func fprintFunctionNames(w io.Writer, p *Program, isClass bool, indent int, functions []*Function, registers bool) {
	for i, f := range functions {
		if !isClass && f.IsClass {
			continue
		}

		fmt.Fprintf(w, "\n%s%dF %s", strings.Repeat("\t", indent), i, f.Name)

		fmt.Fprintf(w, "    %s", p.ToTraceLine(f, 0).String())

		var addrType string
		if f.IsGlobal {
			addrType = "G"
		} else {
			addrType = "L"
		}

		if registers {
			for j, r := range f.Registers {
				fmt.Fprintf(w, "\n%s%d%s %s", strings.Repeat("\t", indent+1), j, addrType, r.Name)
			}
		}
	}
}

func split(s, sep string) []string {
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
