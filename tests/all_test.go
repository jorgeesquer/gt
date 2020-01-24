package tests

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	_ "github.com/gtlang/gt/lib"

	"github.com/gtlang/filesystem"
	"github.com/gtlang/gt/core"
)

func TestTypescript(t *testing.T) {
	var verbose bool
	for _, a := range os.Args {
		if strings.Contains(a, "-test.v=true") {
			verbose = true
			break
		}
	}

	files, err := ioutil.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		name := file.Name()
		if !strings.HasSuffix(name, "_test.ts") {
			continue
		}

		p, err := core.Compile(filesystem.OS, name)
		if err != nil {
			t.Fatal(err)
		}

		for _, fn := range p.Functions {
			if !strings.HasPrefix(fn.Name, "test") {
				continue
			}

			vm := core.NewVM(p)

			// core.Print(p)

			if _, err = vm.RunFunc(fn.Name); err != nil {
				fmt.Printf("    FAIL  %s: %v", fn.Name, err)
			}

			if verbose {
				fmt.Printf("    PASS:  %s\n", fn.Name)
			}
		}
	}
}
