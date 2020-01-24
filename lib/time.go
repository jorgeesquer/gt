package lib

import (
	"fmt"
	"math"
	"github.com/gtlang/gt/core"
	"time"

	"github.com/gtlang/gt/lib/x/i18n"
)

func init() {
	core.RegisterLib(Time, `

declare namespace time {
    /**
     * The ISO time format.
     */
    export const RFC3339: string
    /**
     * The default date format.
     */
    export const DefaultDateFormat: string

    export const AllWeekDays: number

    export const Nanosecond: number
    export const Microsecond: number
    export const Millisecond: number
    export const Second: number
    export const Minute: number
    export const Hour: number

    export const SecMillis: number
    export const MinMillis: number
    export const HourMillis: number
    export const DayMillis: number

    export function now(): Time
    export function nowUTC(): Time

    export const Monday: number
    export const Tuesday: number
    export const Wednesday: number
    export const Thursday: number
    export const Friday: number
    export const Saturday: number
    export const Sunday: number

    /**
     * The number of nanoseconds since the unix epoch.
     */
    export let unixNano: number

    export interface Location {
        name: string
    }

    export const utc: Location
    export const local: Location

    export function setDefaultLocation(name: string): void

    /**
     * Sets a fixed value for now() for testing.
     */
    export function setFixedNow(t: time.Time): void

    /**
     * Remove a fixed value for now().
     */
    export function unsetFixedNow(): void
    export function loadLocation(name: string): Location

    export function convertFormat(format: string): string

    export function setDayOfWeek(value: number, dayOfWeek: number, active: boolean): number
    export function isDayOfWeekActive(value: number, dayOfWeek: number): boolean

    /**
     * 
     * @param seconds from unix epoch
     */
    export function unix(seconds: number): Time

    export function date(year?: number, month?: number, day?: number, hour?: number, min?: number, sec?: number, loc?: Location): Time
    export function localDate(year?: number, month?: number, day?: number, hour?: number, min?: number, sec?: number): Time

    export function duration(nanoseconds: number | Duration): Duration
    export function toDuration(hour: number, minute?: number, second?: number): Duration
    export function toMilliseconds(hour: number, minute?: number, second?: number): number

    export function daysInMonth(year: number, month: number): number

    export interface Time {
        unix: number
        second: number
        nanosecond: number
        minute: number
        hour: number
        day: number
        /**
         * sunday = 0, monday = 1, ...
         */
        dayOfWeek: number
        month: number
        year: number
        yearDay: number
        location: Location
        /**
         * The time part in milliseconds
         */
        time(): number

        /**
         * Return the date discarding the time part in local time.
         */
        startOfDay(): Time
        /**
         * Returns the las moment of the day in local time
         */
        endOfDay(): Time
        utc(): Time
        local(): Time
        sub(t: Time): Duration
        add(t: Duration | number): Time
        addYears(t: number): Time
        addMonths(t: number): Time
        addDays(t: number): Time
        addHours(t: number): Time
        addMinutes(t: number): Time
        addSeconds(t: number): Time
        addMilliseconds(t: number): Time

        setDate(year?: number, month?: number, day?: number): Time
        setTime(hour?: number, minute?: number, second?: number, millisecond?: number): Time
        setTimeMillis(millis: number): Time

        format(f: string): string
        formatIn(f: string, loc: Location): string
        in(loc: Location): Time
        /**
         * setLocation returns the same time with the location. No conversions
         * are made. 9:00 UTC becomes 9:00 Europe/Madrid
         */
        setLocation(loc: Location): Time
        equal(t: Time): boolean
        after(t: Time): boolean
        afterOrEqual(t: Time): boolean
        before(t: Time): boolean
        beforeOrEqual(t: Time): boolean
        between(t1: Time, t2: Time): boolean
        sameDay(t: Time): boolean
    }

    export interface Duration {
        hours: number
        minutes: number
        seconds: number
        milliseconds: number
        nanoseconds: number
        equal(other: number | Duration): boolean
        greater(other: number | Duration): boolean
        lesser(other: number | Duration): boolean
        add(other: number | Duration): Duration
        sub(other: number | Duration): Duration
        multiply(other: number | Duration): Duration
    }

    export interface Period {
        start?: time.Time
        end?: time.Time
    }

    export function after(d: number | Duration, value?: any): sync.Channel
    export function sleep(millis: number): void
    export function sleep(d: time.Duration): void
    export function parse(value: any, format?: string): time.Time
    export function parseLocal(value: any, format?: string): time.Time
    export function parseInLocation(value: any, format: string, location: Location): time.Time
}



`)
}

const (
	secMillis  = 1000
	minMillis  = 60 * secMillis
	hourMillis = 60 * minMillis
	dayMillis  = 24 * hourMillis
)

var Time = []core.NativeFunction{
	core.NativeFunction{
		Name: "->time.AllWeekDays",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(127), nil
		},
	},
	core.NativeFunction{
		Name:      "time.setDayOfWeek",
		Arguments: 3,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Int, core.Int, core.Bool); err != nil {
				return core.NullValue, err
			}

			value := uint64(args[0].ToInt())
			dayOfWeek := uint(args[1].ToInt())
			active := args[2].ToBool()

			if active {
				value = value | (1 << dayOfWeek)
			} else {
				value = value & ^(1 << dayOfWeek)
			}
			return core.NewInt64(int64(value)), nil
		},
	},
	core.NativeFunction{
		Name:      "time.isDayOfWeekActive",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Int, core.Int); err != nil {
				return core.NullValue, err
			}

			value := uint64(args[0].ToInt())
			dayOfWeek := uint(args[1].ToInt())

			active := ((value >> dayOfWeek) & 1) == 1
			return core.NewBool(active), nil
		},
	},
	core.NativeFunction{
		Name: "->time.Monday",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(int(time.Monday)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.Tuesday",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(int(time.Tuesday)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.Wednesday",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(int(time.Wednesday)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.Thursday",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(int(time.Thursday)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.Friday",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(int(time.Friday)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.Saturday",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(int(time.Saturday)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.Sunday",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(int(time.Sunday)), nil
		},
	},

	core.NativeFunction{
		Name: "->time.SecMillis",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(int(secMillis)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.MinMillis",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(int(minMillis)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.HourMillis",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(int(hourMillis)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.DayMillis",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(int(dayMillis)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.Nanosecond",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(int(time.Nanosecond)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.Microsecond",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(int(time.Microsecond)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.Millisecond",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(int(time.Millisecond)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.Second",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt(int(time.Second)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.Minute",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt64(int64(time.Minute)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.Hour",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewInt64(int64(time.Hour)), nil
		},
	},
	core.NativeFunction{
		Name: "->time.RFC3339",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewString(time.RFC3339), nil
		},
	},
	core.NativeFunction{
		Name: "->time.DefaultDateFormat",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewString("2006-1-2"), nil
		},
	},
	core.NativeFunction{
		Name:      "time.daysInMonth",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Int, core.Int); err != nil {
				return core.NullValue, err
			}

			year := args[0].ToInt()
			month := args[1].ToInt()
			days := time.Date(int(year), time.Month(month), 0, 0, 0, 0, 0, time.UTC).Day()
			return core.NewInt(days), nil
		},
	},
	core.NativeFunction{
		Name:      "time.date",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return getDate(args, vm, time.UTC)
		},
	},
	core.NativeFunction{
		Name:      "time.localDate",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			loc := GetContext(vm).GetLocation()
			return getDate(args, vm, loc)
		},
	},
	core.NativeFunction{
		Name:      "time.duration",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Int); err != nil {
				return core.NullValue, err
			}

			d, err := ToDuration(args[0])
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(Duration(d)), nil
		},
	},
	core.NativeFunction{
		Name:      "time.toDuration",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateOptionalArgs(args, core.Int, core.Int, core.Int); err != nil {
				return core.NullValue, err
			}

			l := len(args)

			if l == 0 {
				return core.NullValue, fmt.Errorf("expected at least one parameter")
			}

			d := args[0].ToInt() * int64(time.Hour)

			if l > 1 {
				d += args[1].ToInt() * int64(time.Minute)
			}

			if l > 2 {
				d += args[2].ToInt() * int64(time.Second)
			}

			return core.NewObject(Duration(d)), nil
		},
	},
	core.NativeFunction{
		Name:      "time.toMilliseconds",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateOptionalArgs(args, core.Int, core.Int, core.Int); err != nil {
				return core.NullValue, err
			}

			l := len(args)

			if l == 0 {
				return core.NullValue, fmt.Errorf("expected at least one parameter")
			}

			m := args[0].ToInt() * 60 * 60 * 1000

			if l > 1 {
				m += args[1].ToInt() * 60 * 1000
			}

			if l > 2 {
				m += args[2].ToInt() * 1000
			}

			return core.NewInt64(m), nil
		},
	},
	core.NativeFunction{
		Name:      "time.unix",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Int); err != nil {
				return core.NullValue, err
			}
			sec := args[0].ToInt()
			t := time.Unix(sec, 0)
			return core.NewObject(TimeObj(t)), nil
		},
	},
	core.NativeFunction{
		Name:      "time.setDefaultLocation",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			name := args[0].ToString()

			l, err := time.LoadLocation(name)
			if err != nil {
				return core.NullValue, fmt.Errorf("error loading timezone %s: %v", name, err)
			}
			time.Local = l

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "time.setFixedNow",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateArgs(args, core.Object); err != nil {
				return core.NullValue, err
			}

			t, ok := args[0].ToObjectOrNil().(TimeObj)
			if !ok {
				return core.NullValue, ErrInvalidType
			}

			GetContext(vm).Now = time.Time(t)
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "time.unsetFixedNow",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			GetContext(vm).Now = time.Time{}
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name: "time.now",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			c := GetContext(vm)
			if !c.Now.IsZero() {
				return core.NewObject(TimeObj(c.Now)), nil
			}
			loc := c.GetLocation()
			t := time.Now().In(loc)
			return core.NewObject(TimeObj(t)), nil
		},
	},
	core.NativeFunction{
		Name: "time.nowUTC",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			n := GetContext(vm).Now
			if !n.IsZero() {
				return core.NewObject(TimeObj(n.UTC())), nil
			}
			return core.NewObject(TimeObj(time.Now().UTC())), nil
		},
	},
	core.NativeFunction{
		Name: "time.unixNano",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			n := GetContext(vm).Now
			if !n.IsZero() {
				return core.NewInt64(n.UnixNano()), nil
			}
			return core.NewInt64(time.Now().UnixNano()), nil
		},
	},
	core.NativeFunction{
		Name:      "time.sleep",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if len(args) != 1 {
				return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
			}

			d, err := ToDuration(args[0])
			if err != nil {
				return core.NullValue, err
			}

			time.Sleep(d)
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name: "->time.utc",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewObject(location{time.UTC}), nil
		},
	},
	core.NativeFunction{
		Name: "->time.local",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			l := GetContext(vm).GetLocation()
			return core.NewObject(location{l}), nil
		},
	},
	core.NativeFunction{
		Name:      "time.loadLocation",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}
			l, err := time.LoadLocation(args[0].ToString())
			if err != nil {
				return core.NullValue, err
			}
			return core.NewObject(location{l}), nil
		},
	},
	core.NativeFunction{
		Name:      "time.parse",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			var value string
			var format string

			switch len(args) {
			case 1:
				if err := ValidateArgs(args, core.String); err != nil {
					return core.NullValue, err
				}
				value = args[0].ToString()
			case 2:
				if err := ValidateArgs(args, core.String, core.String); err != nil {
					return core.NullValue, err
				}
				value = args[0].ToString()
				format = args[1].ToString()
			default:
				return core.NullValue, fmt.Errorf("expected 1 or 2 params, got %d", len(args))
			}

			t, err := parseDate(value, format, time.UTC)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(TimeObj(t)), nil
		},
	},
	core.NativeFunction{
		Name:      "time.parseLocal",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			l := GetContext(vm).GetLocation()
			return parseInLocation(l, this, args, vm)
		},
	},
	core.NativeFunction{
		Name:      "time.parseInLocation",
		Arguments: 3,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			location, ok := args[2].ToObjectOrNil().(location)
			if !ok {
				return core.NullValue, fmt.Errorf("invalid location, got %s", args[2].TypeName())
			}
			return parseInLocation(location.l, this, args, vm)
		},
	},
	core.NativeFunction{
		Name:      "time.convertFormat",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			format := i18n.CSharpStyleToGo(args[0].ToString())
			return core.NewString(format), nil
		},
	},
	core.NativeFunction{
		Name:      "time.formatMinutes",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			var min float64

			a := args[0]

			switch a.Type {
			case core.Int, core.Float:
				min = a.ToFloat()

			default:
				return core.NullValue, fmt.Errorf("expected a number, got %v", a.TypeName())
			}

			negative := min < 0

			min = math.Abs(min)

			h := int(math.Floor(min / 60))
			m := int(min) % 60

			s := fmt.Sprintf("%02d:%02d", h, m)

			if negative {
				s = "-" + s
			}

			return core.NewString(s), nil
		},
	},
}

func parseDate(value, format string, loc *time.Location) (time.Time, error) {
	var formats []string

	if format != "" {
		formats = []string{format}
	} else {
		formats = []string{
			"2006-01-02",
			"2006-01-02T15:04:05",
			"2006-01-02T15:04:05Z07:00",
			"Mon Jan 02 2006 15:04:05 GMT-0700 (MST)",
		}
	}

	for _, f := range formats {
		t, err := time.ParseInLocation(f, value, loc)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, core.NewPublicError(fmt.Sprintf("Error parsing date: %s", value))
}

func getDate(args []core.Value, vm *core.VM, defaultLoc *time.Location) (core.Value, error) {
	if err := ValidateOptionalArgs(args, core.Int, core.Int,
		core.Int, core.Int, core.Int, core.Int, core.Object); err != nil {
		return core.NullValue, err
	}

	var loc *time.Location
	var year, month, day, hour, min, sec int

	switch len(args) {
	case 0:
		year = 1
		month = 1
		day = 1
		loc = defaultLoc

	case 1:
		year = int(args[0].ToInt())
		month = 1
		day = 1
		loc = defaultLoc

	case 2:
		year = int(args[0].ToInt())
		month = int(args[1].ToInt())
		day = 1
		loc = defaultLoc

	case 3:
		year = int(args[0].ToInt())
		month = int(args[1].ToInt())
		day = int(args[2].ToInt())
		loc = defaultLoc

	case 4:
		year = int(args[0].ToInt())
		month = int(args[1].ToInt())
		day = int(args[2].ToInt())
		hour = int(args[3].ToInt())
		loc = defaultLoc

	case 5:
		year = int(args[0].ToInt())
		month = int(args[1].ToInt())
		day = int(args[2].ToInt())
		hour = int(args[3].ToInt())
		min = int(args[4].ToInt())
		loc = defaultLoc

	case 6:
		year = int(args[0].ToInt())
		month = int(args[1].ToInt())
		day = int(args[2].ToInt())
		hour = int(args[3].ToInt())
		min = int(args[4].ToInt())
		sec = int(args[5].ToInt())
		loc = defaultLoc

	case 7:
		year = int(args[0].ToInt())
		month = int(args[1].ToInt())
		day = int(args[2].ToInt())
		hour = int(args[3].ToInt())
		min = int(args[4].ToInt())
		sec = int(args[5].ToInt())
		location, ok := args[6].ToObjectOrNil().(location)
		if !ok {
			return core.NullValue, fmt.Errorf("invalid location, got %s", args[6].TypeName())
		}
		loc = location.l
	}

	d := time.Date(year, time.Month(month), day, hour, min, sec, 0, loc)
	return core.NewObject(TimeObj(d)), nil
}

func parseInLocation(l *time.Location, this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgRange(args, 1, 2); err != nil {
		return core.NullValue, err
	}
	if err := ValidateOptionalArgs(args, core.String, core.String); err != nil {
		return core.NullValue, err
	}

	value := args[0].ToString()

	var format string

	if len(args) == 2 {
		farg := args[1]
		switch farg.Type {
		case core.String:
			format = farg.ToString()
		case core.Null:
		default:
			return core.NullValue, ErrInvalidType
		}
	}

	t, err := parseDate(value, format, l)
	if err != nil {
		return core.NullValue, err
	}
	return core.NewObject(TimeObj(t)), nil
}

type TimeObj time.Time

func (t TimeObj) Type() string {
	return "time"
}

func (t TimeObj) Size() int {
	return 1
}

func (t TimeObj) String() string {
	return time.Time(t).Format(time.RFC3339)
}

func (t TimeObj) Export(recursionLevel int) interface{} {
	return time.Time(t)
}

func (t TimeObj) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "unix":
		return core.NewInt64(time.Time(t).Unix()), nil
	case "second":
		return core.NewInt(time.Time(t).Second()), nil
	case "nanosecond":
		return core.NewInt(time.Time(t).Nanosecond()), nil
	case "minute":
		return core.NewInt(time.Time(t).Minute()), nil
	case "hour":
		return core.NewInt(time.Time(t).Hour()), nil
	case "day":
		return core.NewInt(time.Time(t).Day()), nil
	case "dayOfWeek":
		return core.NewInt(int(time.Time(t).Weekday())), nil
	case "month":
		return core.NewInt(int(time.Time(t).Month())), nil
	case "year":
		return core.NewInt(time.Time(t).Year()), nil
	case "yearDay":
		return core.NewInt(time.Time(t).YearDay()), nil
	case "location":
		l := time.Time(t).Location()
		if l == nil {
			return core.NullValue, nil
		}
		return core.NewObject(location{l}), nil
	}

	return core.UndefinedValue, nil
}

func (t TimeObj) GetMethod(name string) core.NativeMethod {
	switch name {
	case "sub":
		return t.sub
	case "add":
		return t.add
	case "setDate":
		return t.setDate
	case "setTime":
		return t.setTime
	case "setTimeMillis":
		return t.setTimeMillis
	case "addMilliseconds":
		return t.addMilliseconds
	case "addSeconds":
		return t.addSeconds
	case "addMinutes":
		return t.addMinutes
	case "addHours":
		return t.addHours
	case "addYears":
		return t.addYears
	case "addMonths":
		return t.addMonths
	case "addDays":
		return t.addDays
	case "format":
		return t.format
	case "formatIn":
		return t.formatIn
	case "utc":
		return t.utc
	case "local":
		return t.local
	case "time":
		return t.time
	case "startOfDay":
		return t.startOfDay
	case "endOfDay":
		return t.endOfDay
	case "setLocation":
		return t.setLocation
	case "in":
		return t.in
	case "after":
		return t.after
	case "afterOrEqual":
		return t.afterOrEqual
	case "before":
		return t.before
	case "beforeOrEqual":
		return t.beforeOrEqual
	case "between":
		return t.between
	case "equal":
		return t.equal
	case "sameDay":
		return t.sameDay
	}
	return nil
}

func (t TimeObj) setDate(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l > 3 {
		return core.NullValue, fmt.Errorf("expected max 3 arguments, got %d", l)
	}

	tt := time.Time(t)
	year := tt.Year()
	month := tt.Month()
	day := tt.Day()
	dur := tt.Sub(time.Date(year, month, day, 0, 0, 0, 0, tt.Location()))

	if l >= 1 {
		a := args[0]
		switch a.Type {
		case core.Null, core.Undefined:
		case core.Int:
			year = int(a.ToInt())
		default:
			return core.NullValue, ErrInvalidType
		}
	}

	if l >= 2 {
		a := args[1]
		switch a.Type {
		case core.Null, core.Undefined:
		case core.Int:
			month = time.Month(a.ToInt())
		default:
			return core.NullValue, ErrInvalidType
		}
	}

	if l >= 3 {
		a := args[2]
		switch a.Type {
		case core.Null, core.Undefined:
		case core.Int:
			day = int(a.ToInt())
		default:
			return core.NullValue, ErrInvalidType
		}
	}

	date := time.Date(year, month, day, 0, 0, 0, 0, tt.Location())
	date = date.Add(dur)

	return core.NewObject(TimeObj(date)), nil
}

func (t TimeObj) setTimeMillis(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Int); err != nil {
		return core.NullValue, err
	}

	millis := int(args[0].ToInt())

	hour := millis / (60 * 60 * 1000)

	mod := millis % (60 * 60 * 1000)

	min := mod / (60 * 1000)

	secMod := mod % (60 * 1000)

	sec := secMod / 1000

	ms := secMod % 1000

	loc := GetContext(vm).GetLocation()

	// always operate with time in local time
	tt := time.Time(t).In(loc)

	date := time.Date(tt.Year(), tt.Month(), tt.Day(), hour, min, sec, ms, tt.Location())

	return core.NewObject(TimeObj(date)), nil
}

func (t TimeObj) setTime(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateOptionalArgs(args, core.Int, core.Int, core.Int, core.Int); err != nil {
		return core.NullValue, err
	}

	var hour, min, sec, milli int

	l := len(args)

	if l >= 1 {
		hour = int(args[0].ToInt())
	}

	if l >= 2 {
		min = int(args[1].ToInt())
	}

	if l >= 3 {
		sec = int(args[2].ToInt())
	}

	if l >= 4 {
		milli = int(args[3].ToInt())
	}

	loc := GetContext(vm).GetLocation()

	// always operate with time in local time
	tt := time.Time(t).In(loc)

	date := time.Date(tt.Year(), tt.Month(), tt.Day(), hour, min, sec, milli, tt.Location())

	return core.NewObject(TimeObj(date)), nil
}

func (t TimeObj) sameDay(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	t2, ok := args[0].ToObject().(TimeObj)
	if !ok {
		return core.NullValue, fmt.Errorf("expected time.Time, got %s", args[0].TypeName())
	}

	tt1 := time.Time(t)
	tt2 := time.Time(t2)
	eq := tt1.Year() == tt2.Year() && tt1.Month() == tt2.Month() && tt1.Day() == tt2.Day()
	return core.NewBool(eq), nil
}

func (t TimeObj) equal(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	t2, ok := args[0].ToObject().(TimeObj)
	if !ok {
		return core.NullValue, fmt.Errorf("expected time.Time, got %s", args[0].TypeName())
	}

	eq := time.Time(t).Equal(time.Time(t2))
	return core.NewBool(eq), nil
}

func (t TimeObj) after(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	t2, ok := args[0].ToObject().(TimeObj)
	if !ok {
		return core.NullValue, fmt.Errorf("expected time.Time, got %s", args[0].TypeName())
	}

	after := time.Time(t).After(time.Time(t2))
	return core.NewBool(after), nil
}

func (t TimeObj) afterOrEqual(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	t2, ok := args[0].ToObject().(TimeObj)
	if !ok {
		return core.NullValue, fmt.Errorf("expected time.Time, got %s", args[0].TypeName())
	}

	after := !time.Time(t).Before(time.Time(t2))
	return core.NewBool(after), nil
}

func (t TimeObj) before(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	t2, ok := args[0].ToObject().(TimeObj)
	if !ok {
		return core.NullValue, fmt.Errorf("expected time.Time, got %s", args[0].TypeName())
	}

	after := time.Time(t).Before(time.Time(t2))
	return core.NewBool(after), nil
}

func (t TimeObj) beforeOrEqual(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	t2, ok := args[0].ToObject().(TimeObj)
	if !ok {
		return core.NullValue, fmt.Errorf("expected time.Time, got %s", args[0].TypeName())
	}

	after := !time.Time(t).After(time.Time(t2))
	return core.NewBool(after), nil
}

func (t TimeObj) between(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object, core.Object); err != nil {
		return core.NullValue, err
	}

	t1, ok := args[0].ToObject().(TimeObj)
	if !ok {
		return core.NullValue, fmt.Errorf("expected time.Time, got %s", args[0].TypeName())
	}

	t2, ok := args[1].ToObject().(TimeObj)
	if !ok {
		return core.NullValue, fmt.Errorf("expected time.Time, got %s", args[1].TypeName())
	}

	t0 := time.Time(t)

	between := !t0.Before(time.Time(t1)) && !t0.After(time.Time(t2))

	return core.NewBool(between), nil
}

// SetLocation returns exactly the same time but with a different location
func (t TimeObj) setLocation(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	loc, ok := args[0].ToObject().(location)
	if !ok {
		return core.NullValue, fmt.Errorf("expected location, got %s", args[0].TypeName())
	}

	a := time.Time(t)
	b := time.Date(a.Year(), a.Month(), a.Day(),
		a.Hour(), a.Minute(), a.Second(), a.Nanosecond(), loc.l)

	return core.NewObject(TimeObj(b)), nil
}

func (t TimeObj) in(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	loc, ok := args[0].ToObject().(location)
	if !ok {
		return core.NullValue, fmt.Errorf("expected location, got %s", args[0].TypeName())
	}

	tt := time.Time(t).In(loc.l)

	return core.NewObject(TimeObj(tt)), nil
}

func (t TimeObj) startOfDay(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 arguments, got %d", len(args))
	}

	loc := GetContext(vm).GetLocation()

	tt := time.Time(t).In(loc)

	// construct a new date ignoring the time part
	u := TimeObj(time.Date(tt.Year(), tt.Month(), tt.Day(), 0, 0, 0, 0, loc))

	return core.NewObject(u), nil
}

func (t TimeObj) endOfDay(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 arguments, got %d", len(args))
	}

	loc := GetContext(vm).GetLocation()

	tt := time.Time(t).In(loc)

	u := time.Date(tt.Year(), tt.Month(), tt.Day(), 0, 0, 0, 0, loc)
	u = u.AddDate(0, 0, 1).Add(-1 * time.Second)
	return core.NewObject(TimeObj(u)), nil
}

// return the milliseconds of the time part
func (t TimeObj) time(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 arguments, got %d", len(args))
	}

	loc := GetContext(vm).GetLocation()

	// always operate with time in local time
	tt := time.Time(t).In(loc)

	millis := (tt.Hour() * 60 * 60 * 1000) + (tt.Minute() * 60 * 1000) + (tt.Second() * 1000)

	return core.NewInt(millis), nil
}

func (t TimeObj) utc(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 arguments, got %d", len(args))
	}

	u := TimeObj(time.Time(t).UTC())

	return core.NewObject(u), nil
}

func (t TimeObj) local(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 arguments, got %d", len(args))
	}

	loc := GetContext(vm).GetLocation()

	l := TimeObj(time.Time(t).In(loc))
	return core.NewObject(l), nil
}

func (t TimeObj) sub(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	var a = args[0]
	if a.Type != core.Object {
		return core.NullValue, fmt.Errorf("expected time, got %s", a.TypeName())
	}

	at, ok := a.ToObject().(TimeObj)
	if !ok {
		return core.NullValue, fmt.Errorf("expected time, got %s", a.TypeName())
	}

	d := time.Time(t).Sub(time.Time(at))

	return core.NewObject(Duration(d)), nil
}

func (t TimeObj) add(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	var a = args[0]
	var ad time.Duration

	switch a.Type {
	case core.Int:
		ad = time.Duration(a.ToInt())
	case core.Object:
		dur, ok := a.ToObject().(Duration)
		if !ok {
			return core.NullValue, fmt.Errorf("expected duration, got %s", a.TypeName())
		}
		ad = time.Duration(dur)
	}

	d := time.Time(t).Add(ad)

	return core.NewObject(TimeObj(d)), nil
}

func parseAddArg(a core.Value) (int, error) {
	switch a.Type {
	case core.Int:
		return int(a.ToInt()), nil
	case core.Float:
		f := a.ToFloat()
		if f != float64(int(f)) {
			return 0, fmt.Errorf("expected int, got %s", a.TypeName())
		}
		return int(f), nil
	default:
		return 0, fmt.Errorf("expected int, got %s", a.TypeName())
	}

}

func (t TimeObj) addYears(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	i, err := parseAddArg(args[0])
	if err != nil {
		return core.NullValue, err
	}

	return t.addDate(int(i), 0, 0)
}

func (t TimeObj) addMonths(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	i, err := parseAddArg(args[0])
	if err != nil {
		return core.NullValue, err
	}

	return t.addDate(0, int(i), 0)
}

func (t TimeObj) addDays(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	i, err := parseAddArg(args[0])
	if err != nil {
		return core.NullValue, err
	}

	return t.addDate(0, 0, int(i))
}

func (t TimeObj) addHours(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	i, err := parseAddArg(args[0])
	if err != nil {
		return core.NullValue, err
	}

	d := time.Time(t).Add(time.Duration(i) * time.Hour)

	return core.NewObject(TimeObj(d)), nil
}

func (t TimeObj) addMinutes(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	i, err := parseAddArg(args[0])
	if err != nil {
		return core.NullValue, err
	}

	d := time.Time(t).Add(time.Duration(i) * time.Minute)
	return core.NewObject(TimeObj(d)), nil
}

func (t TimeObj) addMilliseconds(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	i, err := parseAddArg(args[0])
	if err != nil {
		return core.NullValue, err
	}

	d := time.Time(t).Add(time.Duration(i) * time.Millisecond)
	return core.NewObject(TimeObj(d)), nil
}

func (t TimeObj) addSeconds(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	i, err := parseAddArg(args[0])
	if err != nil {
		return core.NullValue, err
	}

	d := time.Time(t).Add(time.Duration(i) * time.Second)
	return core.NewObject(TimeObj(d)), nil
}

func (t TimeObj) addDate(years, months, days int) (core.Value, error) {
	d := time.Time(t).AddDate(years, months, days)
	return core.NewObject(TimeObj(d)), nil
}

func (t TimeObj) format(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	format := args[0].ToString()
	c := GetContext(vm)
	cl := c.GetCulture()
	loc := c.GetLocation()
	return formatIn(cl.culture, loc, format, time.Time(t))
}

func (t TimeObj) formatIn(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String, core.Object); err != nil {
		return core.NullValue, err
	}

	format := args[0].ToString()

	loc, ok := args[0].ToObject().(location)
	if !ok {
		return core.NullValue, fmt.Errorf("expected location, got %s", args[0].TypeName())
	}

	c := GetContext(vm)
	cl := c.GetCulture()
	return formatIn(cl.culture, loc.l, format, time.Time(t))
}

func formatIn(cl i18n.Culture, loc *time.Location, format string, t time.Time) (core.Value, error) {
	// convert alias
	switch format {
	case i18n.ShortDatePattern:
		format = i18n.CSharpStyleToGo(cl.ShortDatePattern)

	case i18n.LongDatePattern:
		format = i18n.CSharpStyleToGo(cl.LongDatePattern)

	case i18n.ShortTimePattern:
		format = i18n.CSharpStyleToGo(cl.ShortTimePattern)

	case i18n.LongTimePattern:
		format = i18n.CSharpStyleToGo(cl.LongTimePattern)

	case i18n.DateTimePattern:
		format = i18n.CSharpStyleToGo(cl.DateTimePattern)

	case i18n.LongDateTimePattern:
		format = i18n.CSharpStyleToGo(cl.LongDateTimePattern)
	}

	if loc != nil {
		t = t.In(loc)
	}

	s := t.Format(format)
	return core.NewString(s), nil
}

type Duration time.Duration

func (t Duration) Type() string {
	return "duration"
}

func (t Duration) Size() int {
	return 1
}

func (t Duration) Export(recursionLevel int) interface{} {
	return time.Duration(t)
}

func (t Duration) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "hours":
		return core.NewFloat(time.Duration(t).Hours()), nil
	case "minutes":
		return core.NewFloat(time.Duration(t).Minutes()), nil
	case "seconds":
		return core.NewFloat(time.Duration(t).Seconds()), nil
	case "milliseconds":
		return core.NewInt64(time.Duration(t).Nanoseconds() / 1000000), nil
	case "nanoseconds":
		return core.NewInt64(time.Duration(t).Nanoseconds()), nil
	}

	return core.UndefinedValue, nil
}

func (t Duration) GetMethod(name string) core.NativeMethod {
	switch name {
	case "equal":
		return t.equal
	case "greater":
		return t.greater
	case "lesser":
		return t.lesser
	case "add":
		return t.add
	case "sub":
		return t.sub
	case "multiply":
		return t.multiply
	}
	return nil
}

func (t Duration) add(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	d, err := ToDuration(args[0])
	if err != nil {
		return core.NullValue, err
	}

	return core.NewObject(Duration(time.Duration(t) + d)), nil
}

func (t Duration) sub(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	d, err := ToDuration(args[0])
	if err != nil {
		return core.NullValue, err
	}

	return core.NewObject(Duration(time.Duration(t) - d)), nil
}

func (t Duration) multiply(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	d, err := ToDuration(args[0])
	if err != nil {
		return core.NullValue, err
	}

	return core.NewObject(Duration(time.Duration(t) * d)), nil
}

func (t Duration) equal(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	a := args[0]
	var t2 int

	switch a.Type {
	case core.Object:
		d, ok := a.ToObject().(Duration)
		if !ok {
			return core.NullValue, fmt.Errorf("expected time.Duration, got %s", a.TypeName())
		}
		t2 = int(d)

	case core.Int:
		t2 = int(a.ToInt())

	case core.Float:
		t2 = int(a.ToInt())

	default:
		return core.NullValue, fmt.Errorf("expected time.Duration, got %s", a.TypeName())
	}

	eq := int(t) == t2
	return core.NewBool(eq), nil
}

func (t Duration) greater(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	a := args[0]
	var t2 int

	switch a.Type {
	case core.Object:
		d, ok := a.ToObject().(Duration)
		if !ok {
			return core.NullValue, fmt.Errorf("expected time.Duration, got %s", a.TypeName())
		}
		t2 = int(d)

	case core.Int:
		t2 = int(a.ToInt())

	case core.Float:
		t2 = int(a.ToInt())

	default:
		return core.NullValue, fmt.Errorf("expected time.Duration, got %s", a.TypeName())
	}

	eq := int(t) > t2
	return core.NewBool(eq), nil
}

func (t Duration) lesser(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	a := args[0]
	var t2 int

	switch a.Type {
	case core.Object:
		d, ok := a.ToObject().(Duration)
		if !ok {
			return core.NullValue, fmt.Errorf("expected time.Duration, got %s", a.TypeName())
		}
		t2 = int(d)

	case core.Int:
		t2 = int(a.ToInt())

	case core.Float:
		t2 = int(a.ToInt())

	default:
		return core.NullValue, fmt.Errorf("expected time.Duration, got %s", a.TypeName())
	}

	eq := int(t) < t2
	return core.NewBool(eq), nil
}

type location struct {
	l *time.Location
}

func (l location) Type() string {
	return "time.Location"
}

func (l location) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "name":
		return core.NewString(l.l.String()), nil
	}

	return core.UndefinedValue, nil
}

func ToDuration(v core.Value) (time.Duration, error) {
	switch v.Type {
	case core.Object:
		d, ok := v.ToObject().(Duration)
		if !ok {
			return 0, fmt.Errorf("expected time.Duration, got %s", v.TypeName())
		}
		return time.Duration(d), nil

	case core.Int:
		return time.Duration(v.ToInt()), nil

	default:
		return 0, fmt.Errorf("expected time.Duration, got %s", v.TypeName())
	}
}
