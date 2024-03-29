package core

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/gtlang/filesystem"
)

const VERSION = "0.9"

var BUILD string

func Run(fs filesystem.FS, path string) (Value, error) {
	p, err := Compile(fs, path)
	if err != nil {
		return NullValue, err
	}

	vm := NewVM(p)
	return vm.Run()
}

func RunStr(code string) (Value, error) {
	p, err := CompileStr(code)
	if err != nil {
		return NullValue, err
	}

	vm := NewVM(p)
	return vm.Run()
}

func NewVM(p *Program) *VM {
	vm := &VM{Program: p}

	globalFrame := &stackFrame{
		funcIndex: 0,
		values:    make([]Value, p.Functions[0].MaxRegIndex),
	}

	vm.callStack = []*stackFrame{globalFrame}
	vm.allocations = int64(p.kSize)
	return vm
}

func NewInitializedVM(p *Program, globals []Value) *VM {
	vm := &VM{
		Program:     p,
		initialized: true,
	}

	if len(globals) != p.Functions[0].MaxRegIndex {
		panic("invalid globals size")
	}

	globalFrame := &stackFrame{
		funcIndex: 0,
		values:    globals,
	}

	vm.callStack = []*stackFrame{globalFrame}
	vm.allocations = int64(p.kSize)
	return vm
}

type stackFrame struct {
	pc           int
	funcIndex    int
	maxRegIndex  int
	retAddress   *Address
	values       []Value
	closures     []*closureRegister
	finalizables []Finalizable
	retValueSet  bool
	retValue     Value
	exit         bool // if it should exit the program when returns
}

type VM struct {
	Program        *Program
	MaxSteps       int64
	MaxAllocations int64
	MaxFrames      int
	RetValue       Value
	Error          error
	Trusted        bool
	Context        interface{}
	FileSystem     filesystem.FS
	fp             int
	steps          int64
	allocations    int64
	initialized    bool
	callStack      []*stackFrame
	tryCatchs      []*tryCatch
}

func (vm *VM) Steps() int64 {
	return vm.steps
}

func (vm *VM) ResetSteps() {
	vm.steps = 0
}

func (vm *VM) AddSteps(n int64) error {
	vm.steps += n

	// Go doesn't check overflows
	if vm.steps < 0 {
		return vm.NewError("Step limit overflow: %d", vm.steps)
	}

	if vm.MaxSteps > 0 && vm.steps > vm.MaxSteps {
		return vm.NewError("Step limit reached: %d", vm.MaxSteps)
	}
	return nil
}

func (vm *VM) HasPermission(name string) bool {
	if vm.Trusted {
		return true
	}
	return vm.Program.HasPermission(name)
}

func (vm *VM) Clone(p *Program, globals []Value) *VM {
	m := NewInitializedVM(p, globals)
	m.MaxAllocations = vm.MaxAllocations
	m.MaxFrames = vm.MaxFrames
	m.MaxSteps = vm.MaxSteps
	m.FileSystem = vm.FileSystem
	m.Context = vm.Context
	m.Trusted = vm.Trusted
	return m
}

func (vm *VM) Initialized() bool {
	return vm.initialized
}

func (vm *VM) Initialize() error {
	vm.run(false)

	if vm.Error == io.EOF {
		vm.Error = nil
	}

	if vm.Error != nil {
		return vm.Error
	}

	// at this point global values have been initialized and
	// RunFunc can be called
	vm.initialized = true

	// the global function ends with a return so restore the fp back
	vm.fp = 0

	return nil
}

func (vm *VM) Run(args ...Value) (Value, error) {
	if !vm.initialized {
		if err := vm.Initialize(); err != nil {
			return NullValue, err
		}
	}

	// if it has an entry point call it
	f, ok := vm.Program.Function("main")
	if !ok {
		return vm.RetValue, nil
	}

	return vm.runFunc(f, true, nil, args...)
}

// RunFunc executes a program function by name with arguments as Value
func (vm *VM) RunFunc(name string, args ...Value) (Value, error) {
	f, ok := vm.Program.Function(name)
	if !ok {
		return NullValue, fmt.Errorf("function %s not found", name)
	}
	return vm.runFunc(f, false, nil, args...)
}

// RunFuncIndex executes a program function by index with arguments as Value
func (vm *VM) RunFuncIndex(index int, args ...Value) (Value, error) {
	f := vm.Program.Functions[index]
	return vm.runFunc(f, false, nil, args...)
}

// RunClosure executes a program closure
func (vm *VM) RunClosure(c Closure, args ...Value) (Value, error) {
	f := vm.Program.Functions[c.funcIndex]
	return vm.runFunc(f, false, c.closures, args...)
}

func (vm *VM) Globals() []Value {
	return vm.callStack[0].values
}

func (vm *VM) runFunc(f *Function, finalizeGlobals bool, closures []*closureRegister, args ...Value) (Value, error) {
	// allow to pass less args but not more
	if !f.Variadic && f.Arguments < len(args) {
		return NullValue, fmt.Errorf("function '%s' expects %d parameters, got %d",
			f.Name, f.Arguments, len(args))
	}

	currentFp := vm.fp
	currentTryCatchs := vm.tryCatchs

	// reset for the call
	vm.tryCatchs = nil

	// store the last pc for the return
	currentFrame := vm.callStack[vm.fp]
	currentFrame.retAddress = Void

	// add a new frame
	frame := vm.addFrame(f)
	frame.funcIndex = f.Index
	frame.maxRegIndex = f.MaxRegIndex
	frame.exit = true
	frame.closures = closures

	lenArgs := len(args)
	locals := vm.callStack[vm.fp].values

	if f.Variadic {
		regularArgs := f.Arguments - 1
		// set the parameters that are not variadic
		if regularArgs > 0 {
			for i := 0; i < regularArgs; i++ {
				if i >= lenArgs {
					// skip if not enouth parameters have been provided
					break
				}
				v := args[i]
				locals[i] = v
				if err := vm.AddAllocations(v.Size()); err != nil {
					return NullValue, err
				}
			}
		}
		// set the variadic as an array with the rest of the parameters
		if lenArgs > regularArgs {
			v := NewArrayValues(args[regularArgs:])
			if err := vm.AddAllocations(v.Size()); err != nil {
				return NullValue, err
			}
			locals[regularArgs] = v
		} else {
			// if no arguments are passed set the variadic param as an empty array
			locals[regularArgs] = NewArray(0)
		}
	} else {
		for i := 0; i < f.Arguments; i++ {
			if i >= lenArgs {
				// skip if not enouth parameters have been provided
				break
			}
			v := args[i]
			if err := vm.AddAllocations(v.Size()); err != nil {
				return NullValue, err
			}
			locals[i] = v
		}
	}

	vm.run(finalizeGlobals)

	// restore
	vm.tryCatchs = currentTryCatchs
	vm.fp = currentFp

	if vm.Error != nil && vm.Error != io.EOF {
		return NullValue, vm.Error
	}

	return vm.RetValue, nil
}

func (vm *VM) addFrame(f *Function) *stackFrame {
	frame := &stackFrame{values: make([]Value, f.MaxRegIndex)}
	vm.fp++
	vm.callStack = append(vm.callStack[:vm.fp], frame)
	return frame
}

func (vm *VM) createClosure() {
	// R(A) dest R(B value) funcIndex
	instr := vm.instruction()
	funcIndex := instr.B.Value

	// copy  closures carried from parent functions
	frame := vm.callStack[vm.fp]
	f := vm.Program.Functions[frame.funcIndex]
	fLen := len(f.Closures)
	frLen := len(frame.closures)
	c := Closure{funcIndex: int(funcIndex), closures: make([]*closureRegister, fLen+frLen)}
	copy(c.closures, frame.closures)

	// copy closures defined in this function.
	for i, r := range f.Closures {
		c.closures[frLen+i] = &closureRegister{register: r, values: frame.values}
	}

	vm.set(instr.A, NewObject(c))
}

// return a value from the current scope
func (vm *VM) RegisterValue(name string) (Value, bool) {
	// try the current frame
	if vm.fp > 0 {
		frame := vm.callStack[vm.fp]
		fn := vm.Program.Functions[frame.funcIndex]
		locals := frame.values
		for _, r := range fn.Registers {
			if r.Name == name {
				return locals[r.Index], true
			}
		}
	}

	// try globals
	fn := vm.Program.Functions[0]
	globals := vm.callStack[0].values
	for _, r := range fn.Registers {
		if r.Name == name {
			return globals[r.Index], true
		}
	}

	return NullValue, false
}

func (vm *VM) SetFinalizer(v Finalizable) {
	frame := vm.callStack[vm.fp]
	frame.finalizables = append(frame.finalizables, v)
}

func (vm *VM) SetGlobalFinalizer(v Finalizable) {
	frame := vm.callStack[0]
	frame.finalizables = append(frame.finalizables, v)
}

func (vm *VM) get(a *Address) Value {
	switch a.Kind {
	case AddrGlobal:
		return vm.callStack[0].values[a.Value]
	case AddrLocal:
		return vm.callStack[vm.fp].values[a.Value]
	case AddrFunc:
		return NewFunction(int(a.Value))
	case AddrNativeFunc:
		return NewNativeFunction(int(a.Value))
	case AddrData:
		return NewInt(int(a.Value))
	case AddrClosure:
		return vm.callStack[vm.fp].closures[a.Value].get()
	case AddrConstant:
		return vm.Program.Constants[a.Value]
	case AddrUnresolved:
		panic(fmt.Sprintf("Unresolved address: %v", a))
	default:
		panic(fmt.Sprintf("Invalid address kind: %v", a.Value))
	}
}

func (vm *VM) set(a *Address, v Value) {
	if err := vm.AddAllocations(v.Size()); err != nil {
		vm.Error = err
		return
	}

	switch a.Kind {
	case AddrGlobal:
		vm.callStack[0].values[a.Value] = v
	case AddrLocal:
		vm.callStack[vm.fp].values[a.Value] = v
	case AddrClosure:
		vm.callStack[vm.fp].closures[a.Value].set(v)
	default:
		panic(fmt.Sprintf("Invalid register address: %v", a))
	}
}

func (vm *VM) AddAllocations(size int) error {
	if vm.MaxAllocations == 0 {
		return nil
	}

	vm.allocations += int64(size)
	if vm.allocations > vm.MaxAllocations {
		return vm.NewError("Max allocations reached: %d", vm.MaxAllocations)
	}
	return nil
}

func (vm *VM) setPrototype(name string, this Value, dst *Address) bool {
	if m, ok := vm.getNativePrototype(name, this); ok {
		vm.set(dst, NewObject(m))
		return true
	}

	if m, ok := vm.getProgramPrototype(name, this); ok {
		vm.set(dst, NewObject(m))
		return true
	}
	return false
}

func (vm *VM) getNativePrototype(name string, this Value) (nativePrototype, bool) {
	f, ok := allNativeMap[name]
	if ok {
		return nativePrototype{this: this, fn: f.Index}, true
	}
	return nativePrototype{}, false
}

func (vm *VM) getProgramPrototype(name string, this Value) (method, bool) {
	p := vm.Program
	f, ok := p.Function(name)
	if !ok {
		return method{}, false
	}
	return method{fn: f.Index, this: this}, true
}

func (vm *VM) Stacktrace() []string {
	st := vm.getStackTrace()
	s := make([]string, len(st))

	for i, l := range st {
		s[i] = l.String()
	}

	return s
}

func (vm *VM) getStackTrace() []TraceLine {
	var trace []TraceLine

	p := vm.Program

	for i := vm.fp; i >= 0; i-- {
		frame := vm.callStack[i]
		f := p.Functions[frame.funcIndex]

		if f.IsGlobal && vm.initialized {
			// the global function has ended
			continue
		}

		trace = append(trace, p.ToTraceLine(f, frame.pc))
	}

	return trace
}

type messageError interface {
	Message() string
}

func (vm *VM) WrapError(err error) Error {
	var msg string

	st := vm.getStackTrace()

	switch t := err.(type) {
	case Error:
		t.stacktrace = append(t.stacktrace, st...)
		return t
	case messageError:
		msg = t.Message()
	default:
		msg = t.Error()
	}

	return Error{
		message:     msg,
		instruction: vm.instruction(),
		stacktrace:  st,
	}
}

func (vm *VM) NewError(format string, a ...interface{}) Error {
	st := vm.getStackTrace()
	return Error{
		message:     fmt.Sprintf(format, a...),
		instruction: vm.instruction(),
		stacktrace:  st,
	}
}

func (vm *VM) returnFromFinally() bool {
	l := len(vm.tryCatchs)
	if l == 0 {
		return false
	}
	fp := vm.fp

	// loop trough all the nested finally's and execute them
	for i := l - 1; i >= 0; i-- {
		try := vm.tryCatchs[i]

		// only execute the finallys of its own frame. Other frames
		// will execute their own.
		if try.fp != fp {
			return false
		}

		finallyPC := try.finallyPC

		// if there is no finally
		if finallyPC == -1 {
			vm.tryCatchs = vm.tryCatchs[:l-1]
			continue
		}

		frame := vm.callStack[fp]

		// if we are already inside the finally and returning from it
		fi := frame.funcIndex
		f := vm.Program.Functions[fi]
		fnEndPC := len(f.Instructions)

		if frame.pc >= finallyPC && (frame.pc <= fnEndPC) {
			vm.tryCatchs = vm.tryCatchs[:l-1]
			if frame.retValueSet {
				// if there was a return value stored from the main block, clear it
				// because now we are returning from inside the finally and this is
				// return value has precedence.
				frame.retValueSet = false
				frame.retValue = NullValue
			}

			continue
		}

		try.retPC = frame.pc
		vm.setPC(finallyPC)
		return true
	}

	return false
}

// returns true if the error is handled
func (vm *VM) handle(err error) bool {
	ln := len(vm.tryCatchs)
	if ln == 0 {
		vm.Error = err
		return false
	}

	try := vm.tryCatchs[ln-1]

	// if its handled is an exception inside the catch
	if try.catchExecuted {
		// execute the finally even if the catch has thrown an exception
		if try.finallyPC != -1 && !try.finallyExecuted {
			try.err = err
			try.finallyExecuted = true
			vm.restoreStackframe(try)
			vm.setPC(try.finallyPC)
			return true
		}

		// jump to the parent catch if exists
		vm.tryCatchs = vm.tryCatchs[:ln-1]
		if ln > 1 {
			for i := ln - 2; i >= 0; i-- {
				try = vm.tryCatchs[i]
				if try.catchExecuted {
					// consume try-catchs that have thrown inside the catch
					continue
				}
				break
			}
			if try.catchExecuted {
				// if no catch found then it is unhandled
				vm.Error = err
				return false
			}
		} else {
			// or return an unhandled error.
			// An error thrown in the catch has precedence
			if try.err != nil {
				vm.Error = try.err
			} else {
				vm.Error = err
			}
			return false
		}
	}

	// mark this try as handled in case a exception is throw inside the
	// catch block to discard it.
	try.catchExecuted = true

	vm.Error = nil // handled

	jumpTo := try.catchPC
	if jumpTo == -1 {
		// if there is no catch block the err is unhandled
		try.err = err

		// if there is no catch block go directly to the finally block.
		jumpTo = try.finallyPC
		if jumpTo == -1 {
			// TODO: this could be catched by the compiler
			vm.Error = vm.NewError("try without catch or finally")
			return false
		}
	}

	// If jumps to finally directly the error is unhandled
	// check also that it doesn't have an empty catch
	if jumpTo == try.finallyPC {
		try.finallyExecuted = true
	}

	// restore the framepointer and local memory
	// where the try-catch is declared
	vm.restoreStackframe(try)

	// advance to the catch part
	vm.setPC(jumpTo)

	if try.errorReg != Void {
		e, ok := err.(Error)
		if !ok {
			e = Error{message: err.Error()}
		}
		vm.set(try.errorReg, NewObject(e))
	}

	return true
}

// restore the framepointer and local memory
// where the try-catch is declared
func (vm *VM) restoreStackframe(try *tryCatch) {
	if vm.fp == try.fp {
		return
	}

	// clean up frames until the catch
	for i := vm.fp; i > try.fp; i-- {
		vm.cleanupFrame(i)
	}

	vm.fp = try.fp
}

func (vm *VM) runFinalizables(frame *stackFrame) {
	if frame.finalizables != nil {
		fzs := frame.finalizables
		frame.finalizables = nil

		for _, v := range fzs {
			if err := v.Close(); err != nil {
				// set the error but continue running all the finalizers
				vm.Error = err
			}
		}
	}
}

func (vm *VM) cleanupFrame(index int) {
	frame := vm.callStack[index]
	vm.runFinalizables(frame)
}

func (vm *VM) run(finalizeGlobals bool) {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("PANIC: %s", r)

			pt := "\n" + strings.Join(vm.Stacktrace(), "\n")
			pt = strings.Replace(pt, "\n", "\n [GT] -> ", -1)
			msg += pt

			st := strings.Replace("\n"+Stacktrace(), "\n ", "\n [Go] ", -1)
			msg += st

			// don't call newError because it's handling itself the stack trace.
			vm.Error = Error{
				message:     msg,
				instruction: vm.instruction(),
			}
		}
	}()

	if finalizeGlobals {
		defer func() {
			vm.runFinalizables(vm.callStack[0])
		}()
	}

	p := vm.Program

	for {
		vm.steps++
		if vm.MaxSteps > 0 && vm.steps > vm.MaxSteps {
			vm.Error = vm.NewError("Step limit reached: %d", vm.MaxSteps)
			return
		}

		frame := vm.callStack[vm.fp]
		i := frame.funcIndex
		f := p.Functions[i]
		instr := f.Instructions[frame.pc]

		// Print step
		// fmt.Println("->", fmt.Sprintf("FN %-2d", i), fmt.Sprintf("PC %-6d", frame.pc), instr, "  "+f.Name)

		r := exec(instr, vm)
		switch r {
		case vm_next:
			if vm.Error != nil {
				return
			}
			frame.pc++
			continue

		case vm_continue:
			continue

		case vm_exit:
			return
		}
	}
}

func (vm *VM) setPC(pc int) {
	vm.callStack[vm.fp].pc = pc
}

func (vm *VM) incPC(steps int) {
	vm.callStack[vm.fp].pc += steps
}

func (vm *VM) call(a, b *Address, args []Value) int {
	// TODO Handle variadic and spread with closures.
	// get the function
	var f *Function
	var closures []*closureRegister

	var isMethod bool
	var this Value

	switch a.Kind {
	case AddrFunc:
		f = vm.Program.Functions[a.Value]
	case AddrNativeFunc:
		if err := vm.callNativeFunc(int(a.Value), args, b, this); err != nil {
			if vm.handle(vm.WrapError(err)) {
				return vm_continue
			} else {
				return vm_exit
			}
		}
		return vm_next
	default:
		value := vm.get(a)
		switch value.Type {
		case Func:
			f = vm.Program.Functions[value.ToFunction()]
		case Object:
			switch t := value.ToObject().(type) {
			case Closure:
				f = vm.Program.Functions[t.funcIndex]
				closures = t.closures
			case method:
				f = vm.Program.Functions[t.fn]
				isMethod = true
				this = t.this
			case nativePrototype:
				if err := vm.callNativeFunc(t.fn, args, b, t.this); err != nil {
					if vm.handle(vm.WrapError(err)) {
						return vm_continue
					} else {
						return vm_exit
					}
				}
				return vm_next
			case NativeMethod:
				if err := vm.callNativeMethod(t, args, b); err != nil {
					if vm.handle(vm.WrapError(err)) {
						return vm_continue
					} else {
						return vm_exit
					}
				}
				return vm_next
			default:
				if vm.handle((vm.NewError(fmt.Sprintf("Invalid value. Expected a function, got %v", value)))) {
					return vm_continue
				} else {
					return vm_exit
				}
			}
		default:
			if vm.handle((vm.NewError(fmt.Sprintf("Invalid value. Expected a function, got %v", value)))) {
				return vm_continue
			} else {
				return vm_exit
			}
		}
	}

	return vm.callProgramFunc(f, b, args, isMethod, this, closures)
}

func (vm *VM) callProgramFunc(f *Function, retAddr *Address, args []Value, isMethod bool, this Value, closures []*closureRegister) int {
	// set where to store the return value after the call in the current frame
	frame := vm.callStack[vm.fp]
	frame.retAddress = retAddr

	// add a new frame
	newFrame := vm.addFrame(f)
	newFrame.funcIndex = f.Index
	newFrame.maxRegIndex = f.MaxRegIndex
	newFrame.closures = closures

	if vm.MaxFrames > 0 && vm.fp > vm.MaxFrames {
		vm.Error = vm.NewError("Max stack frames reached: %d", vm.MaxFrames)
		return vm_exit
	}

	locals := newFrame.values

	// copy arguments
	if f.Arguments > 0 {
		count := len(args)

		if f.Variadic {
			regularArgs := f.Arguments - 1
			if count < regularArgs {
				copy(locals, args)
				// zero the rest of the args because memory can be reused
				for i := count; i < f.Arguments; i++ {
					locals[i] = NullValue
				}
			} else {
				for i := 0; i < regularArgs; i++ {
					locals[i] = args[i]
				}
				locals[regularArgs] = NewArrayValues(args[regularArgs:])
			}
		} else {
			if count > f.Arguments {
				// ignore if too many parameters are passed
				copy(locals, args[:f.Arguments])
			} else {
				copy(locals, args)
				if count < f.Arguments {
					// Zero the rest of the args because memory can be reused
					for i := count; i < f.Arguments; i++ {
						locals[i] = NullValue
					}
				}
			}
		}
	}

	if isMethod {
		// this is always the next value after the arguments
		locals[f.Arguments] = this
	}

	return vm_next
}

func (vm *VM) callNativeFunc(i int, args []Value, retAddress *Address, this Value) error {
	f := allNativeFuncs[i]

	l := f.Arguments
	if l != -1 && l != len(args) {
		return fmt.Errorf("function '%s' expects %d parameters, got %d", f.Name, l, len(args))
	}

	ret, err := f.Function(this, args, vm)
	if err != nil {
		return err
	}

	if retAddress != Void {
		vm.set(retAddress, ret)
	}

	return nil
}

func (vm *VM) callNativeMethod(m NativeMethod, args []Value, retAddress *Address) error {
	ret, err := m(args, vm)
	if err != nil {
		return err
	}

	if retAddress != Void {
		vm.set(retAddress, ret)
	}

	return nil
}

func (vm *VM) setToObject(instr *Instruction) error {
	av := vm.get(instr.A) // array or map
	bv := vm.get(instr.B) // index
	cv := vm.get(instr.C) // value

	if av.Type == Object {
		if cr, ok := av.ToObject().(*closureRegister); ok {
			// if it is a closure get the underlying value
			av = cr.get()
		}
	}

	if err := vm.AddAllocations(cv.Size()); err != nil {
		return err
	}

	switch bv.Type {
	case Int:
		switch av.Type {
		case Array:
			av.ToArray()[bv.ToInt()] = cv
		case Bytes:
			if cv.Type != Int {
				return vm.NewError("Can't convert %v to byte", cv.TypeName())
			}
			av.ToBytes()[bv.ToInt()] = byte(cv.ToInt())
		case Object:
			i, ok := av.ToObject().(IndexerSetter)
			if !ok {
				return vm.NewError("Can't set by index %v", av.TypeName())
			}
			if err := i.SetIndex(int(bv.ToInt()), cv); err != nil {
				return vm.WrapError(err)
			}
		case Map:
			m := av.ToMap()
			m.Mutex.Lock()
			m.Map[bv.ToString()] = cv
			m.Mutex.Unlock()
		default:
			return vm.NewError("Can't set %v by index", av.Type)
		}
	case String:
		switch av.Type {
		case Map:
			m := av.ToMap()
			m.Mutex.Lock()
			m.Map[bv.ToString()] = cv
			m.Mutex.Unlock()
		case Object:
			i, ok := av.ToObject().(PropertySetter)
			if !ok {
				return vm.NewError("Readonly property or not a PropertySetter: %T", av.TypeName())
			}
			if err := i.SetProperty(bv.ToString(), cv, vm); err != nil {
				return vm.WrapError(err)
			}

		case Null:
			// allow to set properties by default to uninitialized objects
			v := map[string]Value{
				bv.ToString(): cv,
			}
			vm.set(instr.A, NewMapValues(v))
		default:
			return vm.NewError("Readonly property or not Map or PropertySetter: %v", av.TypeName())
		}
	default:
		return vm.NewError("Invalid index %s", bv.TypeName())
	}

	return nil
}

func (vm *VM) getFromObject(instr *Instruction) error {
	bv := vm.get(instr.B) // source
	if bv.IsNil() {
		return vm.NewError("Attempted to use null in a case where an object is required")
	}

	if bv.Type == Object {
		if cr, ok := bv.ToObject().(*closureRegister); ok {
			// if it is a closure get the underlying value
			bv = cr.get()
		}
	}

	cv := vm.get(instr.C) // index or key

	switch cv.Type {

	case Int:
		switch bv.Type {

		case Array:
			i := cv.ToInt()
			if i < 0 {
				return vm.NewError("Index out of range in string")
			}
			s := bv.ToArray()
			if len(s) <= int(i) {
				return vm.NewError("Index out of range")
			}
			vm.set(instr.A, s[i])

		case Object:
			o := bv.ToObject()

			i, ok := o.(IndexerGetter)
			if !ok {
				return vm.NewError("%T can't be accessed by index", o)
			}

			v, err := i.GetIndex(int(cv.ToInt()))
			if err != nil {
				return vm.WrapError(err)
			}
			vm.set(instr.A, v)

		case String:
			i := cv.ToInt()
			if i < 0 {
				return vm.NewError("Index out of range in string")
			}
			vm.set(instr.A, NewRune(rune(bv.ToString()[i])))

		case Bytes:
			i := cv.ToInt()
			if i < 0 {
				return vm.NewError("Index out of range in string")
			}
			v := bv.ToBytes()
			b := v[i]
			vm.set(instr.A, NewInt(int(b)))

		case Map:
			m := bv.ToMap()
			m.Mutex.RLock()
			key := cv.ToString()
			v, ok := m.Map[key]
			if !ok {
				v = UndefinedValue
			}
			m.Mutex.RUnlock()
			vm.set(instr.A, v)

		default:
			return vm.NewError("The value must be Array or Indexer: %v", bv.Type)
		}

	// If is string is a property or method.
	case String:
		key := cv.ToString()

		switch bv.Type {

		case Map:
			m := bv.ToMap()
			m.Mutex.RLock()
			v, ok := m.Map[key]
			if !ok {
				v = UndefinedValue
			}
			m.Mutex.RUnlock()
			vm.set(instr.A, v)

		case Object:
			obj := bv.ToObject()

			if n, ok := obj.(Callable); ok {
				if m := n.GetMethod(key); m != nil {
					vm.set(instr.A, NewObject(m))
					return nil
				}
			}

			if i, ok := obj.(PropertyGetter); ok {
				v, err := i.GetProperty(key, vm)
				if err != nil {
					return vm.WrapError(err)
				}
				if v.Type != Undefined {
					vm.set(instr.A, v)
					return nil
				}
			}

			// try if it's an enunmerable method
			if _, ok := obj.(Enumerable); ok {
				if m, ok := vm.getNativePrototype("Array.prototype."+key, bv); ok {
					vm.set(instr.A, NewObject(m))
					return nil
				}
				if m, ok := vm.getProgramPrototype("Array.prototype."+key, bv); ok {
					vm.set(instr.A, NewObject(m))
					return nil
				}
			}

			// allow to call anything on an object.
			// If it doesn't exist set it to undefined
			vm.set(instr.A, UndefinedValue)

		case Array:
			switch key {
			case "length":
				vm.set(instr.A, NewInt(len(bv.ToArray())))
				return nil
			default:
				if !vm.setPrototype("Array.prototype."+key, bv, instr.A) {
					vm.set(instr.A, UndefinedValue)
				}
				return nil
			}

		case String:
			switch key {
			case "length":
				vm.set(instr.A, NewInt(len(bv.ToString())))
				return nil
			case "runeCount":
				vm.set(instr.A, NewInt(utf8.RuneCountInString(bv.ToString())))
				return nil
			default:
				if !vm.setPrototype("String.prototype."+key, bv, instr.A) {
					vm.set(instr.A, UndefinedValue)
				}
				return nil
			}

		case Undefined:
			return vm.NewError("Attempted to use undefined in a case where an object is required")

		case Null:
			return vm.NewError("Attempted to use null in a case where an object is required")

		case Bytes:
			switch key {
			case "length":
				vm.set(instr.A, NewInt(len(bv.ToBytes())))
				return nil
			default:
				if !vm.setPrototype("Bytes.prototype."+key, bv, instr.A) {
					if !vm.setPrototype("Array.prototype."+key, bv, instr.A) {
						vm.set(instr.A, UndefinedValue)
					}
				}
				return nil
			}

		default:
			return vm.NewError("Can't read %s of %s", key, bv.TypeName())
		}

	default:
		return vm.NewError("Invalid index %s", cv.TypeName())
	}

	return nil
}

func (vm *VM) instruction() *Instruction {
	frame := vm.callStack[vm.fp]
	i := frame.funcIndex
	f := vm.Program.Functions[i]
	return f.Instructions[frame.pc]
}

type method struct {
	this Value
	fn   int
}

type tryCatch struct {
	catchPC         int
	errorReg        *Address
	finallyPC       int
	fp              int
	retPC           int
	err             error
	catchExecuted   bool
	finallyExecuted bool
}

type Closure struct {
	funcIndex int
	closures  []*closureRegister
}

func (c Closure) Type() string {
	return "Closure"
}

func (c Closure) Export(recursionLevel int) interface{} {
	return "<closure>"
}

type closureRegister struct {
	register *Register
	values   []Value
}

func (c *closureRegister) get() Value {
	return c.values[c.register.Index]
}

func (c *closureRegister) set(v Value) {
	c.values[c.register.Index] = v
}

func (c *closureRegister) Type() string {
	return c.get().Type.String()
}

func (c *closureRegister) Size() int {
	return c.get().Size()
}

func (c *closureRegister) Export(recursionLevel int) interface{} {
	return c.get().Export(recursionLevel)
}
