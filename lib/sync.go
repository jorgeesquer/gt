package lib

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(Sync, `
	
declare function go(f: Function): void

declare namespace sync {
    export function newMutex(): Mutex
    export function newWaitGroup(concurrency?: number): WaitGroup
    export function newTicker(duration: number, func: Function): Ticker
    export function newTimer(duration: number, func: Function): Ticker

    export interface WaitGroup {
        go(f: Function): void
        wait(): void
    }

    export interface Mutex {
        lock(): void
        unlock(): void
    }

    export interface Ticker {
        stop(): void
    }

    export function newChannel(buffer?: number): Channel

    export function select(channels: Channel[], defaultCase?: boolean): { index: number, value: any, receivedOK: boolean }

    export interface Channel {
        send(v: any): void
        receive(): any
        close(): void
    }
}

	`)
}

var Sync = []core.NativeFunction{
	core.NativeFunction{
		Name:      "go",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("sync") {
				return core.NullValue, ErrUnauthorized
			}

			return launchGoroutine(args, vm, nil)
		},
	},
	core.NativeFunction{
		Name:      "sync.newTicker",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("sync") {
				return core.NullValue, ErrUnauthorized
			}

			a := args[0]
			if a.Type != core.Int {
				return core.NullValue, fmt.Errorf("expected duration (int), got: %s", a.TypeName())
			}

			v := args[1]
			switch v.Type {
			case core.Func:

			case core.Object:
				if _, ok := v.ToObjectOrNil().(core.Closure); !ok {
					return core.NullValue, fmt.Errorf("expected a function, got: %s", v.TypeName())
				}

			default:
				return core.NullValue, fmt.Errorf("expected a function, got: %s", v.TypeName())
			}

			d := time.Duration(a.ToInt())
			ticker := time.NewTicker(d)

			go func() {
				for range ticker.C {
					if err := runFuncOrClosure(vm, v); err != nil {
						fmt.Println(err)
					}
				}
			}()

			return core.NewObject(&tickerObj{ticker}), nil
		},
	},
	core.NativeFunction{
		Name:      "sync.newTimer",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("sync") {
				return core.NullValue, ErrUnauthorized
			}

			d, err := ToDuration(args[0])
			if err != nil {
				return core.NullValue, fmt.Errorf("expected time.Duration, got: %s", args[0].TypeName())
			}

			v := args[1]
			switch v.Type {
			case core.Func:
			case core.Object:
			default:
				return core.NullValue, fmt.Errorf("expected a function, got: %s", v.TypeName())
			}

			timer := time.NewTimer(d)

			go func() {
				for range timer.C {
					if err := runFuncOrClosure(vm, v); err != nil {
						fmt.Println(err)
					}
				}
			}()

			return core.NewObject(&timerObj{timer}), nil
		},
	},
	core.NativeFunction{
		Name:      "sync.newWaitGroup",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("sync") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateOptionalArgs(args, core.Int); err != nil {
				return core.NullValue, err
			}

			wg := &waitGroup{w: &sync.WaitGroup{}}

			if len(args) == 1 {
				concurrency := int(args[0].ToInt())
				wg.limit = make(chan bool, concurrency)
			}

			return core.NewObject(wg), nil
		},
	},
	core.NativeFunction{
		Name:      "time.after",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			l := len(args)
			if l == 0 || l > 2 {
				return core.NullValue, fmt.Errorf("expected 1 or 2 args")
			}

			d, err := ToDuration(args[0])
			if err != nil {
				return core.NullValue, fmt.Errorf("expected time.Duration, got: %s", args[0].TypeName())
			}

			ch := make(chan core.Value)
			timer := time.NewTimer(d)

			if l == 1 {
				go func() {
					t := <-timer.C
					ch <- core.NewObject(TimeObj(t))
				}()
			} else {
				go func() {
					<-timer.C
					ch <- args[1]
				}()
			}

			c := &channel{c: ch}
			return core.NewObject(c), nil
		},
	},
	core.NativeFunction{
		Name:      "sync.newChannel",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("sync") {
				return core.NullValue, ErrUnauthorized
			}
			if err := ValidateOptionalArgs(args, core.Int); err != nil {
				return core.NullValue, err
			}

			var ch chan core.Value
			var b int
			if len(args) > 0 {
				b = int(args[0].ToInt())
				ch = make(chan core.Value, b)
			} else {
				ch = make(chan core.Value)
			}

			c := &channel{buffer: b, c: ch}
			return core.NewObject(c), nil
		},
	},
	core.NativeFunction{
		Name:      "sync.select",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("sync") {
				return core.NullValue, ErrUnauthorized
			}

			argLen := len(args)
			if argLen == 0 || argLen > 2 {
				return core.NullValue, fmt.Errorf("expected 1 or 2 args, got %d", argLen)
			}

			a := args[0]
			if a.Type != core.Array {
				return core.NullValue, fmt.Errorf("expected arg 1 to be an array of channels, got %s", a.TypeName())
			}

			chans := a.ToArray()
			l := len(chans)
			cases := make([]reflect.SelectCase, l)
			for i, c := range chans {
				ch := c.ToObjectOrNil().(*channel)
				if ch == nil {
					return core.NullValue, fmt.Errorf("invalid channel at index %d", i)
				}
				cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch.c)}
			}

			if argLen == 2 {
				b := args[1]
				if b.Type != core.Bool {
					return core.NullValue, fmt.Errorf("expected arg 2 to be a bool, got %s", b.TypeName())
				}
				if b.ToBool() {
					cases = append(cases, reflect.SelectCase{Dir: reflect.SelectDefault})
				}
			}

			i, value, ok := reflect.Select(cases)

			m := make(map[string]core.Value, 3)
			m["index"] = core.NewInt(i)

			// case default will send an invalid value and will panic if read
			if value.IsValid() {
				m["value"] = value.Interface().(core.Value)
			}

			m["receivedOK"] = core.NewBool(ok)

			return core.NewMapValues(m), nil
		},
	},
	core.NativeFunction{
		Name:      "sync.newMutex",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("createMutex") {
				return core.NullValue, ErrUnauthorized
			}

			m := &mutex{mutex: &sync.Mutex{}}
			return core.NewObject(m), nil
		},
	}}

type mutex struct {
	mutex *sync.Mutex
}

func (mutex) Type() string {
	return "sync.Mutex"
}

func (m *mutex) GetMethod(name string) core.NativeMethod {
	switch name {
	case "lock":
		return m.lock
	case "unlock":
		return m.unlock
	}
	return nil
}

func (m *mutex) lock(args []core.Value, vm *core.VM) (core.Value, error) {
	m.mutex.Lock()
	return core.NullValue, nil
}

func (m *mutex) unlock(args []core.Value, vm *core.VM) (core.Value, error) {
	m.mutex.Unlock()
	return core.NullValue, nil
}

type channel struct {
	buffer int
	c      chan core.Value
}

func (c *channel) Type() string {
	return "sync.Channel"
}

func (c *channel) Size() int {
	return 1
}

func (c *channel) GetMethod(name string) core.NativeMethod {
	switch name {
	case "send":
		return c.send
	case "receive":
		return c.receive
	case "close":
		return c.close
	}
	return nil
}

func (c *channel) send(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 arg")
	}
	c.c <- args[0]
	return core.NullValue, nil
}

func (c *channel) receive(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 args")
	}
	v := <-c.c
	return v, nil
}

func (c *channel) close(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 args")
	}
	close(c.c)
	return core.NullValue, nil
}

type timerObj struct {
	timer *time.Timer
}

func (t *timerObj) Type() string {
	return "sync.Timer"
}

func (t *timerObj) Size() int {
	return 1
}

func (t *timerObj) GetMethod(name string) core.NativeMethod {
	switch name {
	case "reset":
		return t.reset
	case "stop":
		return t.stop
	}
	return nil
}

func (t *timerObj) reset(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 arg, got 0")
	}

	d, err := ToDuration(args[0])
	if err != nil {
		return core.NullValue, fmt.Errorf("expected time.Duration, got: %s", args[0].TypeName())
	}

	t.timer.Reset(d)
	return core.NullValue, nil
}

func (t *timerObj) stop(args []core.Value, vm *core.VM) (core.Value, error) {
	t.timer.Stop()
	return core.NullValue, nil
}

type tickerObj struct {
	ticker *time.Ticker
}

func (t *tickerObj) Type() string {
	return "sync.Ticker"
}

func (t *tickerObj) Size() int {
	return 1
}

func (t *tickerObj) GetMethod(name string) core.NativeMethod {
	switch name {
	case "stop":
		return t.stop
	}
	return nil
}

func (t *tickerObj) stop(args []core.Value, vm *core.VM) (core.Value, error) {
	t.ticker.Stop()
	return core.NullValue, nil
}

type waitGroup struct {
	w     *sync.WaitGroup
	limit chan bool
}

func (t *waitGroup) Type() string {
	return "sync.WaitGroup"
}

func (t *waitGroup) Size() int {
	return 1
}

func (t *waitGroup) GetMethod(name string) core.NativeMethod {
	switch name {
	case "go":
		return t.goRun
	case "wait":
		return t.wait
	}
	return nil
}

func (t *waitGroup) goRun(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("sync") {
		return core.NullValue, ErrUnauthorized
	}

	if t.limit != nil {
		t.limit <- true
	}

	return launchGoroutine(args, vm, t)
}

func (t *waitGroup) wait(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 arguments, got %d", len(args))
	}

	t.w.Wait()
	return core.NullValue, nil
}

func launchGoroutine(args []core.Value, vm *core.VM, t *waitGroup) (core.Value, error) {
	m, err := cloneForAsync(vm)
	if err != nil {
		return core.NullValue, err
	}

	a := args[0]
	switch a.Type {
	case core.Func:
		if t != nil {
			t.w.Add(1)
		}
		go func() {
			_, err := m.RunFuncIndex(a.ToFunction())
			if err != nil {
				// TODO write to the standard logger
				fmt.Println(err)
			}
			if t != nil {
				t.w.Done()
				if t.limit != nil {
					<-t.limit
				}
			}
		}()

	case core.Object:
		c, ok := a.ToObjectOrNil().(core.Closure)
		if !ok {
			return core.NullValue, fmt.Errorf("expected a function, got: %s", a.TypeName())
		}

		if t != nil {
			t.w.Add(1)
		}

		go func() {
			_, err := m.RunClosure(c)
			if err != nil {
				// TODO write to the standard logger
				fmt.Println(err)
			}
			if t != nil {
				t.w.Done()
				if t.limit != nil {
					<-t.limit
				}
			}
		}()

	default:
		return core.NullValue, fmt.Errorf("expected a function, got: %s", a.TypeName())
	}

	return core.NullValue, nil
}

func cloneForAsync(vm *core.VM) (*core.VM, error) {
	m := core.NewInitializedVM(vm.Program, vm.Globals())
	m.MaxAllocations = vm.MaxAllocations
	m.MaxFrames = vm.MaxFrames
	m.MaxSteps = vm.MaxSteps
	m.FileSystem = vm.FileSystem
	m.Trusted = vm.Trusted

	c := GetContext(vm).Clone()
	if c.DB != nil {
		// transactions and concurrency are problematic.
		// For now: any sync code has its own transaction context.
		c.DB = newDB(c.DB.db.Clone(), m)
	}
	m.Context = c

	if err := m.AddSteps(vm.Steps()); err != nil {
		return nil, err
	}

	return m, nil
}
