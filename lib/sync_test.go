package lib

import (
	"testing"

	"github.com/gtlang/gt/core"
)

func TestAsyncClosure(t *testing.T) {
	p, err := core.CompileStr(`	
		function main() {
			let a = 0
			let wg = sync.newWaitGroup()			
			wg.go(() => {
				for(let i = 0; i < 10; i++) {
					a++
				}
			})
			wg.wait()
			return a
		}
	`)

	if err != nil {
		t.Fatal(err)
	}

	vm := core.NewVM(p)
	vm.Trusted = true

	v, err := vm.Run()
	if err != nil {
		t.Fatal(err)
	}

	if v != core.NewValue(10) {
		t.Fatalf("Returned: %v", v)
	}
}
