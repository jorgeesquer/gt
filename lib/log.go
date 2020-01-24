package lib

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gtlang/gt/lib/x/logdb"

	"github.com/gtlang/filesystem"
	"github.com/gtlang/gt/core"
)

func init() {
	core.RegisterLib(Log, `

declare namespace log {
    export const defaultLogger: Logger

    export function setDefaultLogger(logger: Logger): void

    export function fatal(format: any, ...v: any[]): void
    export function system(format: any, ...v: any[]): void
    export function write(table: string, format: any, ...v: any[]): void

    export function newLogger(path: string, fs?: io.FileSystem): Logger

    export interface Logger {
        path: string
		debug: boolean
        save(table: string, data: string, ...v: any): void
        insert(date: time.Time, table: string, data: string, ...v: any[]): void
        query(table: string, start: time.Time, end: time.Time, offset?: number, limit?: number): Scanner
    }

    export interface Scanner {
        scan(): boolean
        data(): DataPoint
        setFilter(v: string): void
    }

    export interface DataPoint {
        text: string
        time: time.Time
        string(): string
    }
}
`)
}

var defaultLogger *logDB

var Log = []core.NativeFunction{
	core.NativeFunction{
		Name:      "->log.defaultLogger",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if defaultLogger == nil {
				return core.NullValue, nil
			}
			return core.NewObject(defaultLogger), nil
		},
	},
	core.NativeFunction{
		Name:      "log.setDefaultLogger",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			db, ok := args[0].ToObjectOrNil().(*logDB)
			if !ok {
				return core.NullValue, fmt.Errorf("expected a logger, got %s", args[0].TypeName())
			}

			defaultLogger = db

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "log.fatal",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			err := writeLog("system", args)
			if err != nil {
				return core.NullValue, err
			}

			os.Exit(1)
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "log.write",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			l := len(args)
			if l < 2 {
				return core.NullValue, fmt.Errorf("expected at least 2 parameters, got %d", len(args))
			}

			table := args[0]
			if table.Type != core.String {
				return core.NullValue, fmt.Errorf("expected parameter 1 to be a string, got %s", table.Type)
			}

			err := writeLog(table.ToString(), args[1:])
			if err != nil {
				return core.NullValue, err
			}

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "log.system",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if len(args) == 0 {
				return core.NullValue, fmt.Errorf("expected at least 1 parameter, got 0")
			}

			err := writeLog("system", args)
			if err != nil {
				return core.NullValue, err
			}

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "log.newLogger",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateOptionalArgs(args, core.String, core.Object); err != nil {
				return core.NullValue, err
			}

			ln := len(args)
			if ln == 0 || ln > 2 {
				return core.NullValue, fmt.Errorf("expected 1 or 2 arguments, got %d", ln)
			}

			path := args[0].ToString()

			var fs filesystem.FS

			if ln == 2 {
				afs, ok := args[1].ToObjectOrNil().(*FileSystemObj)
				if !ok {
					return core.NullValue, fmt.Errorf("invalid argument 2 type: %s", args[1].TypeName())
				}
				fs = afs.FS
			} else {
				fs = filesystem.OS
			}

			t := &logDB{
				db: logdb.New(path, fs),
			}

			return core.NewObject(t), nil
		},
	},
}

func toStringLog(v core.Value) string {
	switch v.Type {
	case core.Null:
		return "<null>"
	case core.String:
		// need to escape the % to prevent interfering with fmt
		return strings.Replace(v.ToString(), "%", "%%", -1)
	}

	return v.String()
}

func writeLog(table string, args []core.Value) error {
	var line string

	ln := len(args)
	if ln == 1 {
		line = toStringLog(args[0])
	} else {
		format := args[0].String()
		values := make([]interface{}, ln-1)
		for i, v := range args[1:] {
			values[i] = toStringLog(v)
		}
		line = fmt.Sprintf(format, values...)
	}

	if defaultLogger == nil {
		fmt.Println(line)
		return nil
	}

	// print system logs in the STD by default in debug mode
	if defaultLogger.debug && table == "system" {
		fmt.Println(line)
	}

	return defaultLogger.db.Save(table, line)
}

type logDB struct {
	db    *logdb.DB
	debug bool
}

func (*logDB) Type() string {
	return "log.DB"
}

func (t *logDB) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "path":
		return core.NewString(t.db.Path), nil
	}
	return core.UndefinedValue, nil
}

func (t *logDB) SetProperty(name string, v core.Value, vm *core.VM) error {
	if !vm.HasPermission("trusted") {
		return ErrUnauthorized
	}

	switch name {
	case "debug":
		if v.Type != core.Bool {
			return ErrInvalidType
		}
		t.debug = v.ToBool()
		return nil
	}

	return ErrReadOnlyOrUndefined
}

func (t *logDB) GetMethod(name string) core.NativeMethod {
	switch name {
	case "save":
		return t.save
	case "insert":
		return t.insert
	case "query":
		return t.query
	case "close":
		return t.close
	}
	return nil
}

func (t *logDB) close(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}
	err := t.db.Close()
	return core.NullValue, err
}

func (t *logDB) save(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String, core.String); err != nil {
		return core.NullValue, err
	}
	err := t.db.Save(args[0].ToString(), args[1].ToString())
	return core.NullValue, err
}

func (t *logDB) insert(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object, core.String, core.String); err != nil {
		return core.NullValue, err
	}

	s, ok := args[0].ToObjectOrNil().(TimeObj)
	if !ok {
		return core.NullValue, fmt.Errorf("expected time, got %s", args[0].TypeName())
	}

	err := t.db.Insert(time.Time(s), args[1].ToString(), args[2].ToString())
	return core.NullValue, err
}

func (t *logDB) query(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateOptionalArgs(args, core.String, core.Object, core.Object, core.Int, core.Int); err != nil {
		return core.NullValue, err
	}

	var table string
	var start, end time.Time
	var offset, limit int

	l := len(args)

	if l == 0 {
		return core.NullValue, fmt.Errorf("expected the table")
	}

	table = args[0].ToString()

	if l == 1 {
		now := time.Now()
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		end = now
	} else if l == 2 {
		s, ok := args[1].ToObjectOrNil().(TimeObj)
		if !ok {
			return core.NullValue, fmt.Errorf("expected time, got %s", args[1].TypeName())
		}
		start = time.Time(s)
		end = time.Now()
	} else if l >= 3 {
		s, ok := args[1].ToObjectOrNil().(TimeObj)
		if !ok {
			return core.NullValue, fmt.Errorf("expected time, got %s", args[1].TypeName())
		}
		start = time.Time(s)
		s, ok = args[2].ToObjectOrNil().(TimeObj)
		if !ok {
			return core.NullValue, fmt.Errorf("expected time, got %s", args[2].TypeName())
		}
		end = time.Time(s)
	}

	if l >= 4 {
		offset = int(args[3].ToInt())
	}

	if l >= 5 {
		limit = int(args[4].ToInt())
	}

	scanner := t.db.Query(table, start, end, offset, limit)
	return core.NewObject(&logDBScanner{scanner}), nil
}

//	err := db.Insert(time.Now(), "log", "something")

//Read:

//	db := New("path/to/data")

//	scanner := db.Query("logs", time.Now(), time.Now())
//	for scanner.Scan() {
//		datapoint := scanner.Data()
//	}

type logDBScanner struct {
	s *logdb.Scanner
}

func (*logDBScanner) Type() string {
	return "logDB.Scanner"
}

func (s *logDBScanner) GetMethod(name string) core.NativeMethod {
	switch name {
	case "scan":
		return s.scan
	case "data":
		return s.data
	case "setFilter":
		return s.setFilter
	}
	return nil
}

func (s *logDBScanner) setFilter(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected a string arg")
	}

	a := args[0]
	if a.Type != core.String {
		return core.NullValue, fmt.Errorf("expected a string arg")
	}

	s.s.SetFilter(a.ToString())
	return core.NullValue, nil
}

func (s *logDBScanner) scan(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	ok := s.s.Scan()
	return core.NewBool(ok), nil
}

func (s *logDBScanner) data(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	dp := s.s.Data()
	return core.NewObject(&dataPoint{dp}), nil
}

type dataPoint struct {
	d logdb.DataPoint
}

func (*dataPoint) Type() string {
	return "logDB.DataPoint"
}

func (d *dataPoint) String() string {
	return d.d.String()
}

func (d *dataPoint) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "text":
		return core.NewString(d.d.Text), nil
	case "time":
		return core.NewObject(TimeObj(d.d.Time)), nil
	}
	return core.UndefinedValue, nil
}

func (d *dataPoint) GetMethod(name string) core.NativeMethod {
	switch name {
	case "string":
		return d.string
	}
	return nil
}

func (d *dataPoint) string(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}
	return core.NewString(d.d.String()), nil
}
