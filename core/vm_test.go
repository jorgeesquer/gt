package core

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/gtlang/filesystem"
	"github.com/gtlang/gt/parser"
)

// Tests: Expressions
func TestExpression1(t *testing.T) {
	data := []struct {
		expression string
		expected   interface{}
	}{
		{"return 3", int64(3)},
		{"return 3+3", int64(6)},
		{"return -1 + 2", int64(1)},
		{"return -1 - 2", int64(-3)},
		{"return -1 + -2", int64(-3)},
		{"return 1 + -2", int64(-1)},
		{"return 3 * 3", int64(9)},
		{"return 6 / 2", float64(3)},
		{"return 1.000000000000001 == 1", false},
		{"return 1.2 == 1", false},
		{"return 1.1 === 1", false},
		{"return 1.2 != 1", true},
		{"return 1.1 !== 1", true},
		{"return 11 % 3", int64(2)},
		{"return 3 + 10 / 2", float64(8)},
		{"return 3 + 10 % 2", int64(3)},
		{"return 3 + 2 * 2", int64(7)},
		{"return (3 + 2) * 2", int64(10)},
		{"return (3 + 2) * (3 * 2)", int64(30)},
		{"return ((3 + 2) * 2) + (3 + 2 * 2) - 2", int64(15)},
		{"return true == 1", true},
		{"return true == 2", false},
		{"return true == 0", false},
		{"return false == 0", true},
		{"return false == 1", false},
		{"return true == 5", false},
		{"return true && false", false},
		{"return true || false", true},
		{"return true || false && true", true},
		{"return (true || false) && true", true},
		{"return 3 > 2", true},
		{"return 3 >= 2", true},
		{"return 1.2 > 1", true},
		{"return 1.2 >= 1", true},
		{"return 3 < 4", true},
		{"return 3 <= 4", true},
		{"return 1.2 < 1", false},
		{"return 1.2 <= 1", false},
		{"return 3 != 2", true},
		{"return 3 == 3", true},
		{"return !false", true},
		{"return !true", false},
		{"return true ? 1 : 2", int64(1)},
		{"return false ? 1 : 2", int64(2)},
		{"return 1 == null ? 1 : 2", int64(2)},
		{"return 0xA + 0xB", int64(21)},
		{"return 0xAA ^ 0xBB", int64(17)},
		{"return 0xFF", int64(255)},
		{"return 1 | 2", int64(3)},
		{"return 1 | 5", int64(5)},
		{"return 3 ^ 6", int64(5)},
		{"return 3 & 6", int64(2)},
		{"return 50 >> 2", int64(12)},
		{"return 2 << 5", int64(64)},
		{`
			let a = 1;
			a++;
			return a;`, int64(2)},
		{`
			let a = 1;
			a--;
			return a;`, int64(0)},
		{`
			let a = 1;
			a += 2;
			return a;`, int64(3)},
		{`
			let a = 1;
			a -= 2;
			return a;`, int64(-1)},
		{`
			let a = 2;
			a *= 2;
			return a;`, int64(4)},
		{`
			let a = 6;
			a /= 2;
			return a;`, float64(3),
		},
	}

	for _, d := range data {
		assertValue(t, d.expected, d.expression)
	}
}

func TestMain(t *testing.T) {
	assertValue(t, 5, `
		function main() {
			return 2 + 3
		}
	`)
}

func TestLoopBasic(t *testing.T) {
	assertValue(t, 2, `
		function main() {
			let b = 0
			for(let i = 0; i < 2; i++) {
				b += 1
			}
			return b
		}
	`)
}

//Tests: For
func TestLoop0(t *testing.T) {
	assertValue(t, 10, `
		let a = 0;
		for (let i = 0; i < 10; i++) {
			a++
		}		
		return a
	`)
}

func TestLoop1(t *testing.T) {
	assertValue(t, 10, `
		let a = 0;
		for (let i = 0, l = 10; i < l; i++) {
			a++
		}		
		return a
	`)
}

func TestLoop2(t *testing.T) {
	assertValue(t, 3, `
		let a = 0;
		for (let i = 0; i < 10 && a < 3; i++) {
			a++
		}		
		return a
	`)
}

func TestLoop3(t *testing.T) {
	assertValue(t, 3, `
		let a = 0;
		let i = 0;
		
		function inc() { i++ }
			
		for (i = 0; i < 10 && a < 3; inc()) {
			a++
		}
		return a
	`)
}

func TestLoopLabel1(t *testing.T) {
	assertValue(t, 10, `
		let a = 0;
	
		foo:
		for(var k = 0; k < 5; k++) {
			a++;
			
			for(var i = 0; i < 5; i++) {	
				a++;
				 
				if(k > 3) {
					break foo
				}
				continue foo
				a++;
			}
		}
		return a
	`)
}

func TestLoopLabel2(t *testing.T) {
	assertValue(t, 8, `
		let a = 0
		let nums = [1,2,3,4,5]
	
		foo:
		for(var k of nums) {	
			a++;
			
			for(var i = 0; i < 5; i++) {	
				a++;
				
				if(k > 3) {
					break foo
				}
				continue foo
				a++;
			}
		}
		return a
	`)
}

func TestLoopLabel3(t *testing.T) {
	assertValue(t, 10, `
		let a = 0;
		let nums = [1,2,3,4,5]
	
		foo:
		for(var k in nums) {	
			a++;
			
			for(var i = 0; i < 5; i++) {	
				a++;
				
				if(k > 3) {
					break foo
				}
				continue foo
				a++;
			}
		}
		return a
	`)
}

func TestLoopLabel4(t *testing.T) {
	assertValue(t, 4, `
		let a = 0;
		foo:
       		for(;;) {	
			a++;

			for(var i = 0; i < 5; i++) {	
				a++;
				
				if(a > 3) {
					break foo
				}
				continue foo
				a++;
			}
		}
		return a
	`)
}

func TestLoopLabel5(t *testing.T) {
	assertValue(t, 4, `
		let a = 0;
		foo:
       		while(true) {	
			a++;

			for(var i = 0; i < 5; i++) {	
				a++;
				
				if(a > 3) {
					break foo
				}
				continue foo
				a++;
			}
		}
		return a
	`)
}

func TestLoopLabel6(t *testing.T) {
	assertValue(t, 3, `
			var i = 0;

			LABEL:
			for (var value of [1]) {
				try {
					i++
					throw "exception"
				}
				catch{
					i++
					break LABEL
				} finally {
					i++
				}
			}
			return i
	`)
}

func TestCall1(t *testing.T) {
	assertValue(t, 5, `
		function sum(a, b) {
			return a + b
		}
		function main() {
			return sum(2, 3)
		}
	`)
}

func TestCall2(t *testing.T) {
	assertValue(t, 2, `
		function foo(a) {
			return bar(a)
		}
		function bar(a) {
			return a + 1
		}
		function main() {
			return foo(1)
		}
	`)
}

func TestRet(t *testing.T) {
	assertValue(t, 3, `
		function foo() {
			let i = 0
			while(true) {
				if (i == 3) {
					return i
				}
				i++
			}
		}

		function main() {
			return foo()
		}
	`)
}

func TestFib(t *testing.T) {
	assertValue(t, 8, `
		function fib(n) {
			if (n < 2) {
				return n
			}
			return fib(n - 1) + fib(n - 2)
		}

		function main() {
			return fib(6)
		}
	`)
}

func TestStacktrace(t *testing.T) {
	p := compileTest(t, `
		function foo() {
			bar()
		}

		function bar() {
			throw "snap!"
		}

		function main() {
			foo()
		}
	`)

	vm := NewVM(p)
	_, err := vm.Run()

	se := normalize(`
		-> line 7
		-> line 3
		-> line 11
	`)

	if !strings.Contains(normalize(err.Error()), se) {
		t.Fatal(err)
	}
}

func TestStacktrace2(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("/main.ts", []byte(`
		import * as foo from "other/path/bar"

		function main() {
			return foo.bar()
		}
	`))

	fs.WritePath("/other/path/bar.ts", []byte(`
		export function bar() {
			return 1 / 0
		}
	`))

	p, err := Compile(fs, "main.ts")
	if err != nil {
		t.Fatal(err)
	}

	vm := NewVM(p)
	_, err = vm.Run()

	se := normalize(`
		-> /other/path/bar.ts:3
		-> /main.ts:5
	`)

	if !strings.Contains(normalize(err.Error()), se) {
		t.Fatal(err)
	}
}

func TestReturnFromScript(t *testing.T) {
	assertValue(t, 5, `
		return 5
	`)
}

func TestTry0(t *testing.T) {
	p := compileTest(t, `
		let x;		
		try {
			x = 1 / 0;
		} 
		finally {
			x = 5;
		}
	`)

	vm := NewVM(p)
	_, err := vm.Run()
	if err == nil || !strings.Contains(err.Error(), "divide by zero") {
		t.Fatal("should throw excetion")
	}

	v, _ := vm.RegisterValue("x")
	if v != NewValue(5) {
		t.Fatal(v)
	}
}

func TestTry1(t *testing.T) {
	assertValue(t, 5, `
		let x	
		try {	
			x = 1 / 0
		} catch {
			x = 5
		}
		return x
	`)
}

func TestTry2(t *testing.T) {
	assertValue(t, -2, `
		let x;		
		try {
			x = 1 / 0;
		} catch {
			x = -1;
		} finally {
			x -= 1;
		}
		return x
	`)
}

func TestTry3(t *testing.T) {
	assertValue(t, 0, `
		let x;		
		try {
			x = 1;
		} catch {
			x = -1;
		} finally {
			x -= 1;
		}
		return x
	`)
}

func TestTry4(t *testing.T) {
	p := compileTest(t, `
		let x;		
		try {
			x = 1 / 0;
		} catch {
			x = 1 / 0;
		} finally {
			x -= 1;
		}
		return x
	`)

	vm := NewVM(p)
	_, err := vm.Run()
	if err == nil || !strings.Contains(err.Error(), "divide by zero") {
		t.Fatal("should throw excetion")
	}
}

func TestTryFinally(t *testing.T) {
	assertValue(t, -3, `
		let x;		
		try {
			x = 0;
			try {
				try {
					// noop
				} catch {
					x = -1;
				} finally {
					x -= 1; // <---------
				}
			} catch {
				x = -1;
			} finally {
				x -= 1; // <---------
			}
		} catch {
			x = -1;
		} finally {
			x -= 1; // <---------
		}
		return x
	`)
}

func TestTryFinally2(t *testing.T) {
	assertValue(t, -3, `
		let x;		
		try {
			try {
				try {
					x = 0
				} finally {
					x -= 1; // <---------
				}
			} finally {
				x -= 1; // <---------
			}
		} finally {
			x -= 1; // <---------
		}
		return x
	`)
}

func TestTryFinally3(t *testing.T) {
	assertRegister(t, "x", -3, `
		let x;		
		try {
			try {
				try {
					x = 0;
					return;  // <--------------- exit
				} finally {
					x -= 1; // <---------
				}
			} finally {
				x -= 1; // <---------
			}
		} finally {
			x -= 1; // <---------
		}
	`)
}

func TestTryFinally4(t *testing.T) {
	assertRegister(t, "x", -3, `
		let x;		
		try {
			try {
				try {
					x = 0;
				} finally {
					x -= 1; // <--------- Must execute
					return;  // <--------------- exit
				}
			} finally {
				x -= 1; // <--------- Must execute
			}
		} finally {
			x -= 1; // <--------- Must execute
		}
	`)
}

func TestTryFinally5(t *testing.T) {
	assertRegister(t, "x", -3, `
		let x = 0	
		function foo() {
			try {
				try {
					try {
						x = 0
					} finally {
						x -= 1
						return // <--------------- exit
					}
				} finally {
					x -= 1 // <--------- Must execute
				}
			} finally {
				x -= 1 // <--------- Must execute
			}
		}
		
		function bar() {
			foo()
		}
		
		bar()	
	`)
}

func TestTryFinally6(t *testing.T) {
	assertRegister(t, "x", -3, `
		let x = 5;	
		function foo() {
			try {
				try {
					try {
						x = 1 / 0;
					} 
					catch(e) {
						x = 0;
					}
					finally {
						x -= 1; // <---------
						return;  // <--------------- exit
					}
				} finally {
					x -= 1; // <---------
				}
			} finally {
				x -= 1; // <---------
			}
		}
		
		function bar() {
			foo()
		}
		
		bar()	
	`)
}

// test that try and finally are in the same scope
func TestTryFinally8(t *testing.T) {
	assertValue(t, 1, `
		let x;
		
		function foo() {
	        try {
				// a L0
				let a = 1
	            return a;
	        }
	        finally {
				// b L1 because is in the same 
				// scope as the try body
	            let b = 2
	        }
		}
				
		return foo()
	`)
}

// test that a return before a finally is not overwritten
func TestTryFinally9(t *testing.T) {
	assertValue(t, 8, `
		function foo() {
		    try {
		        if (true) {
		            return bar(8);
		        }
		       	bar(11)
		    }
		    finally {
		        bar(7)
		    }
		}
		
		function bar(x) { return x }
				
		return foo()
	`)
}

// test that a return before a finally is not overwritten
//
// this tests is like the previous but changes the registers slightly.
func TestTryFinally10(t *testing.T) {
	assertValue(t, 8, `
		function foo() {
		    try {
		        if (true) {
		            return bar(8);
		        }
		    }
		    finally {
		        bar(7)
		    }
		}
		
		function bar(x) { return x }
				
		return foo()
	`)
}

// test that returning inside a finally has precedence
func TestTryFinally11(t *testing.T) {
	assertValue(t, 2, `
		function foo() {
		    try {
		        if (true) {
		            return bar(8);
		        }
		    }
		    finally {
		        bar(7)
				return 2
		    }
		}
		
		function bar(x) { return x }
				
		return foo()
	`)
}

// test that returning inside a finally has precedence
func TestTryFinally12(t *testing.T) {
	assertValue(t, 2, `
		function foo() {
		    try {
		        return bar(8);
		    }
		    finally {
				return 2
		    }
		}
		
		function bar(x) { return x }
				
		return foo()
	`)
}

// test that returning inside a finally has precedence
func TestTryFinally13(t *testing.T) {
	assertValue(t, 3, `
		function foo() {
		    try {
		        try {
			        return bar(8);
			    }
			    finally {
					return 2
			    }
		    }
		    finally {
				return 3
		    }
		}
		
		function bar(x) { return x }
				
		return foo()
	`)
}

// test that returning inside a finally has precedence
func TestTryFinally14(t *testing.T) {
	assertValue(t, 5, `
		function foo() {
		    try {
			    try {
			        try {
				        return bar(8);
				    }
				    finally {
						return 2
				    }
			    }
			    finally {
					return 3
			    }
		    }
		    finally {
				return 5
		    }
		}
		
		function bar(x) { return x }
				
		return foo()
	`)
}

func TestTryThrow1(t *testing.T) {
	p := compileTest(t, `
		throw "foo"			
	`)

	vm := NewVM(p)
	_, err := vm.Run()
	if err == nil {
		t.Fatal("Should fail")
	}
}

func TestTryThrow2(t *testing.T) {
	assertValue(t, 3, `
		let x = 1
		
		try {
		 	throw "foo";
		} 
		catch(e) {
			x += 1
		}
		finally {
			x += 1
		}	

		return x
	`)
}

func TestTryThrow3(t *testing.T) {
	assertValue(t, 3, `
		let x = 1;
		
		try {
		 	throw "foo";
			x = 2	
		} 
		catch(e) {		
			try {
			 	throw "foo";
			} 
			catch(e) {
				x = 3
			}
		}

		return x
	`)
}

// Check that the stackframe is restored
func TestTryThrow4(t *testing.T) {
	assertValue(t, "3", `
		let x;
		
		function foo() {
			foo2()
		}
		
		function foo2() {
			foo3()
		}
		
		function foo3() {
			throw "3"
		}
		
		try {		 	
			foo()	
		} 
		catch(e) {		
			x = e.message
		}

		return x
	`)
}

// Check that the stackframe is restored
func TestTryThrow5(t *testing.T) {
	p := compileTest(t, `
		let x
		
		function b() {
			try {
				throw "xx"
			}	
			finally {
				x = 10
			}
		}
	
		function a() {
			try {
				b()
			}			
			finally {
				x++
			}
		}
		
		a()
		return x
	`)

	vm := NewVM(p)
	_, err := vm.Run()
	if err == nil {
		t.Fatal("Should fail")
	}

	v, ok := vm.RegisterValue("x")
	if !ok {
		t.Fatal("Reg x not found")
	}

	if v != NewValue(11) {
		t.Fatal(v)
	}
}

func TestScope(t *testing.T) {
	ast, err := parser.ParseStr(`
		function foo(a) {
			let a = 1
		}
	`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewCompiler().Compile(ast)
	if err == nil || !strings.Contains(err.Error(), "Redeclared identifier") {
		t.Fatal(err)
	}
}

func TestClosure0(t *testing.T) {
	assertValue(t, 3, `
		function foo(a) {
			return function () {
				return a
			}
		}
		return foo(3)()
	`)
}

func TestClosure01(t *testing.T) {
	assertValue(t, 1, `
		function main() {
			let a = 0			
			let foo = () => { a++ }
			foo() 
			return a
		}
	`)
}

func TestClosure02(t *testing.T) {
	assertValue(t, 1, `
		function main() {
			let a = 0			
			let foo = () => { 
				a++
				for(let i = 0; i < 10; i++) {
					try {
						a += 1 / 0
					} catch {
						a++
					} finally {
						a--
					}
				}
			}			
			foo() 
			a--
			foo()
			return a
		}
	`)
}

func TestClosure1(t *testing.T) {
	assertValue(t, 15, `
	function foo() {
		let a = 15;			
		return function () {
			return function () {
				return function () {
					return a
				}
			}
		}
	}
	return foo()()()()
	`)
}

func TestClosure2(t *testing.T) {
	assertValue(t, 3, `	
		function counter() {
			let i = 0
			return function() {
				i++
				return i
			}
		}		
		
		let next = counter()
		let r = next()
		r += next()		
		return r
	`)
}

func TestClosure3(t *testing.T) {
	assertValue(t, 3, `	
		function counter() {
			let i = 0
			return {
				fn: function() {
					i++
					return i
				}
			}
		}		
		
		let next = counter().fn
		let r = next()
		r += next()
		return r
	`)
}

func TestClosure4(t *testing.T) {
	assertValue(t, 3, `	
		function counter() {
			let i = 0;			
			return () => { i++; return i }
		}		
		
		let next = counter();			
		let r = next();
		r += next();		
		return r;
	`)
}

func TestClosure5(t *testing.T) {
	assertValue(t, 3, `	
		function newDev(a, b) {
			return { a: a, b: b }
		}
		
		function bar(dev) {
			return dev.a;
		}
		
		function newReader(a, b) {
			let dev = newDev(a, b)			
			return { foo: () => bar(dev) }
		}
	
		return newReader(3,2).foo();
	`)
}

func TestClosure6(t *testing.T) {
	assertValue(t, 3, `
		function counterWrap() {
			let f = counter();
			return f;
		}
			
		function counter() {
			let i = 0;			
			return () => { i++; return i }
		}		
		
		let next = counterWrap();			
		let r = next();
		r += next();		
		return r;
	`)
}

func TestClosure7(t *testing.T) {
	assertValue(t, 21, `
		function foo() {
			let a = 8;	
			let b = 5;			
			return function () {
				let c = 2;
				return function () {		
				let j = 6;			
					return function () {
						return a + b + j + c;
					}
				}
			}
		}
		return foo()()()();
	`)
}

func TestClassClosure1(t *testing.T) {
	assertValue(t, 9, `
		class Foo {
			powFunc(a) {
				return () => a * a 
			}
			sumFunc(a, b) {
				return () => a + b 
			}
		}

		let foo = new Foo()
		let a = foo.powFunc(2)()
		let b = foo.sumFunc(2, 3)()
		return a + b
	`)
}

func TestClassClosure2(t *testing.T) {
	assertValue(t, 10, `
		class Foo {
			z;
			constructor(z) {
				this.z = z
			}
			powFunc(a) {
				return () => a * a 
			}
			sumFunc(a, b) {
				return () => a + b 
			}
		}

		let foo = new Foo(1)
		let a = foo.powFunc(2)()
		let b = foo.sumFunc(2, 3)()
		return a + b + foo.z // 1 + 4 + 5
	`)
}

// Tests: Enum
func TestEnum1(t *testing.T) {
	assertValue(t, 4, `
		enum Direction {
		    Up = 1,
		    Down,
		    Left,
		    Right
		}
		return Direction.Right
	`)
}

func TestEnum2(t *testing.T) {
	assertValue(t, 3, `
		enum Direction {
		    Up,
		    Down,
		    Left,
		    Right
		}
		return Direction.Right
	`)
}

func TestEnum3(t *testing.T) {
	assertValue(t, 5, `
		enum Direction {
		    Up = 5,
		    Down,
		    Left,
		    Right
		}
		return Direction.Up
	`)
}

func TestEnum4(t *testing.T) {
	assertValue(t, 8, `
		enum Direction {
		    Up = 5,
		    Down,
		    Left,
		    Right
		}
		return Direction.Right
	`)
}
func TestEnumString(t *testing.T) {
	assertValue(t, "up", `
		enum Direction {
		    Up = "up",
		    Down = "down"
		}
		return Direction.Up
	`)
}

// Tests: Error
func TestError(t *testing.T) {
	assertValue(t, "Attempt to divide by zero", `
		let x;		
		try {
			x = 1 / 0;
		} catch(e) {
			x = e.message
		}
		return x
	`)
}

func TestClass1(t *testing.T) {
	assertValue(t, 2, `
		class Foo {
			bar: number
			constructor(x: number) {
				this.bar = x
			}
			getNum() {
				return this.bar
			}
		}

		return new Foo(2).getNum()
	`)
}

func TestClassInitializeFields(t *testing.T) {
	assertValue(t, 2, `
		class Foo {
			bar: number = 2
		}
		return new Foo(2).bar
	`)
}

func TestModuleImports1(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("main.ts", []byte(`
		import * as foo from "foo"

		function main() {
			return foo.bar
		}
	`))

	fs.WritePath("foo.ts", []byte(`
		export const bar = 3
	`))

	assertValueFS(t, fs, "main.ts", 3)
}

func TestModuleImports2(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("/dir1/main.ts", []byte(`
		import * as foo from "/libs/foo"

		function main() {
			return foo.bar
		}
	`))

	fs.WritePath("/libs/foo.ts", []byte(`
		export const bar = 3
	`))

	assertValueFS(t, fs, "/dir1/main.ts", 3)
}

func TestModuleImports3(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("/dir1/main.ts", []byte(`
		import * as foo from "../libs/foo"

		function main() {
			return foo.bar
		}
	`))

	fs.WritePath("/libs/foo.ts", []byte(`
		export const bar = 3
	`))

	assertValueFS(t, fs, "/dir1/main.ts", 3)
}

func TestModuleImports4(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("/dir1/main.ts", []byte(`
		import * as foo from "../dir1/dir2/dir3/foo"

		function main() {
			return foo.bar
		}
	`))

	fs.WritePath("/dir1/dir2/dir3/foo.ts", []byte(`
		import * as xxx from "../../../other/path/bar"

		export const bar = xxx.foo()
	`))

	fs.WritePath("/other/path/bar.ts", []byte(`
		export function foo() {
			return 3
		}
	`))

	assertValueFS(t, fs, "/dir1/main.ts", 3)
}

func TestModuleImportsRelativeToModule(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("/dir1/main.ts", []byte(`
		import * as foo from "../dir1/foo"

		function main() {
			return foo.bar
		}
	`))

	fs.WritePath("/dir1/foo.ts", []byte(`
		import * as xxx from "../dir2/bar"

		export const bar = xxx.foo()
	`))

	fs.WritePath("/dir2/bar.ts", []byte(`
		export function foo() {
			return 3
		}
	`))

	assertValueFS(t, fs, "/dir1/main.ts", 3)
}

func TestModuleImports5(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("/dir1/main.ts", []byte(`
		import * as foo from "../other/path/bar"

		function main() {
			return new foo.Foo(3).bar
		}
	`))

	fs.WritePath("/other/path/bar.ts", []byte(`
		export class Foo {
			bar: number
			constructor(x: number) {
				this.bar = x
			}
			getNum() {
				return this.bar
			}
		}
	`))

	assertValueFS(t, fs, "/dir1/main.ts", 3)
}

func TestModuleImports6(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("/dir1/main.ts", []byte(`
		import * as foo from "bar"

		function main() {
			return new foo.Foo(3).bar
		}
	`))

	fs.WritePath("/dir1/bar.ts", []byte(`
		export class Foo {
			bar: number
			constructor(x: number) {
				this.bar = x
			}
			getNum() {
				return this.bar
			}
		}
	`))

	assertValueFS(t, fs, "/dir1/main.ts", 3)
}

func TestModuleImports7(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("/dir1/main.ts", []byte(`
		import * as foo from "./bar"

		function main() {
			return new foo.Foo(3).bar
		}
	`))

	fs.WritePath("/dir1/bar.ts", []byte(`
		export class Foo {
			bar: number
			constructor(x: number) {
				this.bar = x
			}
			getNum() {
				return this.bar
			}
		}
	`))

	assertValueFS(t, fs, "/dir1/main.ts", 3)
}

func TestModuleImportsFromOtherDir(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("/dir1/main.ts", []byte(`
		import * as foo from "./bar"

		function main() {
			return foo.bar()
		}
	`))

	fs.WritePath("/dir1/bar.ts", []byte(`
		export function bar() {
			return 3
		}
	`))

	fs.MkdirAll("/foo/bar")

	if err := fs.Chdir("/foo/bar"); err != nil {
		t.Fatal(err)
	}

	assertValueFS(t, fs, "../../dir1/main.ts", 3)
}

func TestModuleForSideEffects(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("main.ts", []byte(`
		import "foo"

		function main() {
			return bar
		}
	`))

	fs.WritePath("foo.ts", []byte(`
		export const bar = 3
	`))

	_, err := Compile(fs, "main.ts")
	if err == nil || !strings.Contains(err.Error(), "Undeclared identifier") {
		t.Fatal(err)
	}
}

func TestModuleForSideEffects2(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("main.ts", []byte(`
		import "foo"

		function main() {
			bar()
		}
	`))

	fs.WritePath("foo.ts", []byte(`
		export function bar() {}
	`))

	_, err := Compile(fs, "main.ts")
	if err == nil || !strings.Contains(err.Error(), "Undeclared identifier") {
		t.Fatal(err)
	}
}

func TestVisibility(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("main.ts", []byte(`
		import * as foo from "foo"

		function main() {
			foo.bar()
		}
	`))

	fs.WritePath("foo.ts", []byte(`
		function bar() {}
	`))

	_, err := Compile(fs, "main.ts")
	if err == nil || !strings.Contains(err.Error(), "not exported") {
		t.Fatal(err)
	}
}

func TestModuleSameNames(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("main.ts", []byte(`
		import * as foo from "bar"

		export function sum() {
			return 88
		}

		function main() {
			return foo.bar()
		}
	`))

	fs.WritePath("bar.ts", []byte(`
		export function bar() {
			return sum()
		}
		export function sum() {
			return 3
		}
	`))

	assertValueFS(t, fs, "main.ts", 3)
}

func TestInit0(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("main.ts", []byte(`
		import * as lib1 from "libs/lib1"

		export let v

		function init() {
			v = 1
		}

		function main() {
			return v + lib1.v
		}
	`))

	fs.WritePath("libs/lib1.ts", []byte(`		
		export let v

		function init() {
			v = 2
		}
	`))

	assertValueFS(t, fs, "main.ts", 3)
}

func TestInit1(t *testing.T) {
	fs := filesystem.NewVirtualFS()
	fs.WritePath("main.ts", []byte(`
		import * as lib1 from "libs/lib1"
		import * as lib2 from "libs/lib2"

		export let v

		function init() {
			v = 0
		}

		function main() {
			return v + lib1.foo() + lib2.v
		}
	`))

	fs.WritePath("libs/lib1.ts", []byte(`
		import * as lib2 from "libs/lib2"
		
		export let v

		function init() {
			v = 1
		}

		export function foo() {
			return v + lib2.v
		}
	`))

	fs.WritePath("libs/lib2.ts", []byte(`		
		export let v

		function init() {
			v = 2
		}
	`))

	assertValueFS(t, fs, "main.ts", 5)
}

func TestNativeFunc(t *testing.T) {
	libs := []NativeFunction{
		NativeFunction{
			Name:      "math.square",
			Arguments: 1,
			Function: func(this Value, args []Value, vm *VM) (Value, error) {
				v := args[0].ToInt()
				return NewInt64(v * v), nil
			},
		},
	}

	assertNativeValue(t, libs, 4, `
		function main() {
			return math.square(2)
		}
	`)
}

func TestNativeProperty(t *testing.T) {
	libs := []NativeFunction{
		NativeFunction{
			Name: "->math.pi",
			Function: func(this Value, args []Value, vm *VM) (Value, error) {
				return NewFloat(3.1416), nil
			},
		},
	}

	assertNativeValue(t, libs, 3.1416, `
		function main() {
			return math.pi
		}
	`)
}

func TestNativeObject(t *testing.T) {
	libs := []NativeFunction{
		NativeFunction{
			Name: "tests.newObject",
			Function: func(this Value, args []Value, vm *VM) (Value, error) {
				return NewObject(obj{}), nil
			},
		},
	}

	assertNativeValue(t, libs, "Hi foo", `
		function main() {
			let obj = tests.newObject()
			return obj.sayHi(obj.name)
		}
	`)
}

func TestNativeFuncError(t *testing.T) {
	libs := []NativeFunction{
		NativeFunction{
			Name:      "math.square",
			Arguments: 1,
			Function: func(this Value, args []Value, vm *VM) (Value, error) {
				return NullValue, fmt.Errorf("snap!")
			},
		},
		NativeFunction{
			Name:      "log.error",
			Arguments: -1,
			Function: func(this Value, args []Value, vm *VM) (Value, error) {
				return NullValue, nil
			},
		},
	}

	assertNativeValue(t, libs, nil, `
		function main() {
			try {
				math.square(2)				
			} catch (error) {
				log.error("asdfas", error)
			}
		}
	`)
}

type obj struct{}

func (d obj) GetProperty(key string, vm *VM) (Value, error) {
	switch key {
	case "name":
		return NewString("foo"), nil
	}
	return UndefinedValue, nil
}

func (d obj) GetMethod(name string) NativeMethod {
	switch name {
	case "sayHi":
		return d.sayHI
	}
	return nil
}

func (d obj) sayHI(args []Value, vm *VM) (Value, error) {
	return NewString("Hi " + args[0].ToString()), nil
}

func assertNativeValue(t *testing.T, funcs []NativeFunction, expected interface{}, code string) {
	a, err := parser.ParseStr(code)
	if err != nil {
		t.Fatal(err)
	}

	c := NewCompiler()

	for _, f := range funcs {
		AddNativeFunc(f)
	}

	p, err := c.Compile(a)
	if err != nil {
		t.Fatal(err)
	}

	vm := NewVM(p)

	// Print(p)
	// vm.MaxSteps = 10

	ret, err := vm.Run()
	if err != nil {
		t.Fatal(err)
	}

	v := NewValue(expected)

	if ret != v {
		t.Fatalf("Expected %v %T, got %v %T", expected, expected, ret, ret)
	}
}

func assertValueFS(t *testing.T, fs filesystem.FS, path string, expected interface{}) {
	p, err := Compile(fs, path)
	if err != nil {
		t.Fatal(err)
	}

	// Print(p)
	// PrintNames(p, true)
	// vm.MaxSteps = 50

	vm := NewVM(p)

	ret, err := vm.Run()
	if err != nil {
		t.Fatal(err)
	}

	v := NewValue(expected)

	if ret != v {
		t.Fatalf("Expected %v %T, got %v %T", expected, expected, ret, ret)
	}
}

func assertValue(t *testing.T, expected interface{}, code string) {
	p := compileTest(t, code)
	vm := NewVM(p)

	// Print(p)
	// vm.MaxSteps = 50

	ret, err := vm.Run()
	if err != nil {

		t.Fatal(err)
	}

	if ret != NewValue(expected) {
		t.Fatalf("Expected %v %T, got %v", expected, expected, ret.ToString())
	}
}

func assertRegister(t *testing.T, register string, expected interface{}, code string) {
	p := compileTest(t, code)
	vm := NewVM(p)

	// Print(p)
	// vm.MaxSteps = 50

	_, err := vm.Run()
	if err != nil {
		t.Fatal(err)
	}

	v, _ := vm.RegisterValue(register)

	if v != NewValue(expected) {
		t.Fatalf("Expected %v, got %v", expected, v)
	}
}

func compileTest(t *testing.T, code string) *Program {
	p, err := CompileStr(code)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func normalize(s string) string {
	var reg = regexp.MustCompile(`\s+`)
	s = reg.ReplaceAllString(s, ` `)
	return s
}
