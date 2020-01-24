package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gtlang/gt/core"

	"github.com/fsnotify/fsnotify"
)

func init() {
	core.RegisterLib(FSNotify, `

declare namespace fsnotify {
    export function newWatcher(onEvent: EventHandler): Watcher

    export type EventHandler = (e: Event) => void

	export interface Watcher {
		add(path: string, recursive?: boolean): void
	}
 
	export interface Event {
		name: string
		operation: number
	}

	// const (
	// 	Create Op = 1 << iota
	// 	Write
	// 	Remove
	// 	Rename
	// 	Chmod
	// )
}

`)
}

var FSNotify = []core.NativeFunction{
	core.NativeFunction{
		Name:      "fsnotify.newWatcher",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			v := args[0]
			switch v.Type {
			case core.Func:
			case core.Object:
			default:
				return core.NullValue, fmt.Errorf("expected a function, got: %s", v.TypeName())
			}

			w, err := newFileWatcher(v, vm)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(w), nil
		},
	},
}

func newFileWatcher(fn core.Value, vm *core.VM) (*fsWatcher, error) {
	if !vm.HasPermission("trusted") {
		return nil, ErrUnauthorized
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &fsWatcher{watcher: watcher}
	vm.SetGlobalFinalizer(w)

	w.start(fn, vm)

	return w, nil
}

type fsWatcher struct {
	watcher *fsnotify.Watcher
	closed  bool
}

func (w *fsWatcher) Type() string {
	return "fsnotify.Watcher"
}

func (w *fsWatcher) Close() error {
	return w.watcher.Close()
}

func (w *fsWatcher) GetMethod(name string) core.NativeMethod {
	switch name {
	case "add":
		return w.add
	case "close":
		return w.close
	}
	return nil
}

func (w *fsWatcher) start(fn core.Value, vm *core.VM) {
	go func() {
		for {
			if w.closed {
				break
			}

			select {
			// watch for events
			case event := <-w.watcher.Events:
				if w.closed {
					return
				}
				e := fsEvent{
					name:      event.Name,
					operation: int(event.Op),
				}

				if err := runFuncOrClosure(vm, fn, core.NewObject(e)); err != nil {
					fmt.Println(err)
				}

			// watch for errors
			case err := <-w.watcher.Errors:
				if w.closed {
					return
				}
				fmt.Println("FsWatcher ERROR", err)
			}
		}
	}()
}

func (w *fsWatcher) add(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	dir := args[0].ToString()

	fi, err := os.Stat(dir)
	if err != nil {
		return core.NullValue, err
	}

	if !fi.Mode().IsDir() {
		err := w.watcher.Add(dir)
		return core.NullValue, err
	}

	// if it is a directory add it recursively
	if err := filepath.Walk(dir, w.watchDir); err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (w *fsWatcher) close(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	w.closed = true

	if err := w.watcher.Close(); err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (w *fsWatcher) watchDir(path string, fi os.FileInfo, err error) error {
	if fi.Mode().IsDir() {
		return w.watcher.Add(path)
	}
	return nil
}

type fsEvent struct {
	name      string
	operation int
}

func (e fsEvent) Type() string {
	return "fsnotify.Event"
}

func (e fsEvent) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "name":
		return core.NewString(e.name), nil
	case "operation":
		return core.NewInt(e.operation), nil
	}
	return core.UndefinedValue, nil
}
