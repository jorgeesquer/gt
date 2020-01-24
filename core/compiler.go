package core

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gtlang/filesystem"
	"github.com/gtlang/gt/ast"
	"github.com/gtlang/gt/parser"
)

var builtinFuncs = []string{"go", "defer", "panic", "T"}

func Compile(fs filesystem.FS, path string) (*Program, error) {
	a, err := parser.Parse(fs, path)
	if err != nil {
		return nil, err
	}

	c := NewCompiler()
	return c.Compile(a)
}

func CompileStr(code string) (*Program, error) {
	a, err := parser.ParseStr(code)
	if err != nil {
		return nil, err
	}

	c := NewCompiler()
	return c.Compile(a)
}

func AddBuiltinFunc(name string) {
	builtinFuncs = append(builtinFuncs, name)
}

func addDirectives(p *Program, file *ast.File) error {
	for _, d := range file.Directives {
		var key, value string
		i := strings.IndexRune(d, ' ')
		if i == -1 {
			key = d
		} else {
			key = d[:i]
			value = d[i+1:]
		}
		existing, ok := p.Directives[key]
		if ok {
			existing += " " + value
		} else {
			existing = value
		}
		p.Directives[key] = existing
	}

	return nil
}

type ByPriority []ast.Stmt

func (a ByPriority) Len() int      { return len(a) }
func (a ByPriority) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByPriority) Less(i, j int) bool {
	v, ok := a[i].(*ast.VarDeclStmt)
	return ok && v.IsEnum
}

type functionInfo struct {
	function  *Function
	parent    *functionInfo
	pos       ast.Position
	module    string
	anonymous bool

	// If it is a method on an object.
	thisAddress  *Address
	receiverType string

	registerTop int
	scopes      []int
}

func (fi *functionInfo) incRegIndex() {
	fi.registerTop++
	if fi.registerTop > fi.function.MaxRegIndex {
		fi.function.MaxRegIndex = fi.registerTop
	}
}

// NeedsClosure returns true if the function or any of its parents contains a
// register marked as a closure (it's accessed by a anonymous child function).
func (fi *functionInfo) NeedsClosure() bool {
	for fi != nil && !fi.function.IsGlobal {
		if len(fi.function.Closures) > 0 {
			return true
		}
		fi = fi.parent
	}
	return false
}

func NewCompiler() *compiler {
	program := &Program{
		Directives: make(map[string]string),
	}

	c := &compiler{
		program:      program,
		functions:    make(map[string]*functionInfo),
		builtinFuncs: builtinFuncs,
	}

	name := c.registerName("@global")
	f, err := c.addFunction(Global, name, false, ast.Position{})
	if err != nil {
		panic(err)
	}

	f.function.IsGlobal = true
	c.globalFunc = f

	return c
}

type compiler struct {
	program         *Program
	unresolved      []*unresolved
	closures        []*closure
	branches        []*branch
	initFuncs       []string
	unresolvedIndex int
	imports         []*ast.ImportStmt
	currentFunc     *functionInfo
	globalFunc      *functionInfo
	module          string // the module being compiled
	functions       map[string]*functionInfo
	builtinFuncs    []string
}

func (c *compiler) Compile(ast *ast.Module) (*Program, error) {
	for path, m := range ast.Modules {
		c.module = path
		if err := c.compileFile(m); err != nil {
			return nil, err
		}
	}

	c.module = ""
	if err := c.compileFile(ast.File); err != nil {
		return nil, err
	}

	if err := c.fixUnresolved(); err != nil {
		return nil, err
	}

	if err := c.generateInits(); err != nil {
		return nil, err
	}

	c.ensureReturn(c.globalFunc.function)

	for _, f := range c.program.Functions {
		if f.Name != "@global" {
			continue
		}
	}

	return c.program, nil
}

func (c *compiler) compileFile(file *ast.File) error {
	c.imports = file.Imports

	if err := addDirectives(c.program, file); err != nil {
		return err
	}

	if err := c.compileStmts(file.Stms); err != nil {
		return err
	}

	if err := c.setTargetOffsets(); err != nil {
		return err
	}

	return nil
}

func (c *compiler) generateInits() error {
	if len(c.initFuncs) == 0 {
		return nil
	}

	// reset the module because this goes at the end of the global function
	c.module = ""

	sort.Strings(c.initFuncs)

	for _, v := range c.initFuncs {
		fi := c.functions[v]
		c.currentFunc = c.globalFunc
		if err := c.compileGeneratedCall(fi); err != nil {
			return err
		}
	}

	return nil
}

func (c *compiler) compileGeneratedCall(fi *functionInfo) error {
	p := ast.Position{FileName: "[compiler generated]"}
	t := &ast.CallExpr{
		Ident:  &ast.IdentExpr{Name: fi.function.Name, Pos: p},
		Lparen: p,
		Rparen: p,
	}

	_, err := c.compileCallExpr(t, Void, false)
	return err
}

func (c *compiler) addFunction(kind FunctionKind, name string, anonymous bool, pos ast.Position) (*functionInfo, error) {
	if _, ok := c.functions[name]; ok {
		return nil, newError(pos, "Redeclared function '%s'", name)
	}

	f := &Function{
		Kind:  kind,
		Name:  name,
		Index: len(c.functions),
	}

	c.program.Functions = append(c.program.Functions, f)

	fi := &functionInfo{
		function:  f,
		pos:       pos,
		module:    c.module,
		parent:    c.currentFunc,
		anonymous: anonymous,
	}

	c.currentFunc = fi
	c.functions[f.Name] = fi

	return fi, nil
}

func (c *compiler) compileStmts(stms []ast.Stmt) error {
	sort.Sort(ByPriority(stms))

	for _, node := range stms {
		switch t := node.(type) {
		case *ast.VarDeclStmt:
			if err := c.compileVarDeclStmt(t); err != nil {
				return err
			}
		case *ast.FuncDeclStmt:
			if _, err := c.compileFuncDecl(t, false); err != nil {
				return err
			}
		case *ast.ClassDeclStmt:
			if err := c.compileClassDeclStmt(t); err != nil {
				return err
			}
		default:
			if stmt, ok := t.(ast.Stmt); ok {
				if err := c.compileStmt(stmt); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *compiler) compileVarDeclStmt(t *ast.VarDeclStmt) error {
	name := t.Name

	if ok, _ := c.isInScope(name); ok {
		return newError(t.Pos, "Redeclared identifier in the same block: '%s'", name)
	}

	i := c.newRegister(name, t.Exported)

	// the right hand is a expression
	if _, err := c.compileExpr(t.Value, i); err != nil {
		return err
	}

	return nil
}

func (c *compiler) compileFuncDecl(t *ast.FuncDeclStmt, isClass bool) (*functionInfo, error) {
	name := t.Name
	var kind FunctionKind

	if t.ReceiverType == "" {
		switch name {
		case "init":
			if len(t.Args.List) > 0 {
				return nil, newError(t.Pos, "init functions can't receive arguments.")
			}

			if t.Exported {
				return nil, newError(t.Pos, "init functions can't be exported.")
			}

			// init functions are renamed to allow multiple versions
			// They can't be called directly so it doesn't matter the renaming.
			name = "@init"
			c.initFuncs = append(c.initFuncs, c.registerName(name))
			kind = Init

		case "main":
			kind = Main
		}
	}

	// if is a class function it is already prefixed
	if !isClass {
		name = c.registerName(name)
	}

	fi, err := c.addFunction(kind, name, t.Anonymous, t.Pos)
	if err != nil {
		return nil, err
	}

	fi.receiverType = t.ReceiverType

	f := fi.function
	f.Arguments = len(t.Args.List)
	f.Variadic = t.Variadic
	f.Exported = t.Exported

	if !fi.anonymous {
		// restart closures references when is a top function
		c.closures = nil
	}

	// reset
	targets := c.branches
	c.branches = nil

	if err := c.compileFuncBody(t, fi); err != nil {
		return nil, err
	}

	c.branches = targets
	return fi, nil
}

func (c *compiler) compileFuncBody(t *ast.FuncDeclStmt, fi *functionInfo) error {
	c.openScope()

	// Create first the arguments because when the function is called
	// they are copied directly to the beginning of the values.
	for _, arg := range t.Args.List {
		c.newRegister(arg.Name, false)
	}

	// if it is a method reserve a register for the "this" object.
	// but *after* params.
	if fi.receiverType != "" {
		fi.thisAddress = c.newRegister("this", false)
	}

	// don't open a block because the arguments are declared in the current scope
	if err := c.compileBlockStmtScope(t.Body); err != nil {
		return err
	}

	// make sure that the last instruction is a return
	c.ensureReturn(fi.function)

	c.closeScope()

	// restore compiler function
	c.currentFunc = fi.parent
	if c.currentFunc == nil {
		c.currentFunc = c.globalFunc
	}

	if fi.parent.function.IsGlobal {
		// we can't know in advance the index in the global array of closures
		// because previous registers can be referenced later and thus marked
		// as closure so keep a index (ix) in the compiler compiler and after
		// compiled the top function, update all now.
		c.updateClosureIndexes(fi)
	}

	if err := c.setTargetOffsets(); err != nil {
		return err
	}

	return nil
}

func (c *compiler) ensureReturn(f *Function) {
	// ln := len(f.Instructions)
	// lastRet := ln > 0 && f.Instructions[ln-1].Opcode == op_ret
	// if !lastRet {
	// 	c.emit(op_ret, Void, Void, Void, ast.Position{})
	// }
	c.emit(op_ret, Void, Void, Void, ast.Position{})
}

func (c *compiler) compileBlockStmt(t *ast.BlockStmt) error {
	c.openScope()
	if err := c.compileBlockStmtScope(t); err != nil {
		return err
	}
	c.closeScope()
	return nil
}

func (c *compiler) compileBlockStmtScope(t *ast.BlockStmt) error {
	for _, stmt := range t.List {
		if err := c.compileStmt(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (c *compiler) compileStmt(stmt ast.Stmt) error {
	switch t := stmt.(type) {
	case *ast.VarDeclStmt:
		if err := c.compileVarDeclStmt(t); err != nil {
			return err
		}
	case *ast.AsignStmt:
		if err := c.compileAsignStmt(t); err != nil {
			return err
		}
	case *ast.IncStmt:
		if err := c.compileIncStmt(t); err != nil {
			return err
		}
	case *ast.FuncDeclStmt:
		if _, err := c.compileFuncDecl(t, false); err != nil {
			return err
		}
	case *ast.ClassDeclStmt:
		if err := c.compileClassDeclStmt(t); err != nil {
			return err
		}
	case *ast.CallStmt:
		if _, err := c.compileCallExpr(t.CallExpr, Void, false); err != nil {
			return err
		}
	case *ast.ReturnStmt:
		if err := c.compileReturnStmt(t); err != nil {
			return err
		}
	case *ast.BlockStmt:
		if err := c.compileBlockStmt(t); err != nil {
			return err
		}
	case *ast.IfStmt:
		if err := c.compileIfStmt(t); err != nil {
			return err
		}
	case *ast.SwitchStmt:
		if err := c.compileSwitchStmt(t); err != nil {
			return err
		}
	case *ast.ForStmt:
		if err := c.compileForStmt(t); err != nil {
			return err
		}
	case *ast.WhileStmt:
		if err := c.compileWhileStmt(t); err != nil {
			return err
		}
	case *ast.ThrowStmt:
		if err := c.compileThrowStmt(t); err != nil {
			return err
		}
	case *ast.TryStmt:
		if err := c.compileTryStmt(t); err != nil {
			return err
		}
	case *ast.BreakStmt:
		if err := c.compileBreakStmt(t); err != nil {
			return err
		}
	case *ast.ContinueStmt:
		if err := c.compileContinueStmt(t); err != nil {
			return err
		}
	default:
		return newError(t.Position(), "stmt not implemented: %T", t)
	}
	return nil
}

func (c *compiler) canContinueOrBreak() string {
	l := len(c.branches)
	if l == 0 {
		return "Unexpected exit"
	}

	for i := len(c.branches) - 1; i >= 0; i-- {
		b := c.branches[i]
		if b.outOfScope {
			continue
		}
		if b.stmt != nil {
			return ""
		}
		if b.inFinally {
			return "Control cannot leave the body of a finally clause"
		}
	}

	return ""
}

func (c *compiler) compileContinueStmt(t *ast.ContinueStmt) error {
	if errMsg := c.canContinueOrBreak(); errMsg != "" {
		return newError(t.Pos, errMsg)
	}

	var i int
	var target *branch

	for i = len(c.branches) - 1; i >= 0; i-- {
		b := c.branches[i]
		if !b.isValidTarget(t.Label) {
			continue
		}
		if _, ok := b.stmt.(continueTarget); ok {
			target = b
			break
		}
	}

	if target == nil {
		return newError(t.Pos, "Invalid continue")
	}

	c.compileBranchExit(c.branches[i:], t)

	cont := c.emit(op_jpb, Void, Void, Void, t.Pos)
	target.continues = append(target.continues, &jumpInstr{cont, c.pc() - 1, target.stmt})
	return nil
}

func (c *compiler) compileBreakStmt(t *ast.BreakStmt) error {
	if errMsg := c.canContinueOrBreak(); errMsg != "" {
		return newError(t.Pos, errMsg)
	}

	var i int
	var target *branch

	for i = len(c.branches) - 1; i >= 0; i-- {
		b := c.branches[i]
		if !b.isValidTarget(t.Label) {
			continue
		}
		if _, ok := b.stmt.(breakTarget); ok {
			target = b
			break
		}
	}

	if target == nil {
		return newError(t.Pos, "Invalid break")
	}

	c.compileBranchExit(c.branches[i:], t)

	brk := c.emit(op_jmp, Void, Void, Void, t.Pos)
	target.breaks = append(target.breaks, &jumpInstr{brk, c.pc() - 1, target.stmt})
	return nil
}

func (c *compiler) compileBranchExit(targets []*branch, branch ast.Stmt) {
	pos := branch.Position()

	for j := len(targets) - 1; j >= 0; j-- {
		target := targets[j]
		if target.outOfScope {
			continue
		}
		// we must jump to finally or just remove the try list before continuing
		for i := range target.tryCatchs {
			if j == 0 && i == 0 && target.inFinally {
				c.emit(op_fen, Void, Void, Void, pos)
			} else {
				c.emit(op_trx, Void, Void, Void, pos)
			}
		}
	}
}

func (c *compiler) compileForStmt(t *ast.ForStmt) error {
	// del tipo "for (let k in items) {}"
	if t.InExpression != nil {
		return c.compileForInOfStmt(t, true)
	}

	// del tipo "for (let k of items) {}"
	if t.OfExpression != nil {
		return c.compileForInOfStmt(t, false)
	}

	if t.Declaration == nil && t.Expression == nil && t.Step == nil {
		// del tipo: "for (;;) {}"
		return c.compileForWithNoExpression(t)
	}

	// del tipo: "for (let i=0; i < 10; i++) {}"
	return c.compileForWithStepStmt(t)
}

func (c *compiler) compileForInOfStmt(t *ast.ForStmt, in bool) error {
	c.openBranch(t)
	c.openScope()

	if len(t.Declaration) != 1 {
		return newError(t.Position(), "Invalid range declaration. Only one variable is valid.")
	}

	// get the range declaration
	dec, ok := t.Declaration[0].(*ast.VarDeclStmt)
	if !ok {
		return newError(t.Position(), "Invalid range declaration")
	}

	// compile the expression that must return a map or array
	var rng *Address
	var err error
	if in {
		if rng, err = c.compileExpr(t.InExpression, Void); err != nil {
			return err
		}
	} else {
		if rng, err = c.compileExpr(t.OfExpression, Void); err != nil {
			return err
		}
	}

	// this is the key variable
	key := c.newRegister(dec.Name, false)

	// create a temp array with the keys/index or values
	items := c.newTempRegister()

	if in {
		c.emit(op_key, items, rng, Void, dec.Pos)
	} else {
		c.emit(op_val, items, rng, Void, dec.Pos)
	}

	// get the length of the keys/index
	iLen := c.newTempRegister()
	c.emit(op_len, iLen, items, Void, ast.Position{})

	// create the counter
	counter := c.newTempRegister()

	// initialize it with 0
	k := c.program.addConstant(NewInt(0))
	c.emit(op_ldk, counter, k, Void, ast.Position{})

	// the register that will hold the condition to continue
	isLess := c.newTempRegister()

	// skip the first step increment
	c.emit(op_jmp, NewAddress(AddrData, 1), Void, Void, ast.Position{})

	// this is start point where it needs to return each iteration
	loopStart := c.pc()
	t.SetContinuePC(loopStart)

	// the step increment part of the for
	c.emit(op_inc, counter, Void, Void, ast.Position{})

	// set in isLess if counter < keys/values length
	c.emit(op_lst, isLess, counter, iLen, ast.Position{})

	bodyStart := c.pc()

	// conditional jump: test R(A) and jump R(B) instructions. R(C)=1 means jump if false
	loopBrk := c.emit(op_tjp, isLess, Void, NewAddress(AddrData, 1), ast.Position{})

	// assign the key
	c.emit(op_get, key, items, counter, ast.Position{})

	// the body of the loop
	if err := c.compileBlockStmt(t.Body); err != nil {
		return err
	}

	// jump back to iterate
	steps := NewAddress(AddrData, c.pc()-loopStart)
	c.emit(op_jpb, steps, Void, Void, ast.Position{})

	bodyEnd := c.pc()
	t.SetBreakPC(bodyEnd)

	// set the offset to jump when the condition for the loop fails
	loopBrk.B = NewAddress(AddrData, bodyEnd-bodyStart-1)

	c.closeScope()
	c.closeBranch()

	return nil
}

// del tipo "for next() {}"
func (c *compiler) compileWhileStmt(t *ast.WhileStmt) error {
	c.openBranch(t)
	c.openScope()

	// this is start point where it needs to return each iteration
	loopStart := c.pc()
	t.SetContinuePC(loopStart)

	// the expression part of the for
	r, err := c.compileExpr(t.Expression, Void)
	if err != nil {
		return err
	}

	bodyStart := c.pc()

	// condition to continue looping. Will set later the jump length
	// Set R(C) to 1 to make it jump if R(A) is false.
	loopBrk := c.emit(op_tjp, r, Void, NewAddress(AddrData, 1), t.Pos)

	// the body of the loop
	if err := c.compileBlockStmt(t.Body); err != nil {
		return err
	}

	// jump back to iterate
	steps := c.pc() - loopStart
	c.emit(op_jpb, NewAddress(AddrData, steps), Void, Void, ast.Position{})

	bodyEnd := c.pc()
	t.SetBreakPC(bodyEnd)

	// set the offset to jump when the condition for the loop fails
	loopBrk.B = NewAddress(AddrData, bodyEnd-bodyStart-1)

	c.closeScope()
	c.closeBranch()

	return nil
}

func (c *compiler) setTargetOffsets() error {
	for _, t := range c.branches {
		for _, b := range t.breaks {
			targetPC := b.target.(breakTarget).BreakPC()
			offset := targetPC - b.pc - 1
			if offset < 0 {
				panic("Invalid break offset")
			}
			b.inst.A = NewAddress(AddrData, offset)
		}

		for _, cont := range t.continues {
			targetPC := cont.target.(continueTarget).ContinuePC()
			offset := cont.pc - targetPC
			if offset < 0 {
				panic("Invalid continue offset")
			}
			cont.inst.A = NewAddress(AddrData, offset)
		}
	}
	return nil
}

func (c *compiler) updateClosureIndexes(fi *functionInfo) {
	if len(c.closures) == 0 {
		return
	}

	pr := c.program
	fIndex := fi.function.Index

loop:
	// first set the global closure index (when a inner function is called
	// it copies its closures to the previous ones creating a global array of
	// all closures in scope).
	for _, cl := range c.closures {
		var count int
		for i, l := fIndex, len(pr.Functions); i < l; i++ {
			f := pr.Functions[i]
			if f == cl.fn {
				for _, j := range f.Closures {
					if j.Equals(cl.reg) {
						cl.index = count
						continue loop
					}
					count++
				}
			} else {
				count += len(f.Closures)
			}
		}
	}

	// Now update the closure registers to point to the correct index
	for i, l := fIndex+1, len(pr.Functions); i < l; i++ {
		f := pr.Functions[i]

		for _, inst := range f.Instructions {
			if inst.A.Kind == AddrClosure {
				inst.A = NewAddress(AddrClosure, c.closures[inst.A.Value].index)
			}
			if inst.B.Kind == AddrClosure {
				inst.B = NewAddress(AddrClosure, c.closures[inst.B.Value].index)
			}
			if inst.C.Kind == AddrClosure {
				inst.C = NewAddress(AddrClosure, c.closures[inst.C.Value].index)
			}
		}
	}
}

func (c *compiler) compileThrowStmt(s *ast.ThrowStmt) error {
	e, err := c.compileExpr(s.Value, Void)
	if err != nil {
		return err
	}

	c.emit(op_trw, e, Void, Void, s.Pos)
	return nil
}

func (c *compiler) compileTryStmt(t *ast.TryStmt) error {
	// op_try jump to R(A) absolute pc, set the error to R(B). R(C) 'finally' abs pc.

	try := c.emit(op_try, Void, Void, Void, t.Pos)

	c.openScope()
	b := c.openBranch(nil)

	b.tryCatchs = append(b.tryCatchs, t)

	if err := c.compileBlockStmt(t.Body); err != nil {
		return err
	}

	if t.Finally == nil {
		// only emit a try end (TRE) instruction if there is no finally because
		// finally emits finally end (FEN)
		c.emit(op_tre, Void, Void, Void, ast.Position{})
	}

	// this jump skips the catch block. This is at the end so it
	// only gets here if no exception is thrown
	jump := c.emit(op_jmp, Void, Void, Void, ast.Position{})

	start := c.pc()

	if t.CatchIdent != nil {
		try.B = c.newRegister(t.CatchIdent.Name, true)

		// make the err register on scope from the beginning of the current scope
		regs := c.currentFunc.function.Registers
		r := regs[len(regs)-1]
		r.StartPC = 0
	}

	if t.Catch != nil {
		try.A = NewAddress(AddrData, start)

		if err := c.compileBlockStmt(t.Catch); err != nil {
			return err
		}

		if t.Finally == nil {
			c.emit(op_cen, Void, Void, Void, ast.Position{})
		}
	}

	jump.A = NewAddress(AddrData, c.pc()-start)

	if t.Finally != nil {
		b.inFinally = true

		try.C = NewAddress(AddrData, c.pc())

		if err := c.compileBlockStmt(t.Finally); err != nil {
			return err
		}

		c.emit(op_fen, Void, Void, Void, ast.Position{})

		b.inFinally = false
	}

	c.closeBranch()
	c.closeScope()

	return nil
}

// del tipo: "for i:=0; i < 10; i++ {}"
func (c *compiler) compileForWithStepStmt(t *ast.ForStmt) error {
	c.openBranch(t)
	c.openScope()

	// the declaration part of the for
	for _, dec := range t.Declaration {
		if err := c.compileStmt(dec); err != nil {
			return err
		}
	}

	// skip the first step increment
	firstj := c.emit(op_jmp, Void, Void, Void, ast.Position{})

	// this is start point where it needs to return each iteration
	loopStart := c.pc()
	t.SetContinuePC(loopStart)

	if t.Step != nil {
		// the step increment part of the for
		if err := c.compileStmt(t.Step); err != nil {
			return err
		}
	}

	firstj.A = NewAddress(AddrData, c.pc()-loopStart)

	// the expression part of the for
	r, err := c.compileExpr(t.Expression, Void)
	if err != nil {
		return err
	}

	bodyStart := c.pc()

	// condition to continue looping. Will set later the jump length
	// Set R(C) to 1 to make it jump if R(A) is false.
	loopBrk := c.emit(op_tjp, r, Void, NewAddress(AddrData, 1), ast.Position{})

	// the body of the loop
	if err := c.compileBlockStmt(t.Body); err != nil {
		return err
	}

	// jump back to iterate
	steps := c.pc() - loopStart
	c.emit(op_jpb, NewAddress(AddrData, steps), Void, Void, ast.Position{})

	pc := c.pc()
	t.SetBreakPC(pc)

	// set the offset to jump when the condition for the loop fails
	loopBrk.B = NewAddress(AddrData, pc-bodyStart-1)

	c.closeScope()
	c.closeBranch()

	return nil
}

func (c *compiler) compileForWithNoExpression(t *ast.ForStmt) error {
	c.openBranch(t)

	// this is start point where it needs to return each iteration
	bodyStart := c.pc()
	t.SetContinuePC(bodyStart)

	// the body of the loop
	if err := c.compileBlockStmt(t.Body); err != nil {
		return err
	}

	// jump back to iterate
	steps := c.pc() - bodyStart
	c.emit(op_jpb, NewAddress(AddrData, steps), Void, Void, ast.Position{})

	t.SetBreakPC(c.pc())

	c.closeBranch()
	return nil
}

func (c *compiler) compileSwitchStmt(t *ast.SwitchStmt) error {
	c.openBranch(t)

	a, err := c.compileExpr(t.Expression, Void)
	if err != nil {
		return err
	}

	fallThroughs := make(map[int]*Instruction)

	const FALLTHROUGH_ERR = "Fallthrough is only allowed in empty case"

	for _, block := range t.Blocks {
		if !endsWithExit(block) {
			return newError(block.Pos, FALLTHROUGH_ERR)
		}

		b, err := c.compileExpr(block.Expression, Void)
		if err != nil {
			return err
		}

		if len(block.List) == 0 {
			fallThroughs[c.pc()+1] = c.emit(op_ejp, a, b, Void, block.Pos)
			continue
		}

		// jump to the next case if false
		jump := c.emit(op_djp, a, b, Void, block.Pos)

		// update falltroughs
		if len(fallThroughs) > 0 {
			totalLen := c.pc()
			for k, i := range fallThroughs {
				i.C = NewAddress(AddrData, totalLen-k)
			}
			// reset for the next case
			fallThroughs = make(map[int]*Instruction)
		}

		if err := c.compileCaseBlock(t, block, jump); err != nil {
			return err
		}
	}

	if t.Default != nil {
		if !endsWithExit(t.Default) {
			return newError(t.Default.Pos, FALLTHROUGH_ERR)
		}
		if err := c.compileCaseBlock(t, t.Default, nil); err != nil {
			return err
		}
	}

	t.SetBreakPC(c.pc())

	c.closeBranch()
	return nil
}

func endsWithExit(block *ast.CaseBlock) bool {
	ln := len(block.List)
	if ln == 0 {
		return true
	}
	last := block.List[ln-1]
	switch last.(type) {
	case *ast.BreakStmt, *ast.ContinueStmt, *ast.ReturnStmt, *ast.ThrowStmt:
		return true
	}
	return false
}

func (c *compiler) compileCaseBlock(t *ast.SwitchStmt, block *ast.CaseBlock, jump *Instruction) error {
	// this is start point where it needs to return each iteration
	bodyStart := c.pc()

	c.openScope()

	for _, stmt := range block.List {
		if err := c.compileStmt(stmt); err != nil {
			return err
		}
	}

	c.closeScope()

	if jump != nil {
		// jump to the next CASE.
		jump.C = NewAddress(AddrData, c.pc()-bodyStart)
	}

	return nil
}

func (c *compiler) compileIfStmt(t *ast.IfStmt) error {
	if len(t.IfBlocks) == 1 && t.Else == nil {
		// produce simpler code if there are no elses
		return c.compileIfBlockStmt(t)
	}

	exits := make(map[int]*Instruction)

	for _, branch := range t.IfBlocks {
		// the expression part of the for
		r, err := c.compileExpr(branch.Condition, Void)
		if err != nil {
			return err
		}

		// if its true dont execute the next jump
		c.emit(op_tjp, r, NewAddress(AddrData, 1), Void, ast.Position{})

		jump := c.emit(op_jmp, Void, Void, Void, ast.Position{})

		// this is start point where it needs to return each iteration
		bodyStart := c.pc()

		// the body of the loop
		if err := c.compileBlockStmt(branch.Body); err != nil {
			return err
		}

		// save the position of the exit instruction to assign the jump
		// length after al the statement is processed
		exits[c.pc()+1] = c.emit(op_jmp, Void, Void, Void, ast.Position{})

		//jump to the nexto branch if the condition fails
		jump.A = NewAddress(AddrData, c.pc()-bodyStart)
	}

	if t.Else != nil {
		if err := c.compileBlockStmt(t.Else); err != nil {
			return err
		}
	}

	// now that all is processed, set the exit jumps
	totalLen := c.pc()
	for k, i := range exits {
		i.A = NewAddress(AddrData, totalLen-k)
	}

	return nil
}

func (c *compiler) compileIfBlockStmt(t *ast.IfStmt) error {
	branch := t.IfBlocks[0]

	// the expression part of the for
	r, err := c.compileExpr(branch.Condition, Void)
	if err != nil {
		return err
	}

	// op_tjp: test R(A) and jump R(B) instructions. R(C)=(0=jump if true, 1 jump if false)

	// if its false jump to the end
	jump := c.emit(op_tjp, r, NewAddress(AddrData, 1), NewAddress(AddrData, 1), t.Pos)

	// this is start point where it needs to return each iteration
	bodyStart := c.pc()

	// the body of the loop
	if err := c.compileBlockStmt(branch.Body); err != nil {
		return err
	}

	// set how long is the jump
	jump.B = NewAddress(AddrData, c.pc()-bodyStart)

	return nil
}

// compileExpr compiles the expression and stores the result in dest
func (c *compiler) compileExpr(t ast.Expr, dest *Address) (*Address, error) {
	switch t := t.(type) {
	case *ast.ConstantExpr:
		return c.compileConstantExpr(t, dest)
	case *ast.UnaryExpr:
		return c.compileUnaryExpr(t, dest)
	case *ast.BinaryExpr:
		return c.compileBinaryExpr(t, dest)
	case *ast.TernaryExpr:
		return c.compileTernaryExpr(t, dest)
	case *ast.IdentExpr:
		return c.compileIdentExpr(t, dest)
	case *ast.MapDeclExpr:
		return c.compileMapDeclExpr(t, dest)
	case *ast.ArrayDeclExpr:
		return c.compileArrayDeclExpr(t, dest)
	case *ast.IndexExpr:
		return c.compileIndexExpr(t, dest)
	case *ast.SelectorExpr:
		return c.compileSelectorExpr(t, dest)
	case *ast.FuncDeclExpr:
		return c.compileFuncDeclExpr(t, dest)
	case *ast.CallExpr:
		return c.compileCallExpr(t, dest, true)
	case *ast.NewInstanceExpr:
		return c.compileNewInstanceExpr(t, dest)
	default:
		panic(fmt.Sprintf("not implemented: %T", t))
	}
}

func (c *compiler) compileFuncDeclExpr(t *ast.FuncDeclExpr, dest *Address) (*Address, error) {
	// get the function address
	i := len(c.program.Functions)

	stmt := &ast.FuncDeclStmt{
		Pos:       t.Pos,
		Name:      fmt.Sprintf("@lambda_%d", i),
		Anonymous: true,
		Args:      t.Args,
		Body:      t.Body,
	}

	fi, err := c.compileFuncDecl(stmt, false)
	if err != nil {
		return Void, err
	}

	// if it needs to create a closure
	if fi.NeedsClosure() {
		if dest == Void {
			dest = c.newTempRegister()
		}
		c.emit(op_clo, dest, NewAddress(AddrFunc, i), Void, t.Position())
		return dest, nil
	}

	if dest == Void {
		return NewAddress(AddrFunc, i), nil
	}

	c.emit(op_mov, dest, NewAddress(AddrFunc, i), Void, t.Position())
	return dest, nil
}

// TODO merge the three: compile expression and inc the result
func (c *compiler) compileIncStmt(t *ast.IncStmt) error {
	switch s := t.Left.(type) {
	case *ast.IdentExpr:
		return c.compileIncIdentExpr(t)
	case *ast.SelectorExpr:
		return c.compileIncSelectorExpr(s, t)
	case *ast.IndexExpr:
		return c.compileIncIndexExpr(s, t)
	default:
		return newError(t.Position(), "Invalid inc/dec statement")
	}
}

func (c *compiler) compileIncIndexExpr(s *ast.IndexExpr, t *ast.IncStmt) error {
	// get the array address
	x, err := c.compileExpr(s.Left, Void)
	if err != nil {
		return err
	}

	// get array index
	i, err := c.compileExpr(s.Index, Void)
	if err != nil {
		return err
	}

	// get the value
	v := c.newTempRegister()
	c.emit(op_get, v, x, i, t.Position())

	// increment/decrement the value
	switch t.Operator {
	case ast.INC:
		c.emit(op_inc, v, Void, Void, t.Position())
	case ast.DEC:
		c.emit(op_dec, v, Void, Void, t.Position())
	default:
		return newError(t.Position(), "Invalid operator %s, expected ++ or --", t.Operator)
	}

	// set the value again
	c.emit(op_set, x, i, v, s.Position())
	return err
}

func (c *compiler) compileIncSelectorExpr(s *ast.SelectorExpr, t *ast.IncStmt) error {
	// get the map address
	x, err := c.compileExpr(s.X, Void)
	if err != nil {
		return err
	}

	// get the key. Its a constant string
	key := c.program.addConstant(NewString(s.Sel.Name))

	// get the value
	v := c.newTempRegister()
	c.emit(op_get, v, x, key, t.Position())

	// increment/decrement the value
	switch t.Operator {
	case ast.INC:
		c.emit(op_inc, v, Void, Void, t.Position())
	case ast.DEC:
		c.emit(op_dec, v, Void, Void, t.Position())
	default:
		return newError(t.Position(), "Invalid operator %s, expected ++ or --", t.Operator)
	}

	// set the value again
	c.emit(op_set, x, key, v, s.Position())
	return err
}

func (c *compiler) compileIncIdentExpr(t *ast.IncStmt) error {
	i, err := c.compileExpr(t.Left, Void)
	if err != nil {
		return err
	}

	switch t.Operator {
	case ast.INC:
		c.emit(op_inc, i, Void, Void, t.Position())
	case ast.DEC:
		c.emit(op_dec, i, Void, Void, t.Position())
	default:
		return newError(t.Position(), "Invalid operator %s, expected ++ or --", t.Operator)
	}

	return err
}

func (c *compiler) compileAsignStmt(t *ast.AsignStmt) error {
	switch s := t.Left.(type) {
	case *ast.IdentExpr:
		return c.compileAsignIdentExpr(t)
	case *ast.SelectorExpr:
		return c.compileAsignSelectorExpr(s, t)
	case *ast.IndexExpr:
		return c.compileAsignIndexExpr(s, t)
	default:
		return newError(t.Position(), "Invalid asign")
	}
}

func (c *compiler) compileAsignIndexExpr(s *ast.IndexExpr, t *ast.AsignStmt) error {
	// get the array address
	x, err := c.compileExpr(s.Left, Void)
	if err != nil {
		return err
	}

	// get array index
	i, err := c.compileExpr(s.Index, Void)
	if err != nil {
		return err
	}

	// compile the value
	val, err := c.compileExpr(t.Value, Void)
	if err != nil {
		return err
	}

	// set the map value
	c.emit(op_set, x, i, val, s.Position())
	return err
}

func (c *compiler) compileAsignSelectorExpr(s *ast.SelectorExpr, t *ast.AsignStmt) error {
	// get the map address
	x, err := c.compileExpr(s.X, Void)
	if err != nil {
		return err
	}

	// get the key. Its a constant string
	key := c.program.addConstant(NewString(s.Sel.Name))

	// compile the value
	val, err := c.compileExpr(t.Value, Void)
	if err != nil {
		return err
	}

	// set the map value
	c.emit(op_set, x, key, val, s.Position())
	return err
}

func (c *compiler) compileAsignIdentExpr(t *ast.AsignStmt) error {
	left, err := c.compileExpr(t.Left, Void)
	if err != nil {
		return err
	}

	_, err = c.compileExpr(t.Value, left)
	return err
}

func (c *compiler) compileReturnStmt(s *ast.ReturnStmt) error {
	var retIndex *Address

	// if there is a return value compile it
	if s.Value != nil {
		var err error
		retIndex, err = c.compileExpr(s.Value, Void)
		if err != nil {
			return err
		}
	} else {
		retIndex = Void
	}

	c.emit(op_ret, retIndex, Void, Void, s.Pos)
	return nil
}

func (c *compiler) compileIdentExpr(t *ast.IdentExpr, dest *Address) (*Address, error) {
	i, err := c.findRegister(t.Name, c.currentFunc)
	if err != nil {
		return Void, err
	}

	if i == Void {
		i = c.getUnresolved(t.Name, t.Pos)
	}

	if dest != Void {
		c.emit(op_mov, dest, i, Void, t.Pos)
		return dest, nil
	}

	return i, nil
}

func (c *compiler) compileMapDeclExpr(t *ast.MapDeclExpr, dest *Address) (*Address, error) {
	if dest == Void {
		dest = c.newTempRegister()
	}

	c.emit(op_map, dest, NewAddress(AddrData, len(t.List)), Void, t.Pos)

	for _, kv := range t.List {
		// the key is a constant
		k := c.program.addConstant(NewString(kv.Key))

		// the value is an expression.
		exp, err := c.compileExpr(kv.Value, Void)
		if err != nil {
			return Void, err
		}

		// copy the value to the map
		c.emit(op_set, dest, k, exp, ast.Position{})
	}

	return dest, nil
}

func (c *compiler) compileIndexExpr(t *ast.IndexExpr, dest *Address) (*Address, error) {
	left, err := c.compileExpr(t.Left, Void)
	if err != nil {
		return Void, err
	}

	index, err := c.compileExpr(t.Index, Void)
	if err != nil {
		return Void, err
	}

	if dest == Void {
		dest = c.newTempRegister()
	}

	// move the value from the array to the dest register
	c.emit(op_get, dest, left, index, t.Left.Position())

	return dest, nil
}

func (c *compiler) compileArrayDeclExpr(t *ast.ArrayDeclExpr, dest *Address) (*Address, error) {
	if dest == Void {
		dest = c.newTempRegister()
	}

	c.emit(op_arr, dest, NewAddress(AddrData, len(t.List)), Void, t.Pos)

	for i, kv := range t.List {
		// the value is an expression.
		exp, err := c.compileExpr(kv, Void)
		if err != nil {
			return Void, err
		}

		// copy the value to the map
		c.emit(op_set, dest, NewAddress(AddrData, i), exp, t.Pos)
	}

	return dest, nil
}

func (c *compiler) compileNewInstanceExpr(t *ast.NewInstanceExpr, dest *Address) (*Address, error) {
	var addr *Address
	var err error

	switch tp := t.Name.(type) {
	case *ast.IdentExpr:
		addr = c.program.addConstant(NewString(tp.Name))

	case *ast.SelectorExpr:
		ident, ok := tp.X.(*ast.IdentExpr)
		if !ok {
			return Void, newError(tp.Position(), "Expected class name")
		}
		addr, err = c.compileModuleExpr(ident.Name, tp.Sel.Name, dest, tp.Position())
		if err != nil {
			return Void, err
		}
		if addr == Void {
			return Void, newError(tp.Position(), "Expected class name")
		}

	default:
		return Void, newError(tp.Position(), "Expected class name")
	}

	if dest == Void {
		dest = c.newTempRegister()
	}

	if !t.Spread && len(t.Args) == 1 {
		exp, err := c.compileExpr(t.Args[0], Void)
		if err != nil {
			return Void, err
		}

		// call: R(A) funcIndex, R(B) retAddress, R(C) argsAddress
		c.emit(op_nes, addr, dest, exp, t.Position())
		return dest, nil
	}

	args, err := c.compileCallArgs(t.Args, t.Spread)
	if err != nil {
		return Void, err
	}

	// call: R(A) funcIndex, R(B) retAddress, R(C) argsAddress
	c.emit(op_new, addr, dest, args, t.Position())

	return dest, nil
}

func (c *compiler) compileTernaryExpr(t *ast.TernaryExpr, dest *Address) (*Address, error) {
	if dest == Void {
		dest = c.newTempRegister()
	}

	condition, err := c.compileExpr(t.Condition, Void)
	if err != nil {
		return Void, err
	}

	// op_tjp: test if true or not null and jump: test R(A) and jump R(B) instructions. R(C)=(0=jump if true, 1 jump if false)
	jump := c.emit(op_tjp, condition, Void, NewAddress(AddrData, 1), t.Left.Position())

	pc := c.pc()

	if _, err = c.compileExpr(t.Left, dest); err != nil {
		return Void, err
	}

	jumpLeft := c.emit(op_jmp, Void, Void, Void, ast.Position{})
	leftPC := c.pc()

	jump.B = NewAddress(AddrData, c.pc()-pc)

	if _, err = c.compileExpr(t.Right, dest); err != nil {
		return Void, err
	}

	jumpLeft.A = NewAddress(AddrData, c.pc()-leftPC)

	return dest, nil
}

func (c *compiler) compileAndOrExpr(t *ast.BinaryExpr, and bool, dest *Address) (*Address, error) {
	left, err := c.compileExpr(t.Left, Void)
	if err != nil {
		return Void, err
	}

	if dest == Void {
		dest = c.newTempRegister()
	}

	leftSet := c.newTempRegister()

	// set if left is true or has a value
	c.emit(op_mob, dest, left, leftSet, t.Left.Position())

	var x *Address

	if and {
		// AND: if not true jump (Both need to be true so its impossible to be true, don't need to execute the second)
		x = NewAddress(AddrData, 1)
	} else {
		// OR: if true jump (just the first is enough, don't need to execute the second)
		x = NewAddress(AddrData, 0)
	}

	// op_tjp: test if true or not null and jump: test R(A) and jump R(B) instructions. R(C)=(0=jump if true, 1 jump if false)
	jump := c.emit(op_tjp, leftSet, Void, x, t.Left.Position())

	start := c.pc()

	right, err := c.compileExpr(t.Right, Void)
	if err != nil {
		return Void, err
	}

	// set if left is true or has a value
	c.emit(op_mov, dest, right, Void, t.Left.Position())

	// set the number of jumps for the right hand
	jump.B = NewAddress(AddrData, c.pc()-start)

	return dest, nil
}

func (c *compiler) compileBinaryExpr(t *ast.BinaryExpr, dest *Address) (*Address, error) {
	switch t.Operator {
	case ast.LAND:
		return c.compileAndOrExpr(t, true, dest)
	case ast.LOR:
		return c.compileAndOrExpr(t, false, dest)
	}

	left, err := c.compileExpr(t.Left, Void)
	if err != nil {
		return Void, err
	}

	right, err := c.compileExpr(t.Right, Void)
	if err != nil {
		return Void, err
	}

	if dest == Void {
		dest = c.newTempRegister()
	}

	switch t.Operator {
	case ast.ADD:
		c.emit(op_add, dest, left, right, t.Left.Position())
	case ast.SUB:
		c.emit(op_sub, dest, left, right, t.Left.Position())
	case ast.MUL:
		c.emit(op_mul, dest, left, right, t.Left.Position())
	case ast.BOR:
		c.emit(op_bor, dest, left, right, t.Left.Position())
	case ast.AND:
		c.emit(op_and, dest, left, right, t.Left.Position())
	case ast.LSH:
		c.emit(op_lsh, dest, left, right, t.Left.Position())
	case ast.RSH:
		c.emit(op_rsh, dest, left, right, t.Left.Position())
	case ast.XOR:
		c.emit(op_xor, dest, left, right, t.Left.Position())
	case ast.DIV:
		c.emit(op_div, dest, left, right, t.Left.Position())
	case ast.MOD:
		c.emit(op_mod, dest, left, right, t.Left.Position())
	case ast.LSS:
		c.emit(op_lst, dest, left, right, t.Left.Position())
	case ast.LEQ:
		c.emit(op_lse, dest, left, right, t.Left.Position())
	case ast.GTR:
		c.emit(op_lst, dest, right, left, t.Left.Position())
	case ast.GEQ:
		c.emit(op_lse, dest, right, left, t.Left.Position())
	case ast.EQL:
		c.emit(op_eql, dest, right, left, t.Left.Position())
	case ast.NEQ:
		c.emit(op_neq, dest, right, left, t.Left.Position())
	case ast.SEQ:
		c.emit(op_seq, dest, right, left, t.Left.Position())
	case ast.SNE:
		c.emit(op_sne, dest, right, left, t.Left.Position())
	default:
		return Void, newError(t.Position(), "Unknown operator %s", t.Operator)
	}

	return dest, nil
}

func (c *compiler) compileUnaryExpr(t *ast.UnaryExpr, dest *Address) (*Address, error) {
	// if it  is a constant calculate the value and store the result constant
	if k, ok := t.Operand.(*ast.ConstantExpr); ok {
		return c.compileUnaryConstantExpr(t.Operator, k, dest)
	}

	i, err := c.compileExpr(t.Operand, Void)
	if err != nil {
		return Void, err
	}

	if dest == Void {
		dest = c.newTempRegister()
	}

	switch t.Operator {
	case ast.SUB:
		c.emit(op_unm, dest, i, Void, t.Pos)
	case ast.NOT:
		c.emit(op_not, dest, i, Void, t.Pos)
	case ast.BNT:
		c.emit(op_bnt, dest, i, Void, t.Pos)
	default:
		return Void, newError(t.Pos, "Invalid unary operator %s", t.Operator)
	}

	return dest, nil
}

func (c *compiler) compileUnaryConstantExpr(operator ast.Type, t *ast.ConstantExpr, dest *Address) (*Address, error) {
	var k *Address

	switch t.Kind {
	case ast.INT:
		n, err := strconv.ParseInt(t.Value, 10, 64)
		if err != nil {
			return Void, newError(t.Pos, "Invalid int value %s", t.Value)
		}
		switch operator {
		case ast.SUB:
			k = c.program.addConstant(NewInt64(n * -1))
		case ast.BNT:
			k = c.program.addConstant(NewInt64(^n))
		default:
			return Void, newError(t.Pos, "Invalid unary operator %s", operator)
		}

	case ast.FLOAT:
		if operator != ast.SUB {
			return Void, newError(t.Pos, "Invalid unary operator %s", operator)
		}
		n, err := strconv.ParseFloat(t.Value, 64)
		if err != nil {
			return Void, newError(t.Pos, "Invalid float value %s", t.Value)
		}
		k = c.program.addConstant(NewFloat(n * -1))

	case ast.TRUE, ast.FALSE:
		if operator != ast.NOT {
			return Void, newError(t.Pos, "Invalid unary operator %s", operator)
		}
		if t.Value == "true" {
			k = c.program.addConstant(FalseValue)
		} else {
			k = c.program.addConstant(TrueValue)
		}
	}

	if dest != Void {
		c.emit(op_ldk, dest, k, Void, t.Pos)
	}

	return k, nil
}

func (c *compiler) compileSelectorExpr(t *ast.SelectorExpr, dest *Address) (*Address, error) {
	// check if is a module call
	ident, ok := t.X.(*ast.IdentExpr)
	if ok {
		addr, err := c.compileModuleExpr(ident.Name, t.Sel.Name, dest, t.Position())
		if err != nil {
			return Void, err
		}
		if addr != Void {
			return addr, nil
		}

		addr, err = c.compileNativeFunction(ident.Name, t.Sel.Name, dest, t.Position())
		if err != nil {
			return Void, err
		}
		if addr != Void {
			return addr, nil
		}

		addr, err = c.compileNativeProperty(ident.Name, t.Sel.Name, dest, t.Position())
		if err != nil {
			return Void, err
		}
		if addr != Void {
			return addr, nil
		}
	}

	// get the map address
	x, err := c.compileExpr(t.X, Void)
	if err != nil {
		return Void, err
	}

	// get the key. Its a constant string
	key := c.program.addConstant(NewString(t.Sel.Name))

	if dest == Void {
		dest = c.newTempRegister()
	}

	// move the value from the map to the dest register
	c.emit(op_get, dest, x, key, t.X.Position())

	return dest, nil
}

func (c *compiler) compileNativeFunction(pkg, name string, dest *Address, pos ast.Position) (*Address, error) {
	var fullName string

	if pkg != "" {
		fullName = pkg + "." + name
	} else {
		// built-ins don't have a package
		fullName = name
	}

	f, ok := allNativeMap[fullName]
	if !ok {
		return Void, nil
	}

	addr := NewAddress(AddrNativeFunc, f.Index)
	if dest == Void {
		return addr, nil
	}

	c.emit(op_mov, dest, addr, Void, pos)
	return dest, nil
}

func (c *compiler) compileNativeProperty(pkg, name string, dest *Address, pos ast.Position) (*Address, error) {
	fullName := "->" + pkg + "." + name

	f, ok := allNativeMap[fullName]
	if !ok {
		return Void, nil
	}

	addr := NewAddress(AddrNativeFunc, f.Index)

	if dest == Void {
		dest = c.newTempRegister()
	}

	c.emit(op_rnp, dest, addr, Void, pos)
	return dest, nil
}

func (c *compiler) compileModuleExpr(module, name string, dest *Address, pos ast.Position) (*Address, error) {
	addr, err := c.findModuleRegister(module, name, pos)
	if err != nil {
		return Void, err
	}

	if addr == Void {
		return Void, nil
	}

	if dest == Void {
		return addr, nil
	}

	c.emit(op_mov, dest, addr, Void, pos)
	return dest, nil
}

func (c *compiler) compileCallExpr(t *ast.CallExpr, dest *Address, retVal bool) (*Address, error) {
	// if it is a method m is the constant of the method name
	i, err := c.compileExpr(t.Ident, Void)
	if err != nil {
		return Void, err
	}

	if retVal && dest == Void {
		dest = c.newTempRegister()
	}

	if !t.Spread && len(t.Args) == 1 {
		exp, err := c.compileExpr(t.Args[0], Void)
		if err != nil {
			return Void, err
		}

		// call: R(A) funcIndex, R(B) retAddress, R(C) argsAddress
		c.emit(op_cas, i, dest, exp, t.Position())
		return dest, nil
	}

	args, err := c.compileCallArgs(t.Args, t.Spread)
	if err != nil {
		return Void, err
	}

	// call: R(A) funcIndex, R(B) retAddress, R(C) argsAddress
	c.emit(op_cal, i, dest, args, t.Position())
	return dest, nil
}

// compile the arguments of a function call.
func (c *compiler) compileCallArgs(params []ast.Expr, spreadArg bool) (*Address, error) {
	ln := len(params)
	if ln == 0 {
		return Void, nil
	}

	dest := c.newTempRegister()
	c.emit(op_arr, dest, NewAddress(AddrData, ln), Void, params[0].Position())

	for i, p := range params {
		exp, err := c.compileExpr(p, Void)
		if err != nil {
			return Void, err
		}

		c.emit(op_set, dest, NewAddress(AddrData, i), exp, p.Position())
	}

	if spreadArg {
		// concat the last arg if is spread
		c.emit(op_spa, dest, Void, Void, params[ln-1].Position())
	}

	return dest, nil
}

func (c *compiler) compileClassDeclStmt(t *ast.ClassDeclStmt) error {
	name := c.registerName(t.Name)

	cl := &Class{
		Name:     name,
		Exported: t.Exported,
	}

	for _, f := range t.Fields {
		cl.Fields = append(cl.Fields, &Field{
			Name:     f.Name,
			Exported: f.Exported,
		})
	}

	// Set as a top function and restart closures
	c.currentFunc = c.globalFunc
	c.closures = nil

	var constructorCompiled bool

	for _, f := range t.Functions {
		f.ReceiverType = name

		if f.Name == "constructor" {
			f.Name = name + ".prototype." + f.Name
			if err := c.compileConstructor(cl, f, t); err != nil {
				return err
			}
			constructorCompiled = true
			continue
		}

		f.Name = name + ".prototype." + f.Name
		fi, err := c.compileFuncDecl(f, true)
		if err != nil {
			return err
		}

		fi.function.IsClass = true
		cl.Functions = append(cl.Functions, fi.function.Index)
	}

	if !constructorCompiled && len(t.Fields) > 0 {
		f := &ast.FuncDeclStmt{
			Name:         name + ".prototype.constructor",
			ReceiverType: name,
			Pos:          t.Pos,
		}
		if err := c.compileConstructor(cl, f, t); err != nil {
			return err
		}
	}

	c.program.Classes = append(c.program.Classes, cl)
	c.currentFunc = c.globalFunc

	return nil
}

// compile fields before the function body if it has one
func (c *compiler) compileConstructor(cl *Class, t *ast.FuncDeclStmt, ct *ast.ClassDeclStmt) error {
	var argsLen int
	if t.Args != nil {
		argsLen = len(t.Args.List)
	}

	fi, err := c.addFunction(User, t.Name, false, t.Pos)
	if err != nil {
		return err
	}

	fi.receiverType = t.ReceiverType

	f := fi.function
	f.IsClass = true
	f.Variadic = t.Variadic
	f.Arguments = argsLen

	cl.Functions = append(cl.Functions, f.Index)

	if t.Args != nil {
		// Create first the arguments because when the function is called
		// they are copied directly to the beginning of the values.
		for _, arg := range t.Args.List {
			c.newRegister(arg.Name, false)
		}
	}

	// reserve a register for the "this" object.
	// but *after* the params.
	this := c.newRegister("this", false)
	fi.thisAddress = this

	// initialize fields
	for _, fl := range ct.Fields {
		i := c.program.addConstant(NewString(fl.Name))

		// if the field is unitialized set it as NULL
		e, ok := fl.Value.(*ast.ConstantExpr)
		if ok && e.Kind == ast.UNDEFINED {
			v := c.program.addConstant(NullValue)
			c.emit(op_set, this, i, v, fl.Position())
			continue
		}

		dst, err := c.compileExpr(fl.Value, Void)
		if err != nil {
			return err
		}
		c.emit(op_set, this, i, dst, fl.Position())
	}

	if t.Body != nil {
		if err := c.compileBlockStmt(t.Body); err != nil {
			return err
		}
	}

	// make sure that the last instruction is a return
	c.ensureReturn(fi.function)

	return nil
}

func (c *compiler) compileConstantExpr(t *ast.ConstantExpr, dest *Address) (*Address, error) {
	k, err := c.newConstant(t)
	if err != nil {
		return Void, err
	}

	if dest != Void {
		c.emit(op_ldk, dest, k, Void, t.Pos)
	}

	return k, nil
}

func (c *compiler) newConstant(t *ast.ConstantExpr) (*Address, error) {
	p := c.program

	switch t.Kind {
	case ast.INT:
		n, err := strconv.ParseInt(t.Value, 10, 64)
		if err != nil {
			return Void, newError(t.Pos, "Invalid int value %s", t.Value)
		}
		return p.addConstant(NewInt64(n)), nil

	case ast.FLOAT:
		n, err := strconv.ParseFloat(t.Value, 64)
		if err != nil {
			return Void, newError(t.Pos, "Invalid float value %s", t.Value)
		}
		return p.addConstant(NewFloat(n)), nil

	case ast.STRING:
		return p.addConstant(NewString(t.Value)), nil

	case ast.TRUE:
		return p.addConstant(TrueValue), nil

	case ast.FALSE:
		return p.addConstant(FalseValue), nil

	case ast.NULL:
		return p.addConstant(NullValue), nil

	case ast.RUNE:
		return p.addConstant(NewRune(rune(t.Value[0]))), nil

	case ast.UNDEFINED:
		return p.addConstant(UndefinedValue), nil

	default:
		return Void, newError(t.Pos, "Invalid type %s", t.Value)
	}
}

func (ctx *compiler) emit(op Opcode, a, b, c *Address, pos ast.Position) *Instruction {
	i := NewInstruction(op, a, b, c)
	f := ctx.currentFunc.function
	f.Instructions = append(f.Instructions, i)

	p := ctx.program

	fileIndex := p.FileIndex(pos.FileName)
	if fileIndex == -1 {
		fileIndex = len(p.Files)
		p.Files = append(p.Files, pos.FileName)
	}

	f.Positions = append(f.Positions, Position{
		File:   fileIndex,
		Line:   pos.Line,
		Column: pos.Column,
	})

	return i
}

func (c *compiler) newTempRegister() *Address {
	return c.newRegister("@", false)
}

func (c *compiler) registerName(name string) string {
	if strings.ContainsRune(name, '.') {
		return name
	}

	if c.module != "" {
		name = c.module + "." + name
	}

	return name
}

func (c *compiler) newRegister(name string, exported bool) *Address {
	if c.currentFunc == c.globalFunc {
		name = c.registerName(name)
	}

	r := &Register{
		Name:     name,
		Module:   c.module,
		StartPC:  c.pc(),
		Exported: exported,
	}

	fi := c.currentFunc
	r.Index = fi.registerTop
	fi.incRegIndex()

	f := fi.function
	f.Registers = append(f.Registers, r)

	var kind AddressKind
	if fi == c.globalFunc {
		kind = AddrGlobal
	} else {
		kind = AddrLocal
	}

	return NewAddress(kind, r.Index)
}

// find a register in the current scope.
func (c *compiler) findRegister(name string, fi *functionInfo) (*Address, error) {
	// search local registers
	if !fi.function.IsGlobal {
		f := fi.function
		pc := len(f.Instructions)
		for i := len(f.Registers) - 1; i >= 0; i-- {
			r := f.Registers[i]
			if r.StartPC > 0 && r.StartPC >= pc {
				continue
			}
			if r.Name == name && (r.EndPC == 0 || pc <= r.EndPC) {
				return NewAddress(AddrLocal, r.Index), nil
			}
		}

		// Search for a closure
		frameIndex := 0
		parentFI := fi.parent
		for parentFI != nil {
			if parentFI.function.IsGlobal {
				break
			}

			frameIndex++
			parentFn := parentFI.function
			pc := len(parentFn.Instructions)
			for i := len(parentFn.Registers) - 1; i >= 0; i-- {
				r := parentFn.Registers[i]
				if r.Name == name && pc >= r.StartPC && (r.EndPC == 0 || pc <= r.EndPC) {
					// we can't know in advance the index in the global array of closures
					// because previous registers can be referenced later and thus marked
					// as closure so keep a index (ix) in the compiler compiler and after
					// compiled the top function, update all in updateClosureIndexes.
					ix := c.markAsClosure(parentFn, r)
					return NewAddress(AddrClosure, ix), nil
				}
			}

			parentFI = parentFI.parent
		}
	}

	// search globals. Global registers are always in scope.
	gfi := c.globalFunc
	gf := gfi.function
	pc := len(gf.Instructions)

	// global registers are prefixed
	globalName := c.registerName(name)

	for i := len(gf.Registers) - 1; i >= 0; i-- {
		r := gf.Registers[i]

		if r.StartPC >= pc {
			continue
		}
		if r.Name == globalName && (r.EndPC == 0 || pc <= r.EndPC) {
			if r.Module != c.module && !r.Exported {
				continue
			}
			return NewAddress(AddrGlobal, r.Index), nil
		}
	}

	// search classes
	for _, cl := range c.program.Classes {
		if name == cl.Name {
			if !cl.Exported {
				return Void, fmt.Errorf("%s is not exported", name)
			}
			k := c.program.addConstant(NewString(name))
			return k, nil
		}
	}

	// search functions
	fullName := c.registerName(name)
	if fi, ok := c.functions[fullName]; ok {
		f := fi.function
		if !strings.ContainsRune(name, '@') {
			if fi.module != c.module && !f.Exported {
				return Void, fmt.Errorf("%s is not exported", name)
			}
		}
		return NewAddress(AddrFunc, f.Index), nil
	}

	// search built-ins
	for _, k := range c.builtinFuncs {
		if name == k {
			return c.compileNativeFunction("", name, Void, ast.Position{})
		}
	}

	return Void, nil
}

func (c *compiler) findModuleRegister(moduleAlias, name string, pos ast.Position) (*Address, error) {
	// check if the prefix is an imported module
	var modulePath string
	for _, imp := range c.imports {
		if imp.Alias == moduleAlias {
			modulePath = imp.AbsPath
			break
		}
	}

	if modulePath != "" {
		addr, err := c.findRegister(modulePath+"."+name, c.globalFunc)
		if err != nil {
			return Void, newError(pos, err.Error())
		}
		if addr != Void {
			return addr, nil
		}
	}

	// if the prefix is an import check if the module is not a local variable
	// of the current function and if it isn't mark it as unresolved
	if modulePath != "" {
		fi := c.currentFunc
		i, err := c.findRegister(moduleAlias, fi)
		if err != nil {
			return Void, err
		}
		if i == Void {
			i := c.getUnresolved(modulePath+"."+name, pos)
			return i, nil
		}
	}

	return Void, nil
}

// signals that the register is referenced by a closure
func (c *compiler) markAsClosure(f *Function, r *Register) int {
	for _, v := range f.Closures {
		if v.Name == r.Name &&
			v.Index == r.Index &&
			v.StartPC == r.StartPC &&
			v.EndPC == r.EndPC {
			// it's already marked
			for i, cl := range c.closures {
				if cl.reg.Equals(r) {
					return i
				}
			}
		}
	}
	f.Closures = append(f.Closures, r)
	c.closures = append(c.closures, &closure{fn: f, reg: r})
	return len(c.closures) - 1
}

func (c *compiler) openScope() {
	fi := c.currentFunc
	fi.scopes = append(fi.scopes, fi.registerTop)
}

func (c *compiler) closeScope() {
	// If we are in a function.
	fi := c.currentFunc
	l := len(fi.scopes) - 1
	lcompiler := fi.scopes[l]
	fi.scopes = fi.scopes[:l]

	f := fi.function
	pc := len(f.Instructions) - 1
	for i, k := 0, len(f.Registers); i < k; i++ {
		r := f.Registers[i]
		if r.Index >= lcompiler && r.EndPC == 0 {
			f.Registers[i].EndPC = pc
		}
	}
}

// returns if a register is in scope for the current PC
func (c *compiler) isInScope(name string) (bool, bool) {
	pc := c.pc()
	fi := c.currentFunc
	return isInScope(name, pc, fi.scopes, fi.function.Registers), false
}

func isInScope(name string, pc int, scopes []int, registers []*Register) bool {
	ln := len(scopes)

	var start int
	if ln > 0 {
		start = scopes[ln-1]
	}

	for i, l := start, len(registers); i < l; i++ {
		r := registers[i]
		if (r.EndPC == 0 || pc <= r.EndPC) && r.Name == name {
			return true
		}
	}
	return false
}

func (c *compiler) pc() int {
	fi := c.currentFunc
	return len(fi.function.Instructions)
}

func (c *compiler) openBranch(stmt target) *branch {
	b := &branch{stmt: stmt}
	c.branches = append(c.branches, b)
	return b
}

func (c *compiler) closeBranch() {
	for i := len(c.branches) - 1; i >= 0; i-- {
		b := c.branches[i]
		if b.outOfScope {
			continue
		}
		b.outOfScope = true
		break
	}
}

type unresolved struct {
	name     string
	pos      ast.Position
	pc       int      // to resolve scope
	address  *Address // to search and replace in the program
	module   string
	function *functionInfo // the function where is declared
}

func (c *compiler) getUnresolved(name string, pos ast.Position) *Address {
	index := -1
	for i, u := range c.unresolved {
		if u.name == name && u.module == c.module {
			index = i
		}
	}

	if index == -1 {
		index = c.unresolvedIndex
		c.unresolvedIndex++
	}

	i := NewAddress(AddrUnresolved, index)

	c.unresolved = append(c.unresolved, &unresolved{
		name:     name,
		pos:      pos,
		pc:       c.pc(),
		address:  i,
		module:   c.module,
		function: c.currentFunc,
	})

	return i
}

func (c *compiler) fixUnresolved() error {
	for _, u := range c.unresolved {

		c.module = u.module
		v, err := c.findRegister(u.name, c.globalFunc)
		if err != nil {
			return newError(u.pos, err.Error())
		}

		if v.Kind == AddrVoid {
			if u.module != "" {
				v, err = c.findRegister(u.module+"."+u.name, c.globalFunc)
				if err != nil {
					return newError(u.pos, err.Error())
				}
			}
		}

		if v.Kind == AddrVoid {
			return newError(u.pos, "Undeclared identifier: %s", u.name)
		}

		// Check that a global variable in the same module is not used before is declared.
		if v.Kind == AddrGlobal {
			// globals called from inside a function are always in scope
			if u.function == c.globalFunc {
				// get the register module
				r := c.globalFunc.function.Registers[v.Value]
				// if they are in different modules then globals are always in scope.
				// only when they are in the same module they cant be used before declared.
				if u.module == r.Module {
					if r.StartPC >= u.pc {
						return newError(u.pos, "Undeclared identifier: %s", u.name)
					}
				}
			}
		}

		// replace in all instructions
		c.replaceAddress(u.address, v)
	}

	c.unresolved = nil

	return nil
}

func (c *compiler) replaceAddress(dst *Address, src *Address) {
	dst.Kind = src.Kind
	dst.Value = src.Value
}

type closure struct {
	fn    *Function
	reg   *Register
	index int
}

func (c *closure) String() string {
	return fmt.Sprintf("%s->%s", c.fn.Name, c.reg.Name)
}

type branch struct {
	stmt       target
	breaks     []*jumpInstr
	continues  []*jumpInstr
	tryCatchs  []*ast.TryStmt
	outOfScope bool
	inFinally  bool // if the current code is inside a finally block
}

func (b *branch) isValidTarget(label string) bool {
	if b.outOfScope {
		return false
	}

	stmt := b.stmt
	if stmt == nil {
		return false
	}

	if label != "" {
		lbl := stmt.Label()
		if lbl == "" || label != lbl {
			return false
		}
	}

	return true
}

type target interface {
	Label() string
}

type continueTarget interface {
	Label() string
	ContinuePC() int
}

type breakTarget interface {
	Label() string
	BreakPC() int
}

type jumpInstr struct {
	inst   *Instruction
	pc     int
	target target
}

type CompilerError struct {
	Pos     ast.Position
	message string
}

func (e CompilerError) Position() ast.Position {
	return e.Pos
}

func (e CompilerError) Message() string {
	return e.message
}

func (e CompilerError) Error() string {
	return fmt.Sprintf("Compiler error: %s\n -> %v", e.message, e.Pos)
}

func newError(pos ast.Position, format string, args ...interface{}) CompilerError {
	return CompilerError{pos, fmt.Sprintf(format, args...)}
}
