package core

import (
	"fmt"
	"sync"
)

func newInstance(class string, vm *VM) *instance {
	return &instance{
		iMap:    make(map[string]Value),
		class:   class,
		program: vm.Program,
	}
}

type instance struct {
	sync.RWMutex
	iMap    map[string]Value
	class   string
	program *Program
}

func (i *instance) String() string {
	return "[" + i.class + "]"
}

func (i *instance) methodName(name string) string {
	return i.class + ".prototype." + name
}

func (i *instance) GetProperty(name string, vm *VM) (Value, error) {
	var v Value

	// first try if it has a class method by that name
	mn := i.methodName(name)

	f, ok := i.program.Function(mn)
	if ok {
		if i.program != vm.Program {
			return NullValue, fmt.Errorf("can't call a method of an object from a different program")
		}
		m := method{fn: f.Index, this: NewObject(i)}
		return NewObject(m), nil
	}

	// then look for a property
	i.RLock()
	v = i.iMap[name]
	i.RUnlock()

	return v, nil
}

func (i *instance) SetProperty(name string, v Value, vm *VM) error {
	i.Lock()
	i.iMap[name] = v
	i.Unlock()
	return nil
}
