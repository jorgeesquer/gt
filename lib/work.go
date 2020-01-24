package lib

import (
	"fmt"
	"github.com/gtlang/gt/core"
	"sync"
	"time"
)

func init() {
	core.RegisterLib(Work, `

declare namespace sync {
  export function newWorker(): Worker
    export function newJob(): Job

    export interface Worker {
        readonly isRunning: boolean
        errorFunc: (job: any, e: errors.Error) => void
        add(job: Job): void
        start(): void
        stop(): void
    }

    export interface Job {
        idTask?: number
        params: any
        timeout?: time.Duration
        workFunc: (params: any) => void
    }
}
`)
}

var Work = []core.NativeFunction{
	core.NativeFunction{
		Name:      "sync.newWorker",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateOptionalArgs(args, core.Int); err != nil {
				return core.NullValue, err
			}

			var size int
			if len(args) == 1 {
				size = int(args[0].ToInt())
			} else {
				size = 100
			}

			w := &worker{
				worker: NewWorker(size),
			}
			vm.SetGlobalFinalizer(w)

			return core.NewObject(w), nil
		},
	},
	core.NativeFunction{
		Name:      "sync.newJob",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			j := &job{
				workFunc: -1,
			}
			return core.NewObject(j), nil
		},
	},
}

type worker struct {
	errorFunc int
	worker    *Worker
}

func (w *worker) Type() string {
	return "sync.Worker"
}

func (w *worker) Close() error {
	w.worker.Stop()
	return nil
}

func (w *worker) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "isRunning":
		return core.NewBool(w.worker.Running), nil
	case "errorFunc":
		return core.NewFunction(w.errorFunc), nil
	}
	return core.UndefinedValue, nil
}

func (w *worker) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "errorFunc":
		f, err := getFunction(v, vm)
		if err != nil {
			return err
		}
		w.errorFunc = f
		w.worker.ErrorFunc = func(j Job, err error) {
			vm.Error = nil // reset the error
			if _, err := vm.RunFuncIndex(f, core.NewValue(j.Params), core.NewObject(err)); err != nil {
				panic(fmt.Errorf("snap!! Error of the error!! %v", vm.Error))
			}
		}
		return nil
	}
	return ErrReadOnlyOrUndefined
}

func (w *worker) GetMethod(name string) core.NativeMethod {
	switch name {
	case "add":
		return w.add
	case "start":
		return w.start
	case "stop":
		return w.stop
	}
	return nil
}

func (w *worker) add(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	j, ok := args[0].ToObject().(*job)
	if !ok {
		return core.NullValue, fmt.Errorf("expected a job object, got %s", args[0].TypeName())
	}

	if j.workFunc == -1 {
		return core.NullValue, fmt.Errorf("invalid workFunc")
	}

	b := Job{
		Params: j.params,
		WorkFunc: func() error {
			return w.run(vm, j.workFunc, j.params)
		},
	}

	w.worker.Add(b)

	return core.NullValue, nil
}

func (w *worker) run(vm *core.VM, funcIndex int, args ...core.Value) error {
	cvm := vm.Clone(vm.Program, vm.Globals())
	_, err := cvm.RunFuncIndex(funcIndex, args...)
	return err
}

func (w *worker) start(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}
	w.worker.Start()
	return core.NullValue, nil
}

func (w *worker) stop(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}
	w.worker.Stop()
	return core.NullValue, nil
}

type job struct {
	idTask   int
	params   core.Value
	timeout  time.Duration
	workFunc int
}

func (j *job) Type() string {
	return "sync.Job"
}

func (j *job) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "idTask":
		return core.NewInt(j.idTask), nil
	case "params":
		return j.params, nil
	case "timeout":
		return core.NewObject(Duration(j.timeout)), nil
	case "workFunc":
		return core.NewInt(j.workFunc), nil
	}
	return core.UndefinedValue, nil
}

func (j *job) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "idTask":
		if v.Type != core.Int {
			return ErrInvalidType
		}
		j.idTask = int(v.ToInt())
		return nil

	case "params":
		j.params = v
		return nil

	case "timeout":
		switch v.Type {
		case core.Object:
			d, ok := v.ToObject().(Duration)
			if !ok {
				return ErrInvalidType
			}
			j.timeout = time.Duration(d)
			return nil
		case core.Int:
			j.timeout = time.Duration(v.ToInt())
			return nil

		default:
			return ErrInvalidType
		}

	case "workFunc":
		f, err := getFunction(v, vm)
		if err != nil {
			return err
		}
		j.workFunc = f
		return nil
	}

	return ErrReadOnlyOrUndefined
}

func getFunction(v core.Value, vm *core.VM) (int, error) {
	if v.Type != core.Func {
		return 0, ErrInvalidType
	}
	i := v.ToFunction()
	p := vm.Program
	if i < 0 || i > len(p.Functions) {
		return 0, fmt.Errorf("argument out of range")
	}
	return i, nil
}

func NewWorker(bufferSize int) *Worker {
	return &Worker{
		JobChan:  make(chan Job, bufferSize),
		QuitChan: make(chan bool),
	}
}

type Worker struct {
	mut       sync.Mutex
	Running   bool
	JobChan   chan Job
	QuitChan  chan bool
	ErrorFunc func(Job, error) // for unhandled errors
}

type Job struct {
	WorkFunc  func() error
	ErrorFunc func(error) error
	Params    interface{}
	Timeout   time.Duration
}

func (w *Worker) Add(j Job) {
	if j.Timeout == 0 {
		j.Timeout = 1 * time.Minute
	}

	go func() {
		w.JobChan <- j
	}()
}

func (w *Worker) Start() {
	w.mut.Lock()
	if w.Running {
		return
	}
	w.Running = true
	w.mut.Unlock()

	go func() {
		for {
			select {
			case job := <-w.JobChan:
				w.runJob(job)

			case <-w.QuitChan:
				w.mut.Lock()
				w.Running = false
				w.mut.Unlock()
				return
			}
		}
	}()
}

func (w *Worker) Stop() {
	go func() {
		w.QuitChan <- true
	}()
}

func (w *Worker) runJob(j Job) {
	c := make(chan error, 1)

	go func() {
		c <- j.WorkFunc()
	}()

	var e error

	select {
	case err := <-c:
		e = err
	case <-time.After(j.Timeout):
		e = fmt.Errorf("timeout")
	}

	if e != nil {
		if w.ErrorFunc != nil {
			w.ErrorFunc(j, e)
		} else {
			fmt.Println("Worker unhandled error", e)
		}
	}
}
