package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/gtlang/gt/lib"

	"github.com/gtlang/filesystem"
	"github.com/gtlang/gt/core"
	"github.com/gtlang/gt/binary"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	args := os.Args
	if len(args) == 1 {
		log.Fatal("Usage: gt [path]")
		return
	}

	name := args[1]

	if err := exec(name, args[2:]); err != nil {
		log.Fatal(err)
	}
}

func exec(name string, args []string) error {
	p, err := loadProgram(name)
	if err != nil {
		return err
	}

	vm := core.NewVM(p)
	vm.FileSystem = filesystem.OS
	vm.Trusted = true
	
	ln := len(args)
	values := make([]core.Value, ln)
	for i := 0; i < ln; i++ {
		values[i] = core.NewValue(args[i])
	}

	_, err = vm.Run(values...)
	return err
}

func loadProgram(name string) (*core.Program, error) {
	path, err := findPath(name)
	if err != nil {
		return nil, err
	}

	// by default source files have a typescript extension
	if strings.HasSuffix(path, ".ts") {
		return core.Compile(filesystem.OS, path)
	}

	// first try to read as compiled
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening %s", path)
	}
	defer f.Close()

	p, err := binary.Read(f)
	if err != nil {
		if err == binary.ErrInvalidHeader {
			// if it is not a compiled program maybe is a source file with a different extension
			return core.Compile(filesystem.OS, path)
		}
		return p, fmt.Errorf("error loading %s: %v", path, err)
	}

	return p, nil
}

func findPath(name string) (string, error) {
	if path := tryPath(name); path != "" {
		return path, nil
	}

	// search in GTPATH
	if !strings.ContainsRune(name, os.PathSeparator) {
		for _, dir := range envDirs() {
			n := filepath.Join(dir, name)
			if path := tryPath(n); path != "" {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("does not exist: %s", name)
}

func tryPath(name string) string {
	if filesystem.Exists(filesystem.OS, name) {
		return name
	}

	test := name + ".gt"
	if filesystem.Exists(filesystem.OS, test) {
		return test
	}

	test = name + ".ts"
	if filesystem.Exists(filesystem.OS, test) {
		return test
	}

	return ""
}

func envDirs() []string {
	path := os.Getenv("GTPATH")
	dirs := strings.Split(path, ":")
	if len(dirs) == 1 && dirs[0] == "" {
		return nil
	}
	return dirs
}
