package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

func NewPublicError(msg string) Error {
	return Error{message: msg, public: true}
}

type Error struct {
	pc          int
	message     string
	public      bool
	instruction *Instruction
	stacktrace  []TraceLine
	wraped      []Error
}

func (e *Error) Wrap(inner Error) {
	if e.public && inner.public {
		e.message += ": " + inner.message
	}

	e.wraped = append(e.wraped, inner)
}

func (e Error) Public() bool {
	return e.public
}

func (e *Error) SetPublic(v bool) {
	e.public = v
}

func (e Error) String() string {
	return e.Error()
}

func (e Error) Error() string {
	var b = &bytes.Buffer{}

	fmt.Fprintf(b, "%s\n", e.message)

	for _, s := range e.stacktrace {
		if s.Function == "" || s.Line == 0 {
			continue // this is an empty position
		}
		fmt.Fprintf(b, " -> %s\n", s.String())
	}

	for _, inner := range e.wraped {
		fmt.Fprintf(b, "\n%s\n", inner.Error())
	}

	return b.String()
}

func (e Error) Stack() string {
	var b = &bytes.Buffer{}

	for _, s := range e.stacktrace {
		if s.Function == "" && s.File == "" && s.Line == 0 {
			continue // this is an empty position
		}
		fmt.Fprintf(b, " -> %s\n", s.String())
	}

	return b.String()
}

func (e Error) stackLines() []string {
	lines := make([]string, len(e.stacktrace))

	for _, s := range e.stacktrace {
		if s.Function == "" && s.File == "" && s.Line == 0 {
			continue // this is an empty position
		}
		lines = append(lines, s.String())
	}

	return lines
}

func (e Error) Message() string {
	return e.message
}

func (e Error) Type() string {
	return "Exception"
}

func (e Error) GetProperty(name string, vm *VM) (Value, error) {
	switch name {
	case "public":
		return NewBool(e.public), nil
	case "message":
		return NewString(e.message), nil
	case "pc":
		return NewInt(e.pc), nil
	case "stackTrace":
		return NewString(e.Stack()), nil
	}

	return UndefinedValue, nil
}

func (e Error) GetMethod(name string) NativeMethod {
	switch name {
	case "toString":
		return e.toString
	}
	return nil
}

func (e Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Message    string
		StackTrace []string
	}{
		Message:    e.message,
		StackTrace: e.stackLines(),
	})
}

func (e Error) toString(args []Value, vm *VM) (Value, error) {
	return NewString(e.Error()), nil
}

func Stacktrace() string {
	c := callers()
	return stacktrace(c)
}

func stacktrace(stack *stack) string {
	var buf bytes.Buffer

	for _, f := range stack.StackTrace() {
		pc := f.pc()
		fn := runtime.FuncForPC(pc)

		if strings.HasPrefix(fn.Name(), "core.") {
			// ignore Go src
			continue
		}

		file, _ := fn.FileLine(pc)

		buf.WriteString(" -> ")
		buf.WriteString(file)
		buf.WriteRune(':')
		buf.WriteString(strconv.Itoa(f.line()))
		buf.WriteRune('\n')
	}

	return buf.String()
}

func callers() *stack {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(4, pcs[:])
	var st stack = pcs[0:n]
	return &st
}

// Frame represents a program counter inside a stack frame.
type Frame uintptr

// pc returns the program counter for this frame;
// multiple frames may have the same PC value.
func (f Frame) pc() uintptr { return uintptr(f) - 1 }

// StackTrace is stack of Frames from innermost (newest) to outermost (oldest).
type StackTrace []Frame

// stack represents a stack of program counters.
type stack []uintptr

func (s *stack) StackTrace() StackTrace {
	f := make([]Frame, len(*s))
	for i := 0; i < len(f); i++ {
		f[i] = Frame((*s)[i])
	}
	return f
}

// line returns the line number of source code of the
// function for this Frame's pc.
func (f Frame) line() int {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return 0
	}
	_, line := fn.FileLine(f.pc())
	return line
}
