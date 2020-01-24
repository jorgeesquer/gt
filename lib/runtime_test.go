package lib

import (
	"testing"

	"github.com/gtlang/gt/core"
)

func TestRunFunc(t *testing.T) {
	v := runTest(t, `	
		function main() {
			return foo(1, 2)
		}

		function foo(a, b){ 
			return runtime.runFunc("sum", a, b)
		}

		function sum(a, b){ 
			return a + b 
		}
	`)

	if v.ToInt() != 3 {
		t.Fatal(v)
	}
}

func TestRunFunc2(t *testing.T) {
	v := runTest(t, `
		let v

		function main() {
			foo()
			return v
		}

		function foo(){ 
			try {
				runtime.runFunc("sum", 1, 2)
			} finally {
				v += 2
			}
		}

		function sum(a, b){ 
			v = a + b 
		}
	`)

	if v.ToInt() != 5 {
		t.Fatal(v)
	}
}

func TestRunFunc3(t *testing.T) {
	v := runTest(t, `
		let v

		function main() {
			let p = runtime.vm.program
			let vm = runtime.newVM(p)
			try {
				vm.runFunc("sum", 1, 2)
			} finally {
				v = vm.getValue("v")
				v += 2
			}
			return v
		}

		function sum(a, b){ 
			v = a + b 
		}
	`)

	if v.ToInt() != 5 {
		t.Fatal(v)
	}
}

func TestRunFunc4(t *testing.T) {
	v := runTest(t, `
		let v

		function main() {
			return foo()
		}

		function foo() {
			let p = runtime.vm.program
			let vm = runtime.newVM(p)
			try {
				vm.runFunc("sum", 1, 2)
			} finally {
				v = vm.getValue("v")
				v += 2
			}
			return v
		}

		function sum(a, b){ 
			v = a + b 
		}
	`)

	if v.ToInt() != 5 {
		t.Fatal(v)
	}
}

func TestPluginManager(t *testing.T) {
	assertRegister(t, "x", 5, `		
		let program = bytecode.compileStr("export function sum(a, b){ return a + b }")
	
		let plugin = runtime.newPlugin("foo", program)
	
		let pm = runtime.newPluginManager()
	
		pm.loadPlugin(plugin)
		
		runtime.context.addPlugin("foo")
	
		let x = pm.exec(runtime.context, "foo.sum", 2, 3)
	`)
}

func TestPluginManagerClone(t *testing.T) {
	runTest(t, `		
		let code = "export let a = 3;" +
				   "export function get(){ return a };" + 
				   "export function set(v){ a = v };"

		let program = bytecode.compileStr(code)	
		let plugin = runtime.newPlugin("foo", program)	
		let pm = runtime.newPluginManager()	
		pm.loadPlugin(plugin)		
		runtime.context.addPlugin("foo")
	
		pm.exec(runtime.context, "foo.set", 2)
		let a = pm.exec(runtime.context, "foo.get")
		if(a != 2) {
			throw a
		}

		let pm2 = pm.clone()
		a = pm2.exec(runtime.context, "foo.get")
		if(a != 3) {
			throw a
		}
	`)
}

func TestPluginManagerClone2(t *testing.T) {
	runTest(t, `		
		let code = "export let a = 3;" +
				   "export function get(){ return a };" + 
				   "export function set(v){ a = v };"

		let program = bytecode.compileStr(code)	
		let plugin = runtime.newPlugin("foo", program)	
		let pm = runtime.newPluginManager()	
		pm.loadPlugin(plugin)	

		runtime.context.pluginManager = pm
		runtime.context.addPlugin("foo")
	
		runtime.exec("foo.set", 2)
		let a = runtime.exec("foo.get")
		if(a != 2) {
			throw a
		}

		runtime.context.pluginManager = pm.clone()
		a = runtime.exec("foo.get")
		if(a != 3) {
			throw a
		}

		runtime.context.pluginManager = pm
		a = runtime.exec("foo.get")
		if(a != 2) {
			throw a
		}
	`)
}

func TestPluginManagerClone3(t *testing.T) {
	runTest(t, `		
		let code = "export let a = 3;" +
				   "export function get(){ return a };" + 
				   "export function set(v){ a = v };"

		let program = bytecode.compileStr(code)	
		let plugin = runtime.newPlugin("foo", program)	
		let pm = runtime.newPluginManager()	
		pm.loadPlugin(plugin)

		let ctx1 = runtime.context
		ctx1.pluginManager = pm
		ctx1.addPlugin("foo")
	
		ctx1.exec("foo.set", 2)
		let a = ctx1.exec("foo.get")
		if(a != 2) {
			throw a
		}

		let ctx2 = runtime.context.clone()
		a = ctx2.exec("foo.get")
		if(a != 2) {
			throw a
		}

		ctx2.pluginManager = pm.clone()
		a = ctx2.exec("foo.get")
		if(a != 3) {
			throw a
		}
	`)
}

// Tests: finalize
func TestFinalize1(t *testing.T) {
	assertFinalized(t, `	
		function main() {	
			let f = test.newFinalizable();
			runtime.setFinalizer(f)
		}
	`)
}

func TestFinalize2(t *testing.T) {
	assertFinalized(t, `	
		function main() {	
			test.newFinalizable();
		}
	`)
}

func TestFinalizeFunc1(t *testing.T) {
	assertFinalized(t, `		
		function foo() {
			let f = test.newFinalizable();
			runtime.setFinalizer(f)
		}
		
		function main() {
			foo()
		}
	`)
}

func TestFinalizeFunc2(t *testing.T) {
	assertFinalized(t, `		
		function foo() {
			test.newFinalizable();
		}
		
		function main() {
			foo()
		}
	`)
}

func TestFinalizeGlobal(t *testing.T) {
	assertFinalized(t, `		
		function foo() {
			test.newGlobalFinalizable();
		}
		
		function main() {
			foo()
		}
	`)
}

func TestDefer10(t *testing.T) {
	code := `		
		let x			
		function main() {
			 defer(() => { x = 3 })
		}
	`

	vm := assertFinalized(t, code)

	v, _ := vm.RegisterValue("x")
	if v != core.NewValue(3) {
		t.Fatalf("Returned: %v", v)
	}
}

func TestDefer1(t *testing.T) {
	v := runTest(t, `	
		let a = 0

		function foo() {
			 defer(() => {a = 33})
		}

		function main() {
			foo()	
			return a
		}
	`)

	if v.ToInt() != 33 {
		t.Fatal(v)
	}
}

func TestDefer2(t *testing.T) {
	v := runTest(t, `	
		let a = 0

		function foo() {
			defer(() => {a = 33})
			throw "aa"
		}

		function main() {
			try {
				foo()	
			} catch { 
				
			}
			return a
		}
	`)

	if v != core.NewValue(33) {
		t.Fatal(v)
	}
}

func TestDefer3(t *testing.T) {
	v := runTest(t, `	
		let a = 0

		function foo() {
			 defer(() => {a = 33})
			bar()
		}

		function bar() {
			throw "aa"
		}

		function main() {
			try {
				foo()	
			} catch { 
				
			}
			return a
		}
	`)

	if v.ToInt() != 33 {
		t.Fatal(v)
	}
}

func TestDefer4(t *testing.T) {
	runTest(t, `	
		function foo(m: Mutex) {
			m.lock()
			defer(() => m.unlock())
		}

		function main() {			
			let mutex = sync.newMutex()
			foo(mutex)
			foo(mutex)
		}
	`)
}

func TestDefer5(t *testing.T) {
	runTest(t, `
		class Device {
			private mutex = sync.newMutex()

			open() {
        		this.mutex.lock()
			}

			close() {
            	this.mutex.unlock()
			}
		}

		function foo(d: Device) {
			d.open()
			defer(() => d.close())
		}

		function main() {
			let d = new Device()
			foo(d)
			foo(d)
		}
	`)
}

func TestDefer11(t *testing.T) {
	code := `	
		let x = 0;
		
		function foo() {
			 defer(() => { x++ })
		}
		
		function main() {
			foo()
		}
	`

	vm := assertFinalized(t, code)

	v, _ := vm.RegisterValue("x")
	if v != core.NewValue(1) {
		t.Fatalf("Returned: %v", v)
	}
}

func TestDefer12(t *testing.T) {
	code := `	
		let x = 0
		
		function foo() {
			let f = runtime.newFinalizable(() => { x++ })
			runtime.setFinalizer(f)
		}
		
		function main() {
			foo()
		}
	`

	vm := assertFinalized(t, code)

	v, _ := vm.RegisterValue("x")
	if v != core.NewValue(1) {
		t.Fatalf("Returned: %v", v)
	}
}

func TestDefer13(t *testing.T) {
	code := `	
		let x = 0
		
		function bar() {
		     defer(() => { x += 10 })
		    throw "ERRR"
		}
		
		function foo() {
		     defer(() => { x += 5 })
		    bar()
		}
		
		export function main() {
			try {
				foo()
			}
			catch {
		    		x += 1;
			}
			x += 2
		}
	`

	vm := assertFinalized(t, code)

	v, _ := vm.RegisterValue("x")
	if v != core.NewValue(18) {
		t.Fatalf("Returned: %v", v)
	}
}

func TestDeferClosure(t *testing.T) {
	p, err := core.CompileStr(`	
		let ret = 0

		function foo() {
			let a = 1
			defer(() => { ret += a })
			a++
		}

		export function main() {
			foo()
			return ret
		}
	`)

	if err != nil {
		t.Fatal(err)
	}

	v, err := core.NewVM(p).Run()
	if err != nil {
		t.Fatal(err)
	}

	if v != core.NewValue(2) {
		t.Fatalf("Returned: %v", v)
	}
}

type finalizableObj struct {
	finalized bool
}

func (f *finalizableObj) Close() error {
	f.finalized = true
	return nil
}

func assertFinalized(t *testing.T, code string) *core.VM {
	var items []*finalizableObj

	var libs = []core.NativeFunction{
		core.NativeFunction{
			Name: "test.newFinalizable",
			Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
				v := &finalizableObj{}
				vm.SetFinalizer(v)
				items = append(items, v)
				return core.NewObject(v), nil
			},
		},
		core.NativeFunction{
			Name: "test.newGlobalFinalizable",
			Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
				v := &finalizableObj{}
				vm.SetGlobalFinalizer(v)
				items = append(items, v)
				return core.NewObject(v), nil
			},
		},
	}

	vm, err := runExpr(t, code, libs...)
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range items {
		if !v.finalized {
			t.Fatal("Not finalized")
		}
	}

	return vm
}
