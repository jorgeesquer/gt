package lib

import (
	"fmt"
	"github.com/gtlang/gt/lib/x/i18n"
	"github.com/gtlang/gt/core"
	"strings"
	"sync"
	"time"
)

func init() {
	core.RegisterLib(I18N, `

declare namespace i18n {
    export interface Culture {
        name: string
        currencySymbol: string
        currency: string
        currencyPattern: string
        decimalSeparator: string
        thousandSeparator: string
        shortDatePattern: string
        longDatePattern: string
        shortTimePattern: string
        longTimePattern: string
        dateTimePattern: string
        firstDayOfWeek: number
    }

	export const DefaultCulture: Culture
    export function setDefaultCulture(c: string): void

    export function getCultureNames(): StringMap
    export function getCulture(name: string): Culture
    export function format(pattern: string, value: any): string
    export function translate(culture: string, key: string): string
    export function addTranslation(culture: string, key: string, translation: string, tenant?: string): void
    export function addResources(culture: string, name: string, resources: any[]): void


	
	/**
     *  Example:
			"es-ES",
			',',
			'.',
			"EUR",
			"€",
			"0:00€",
			"0:0000€",
			"0:00",
			"dd-MM-yyyy HH:mm",
			"dddd, dd-MM-yyyy HH:mm",
			"dd-MM-yyyy",
			"dddd, dd MMM yyyy",
			"HH:mm",
			"HH:mm:ss",
			time.Monday,
     */
    export function addCulture(
        name: string,
        decimalSeparator: string,
        thousandSeparator: string,
        currency: string,
        currencySymbol: string,
        currencyPattern: string,
        currencyPattern2: string,
        floatPattern: string,
        dateTimePattern: string,
        longDateTimePattern: string,
        shortDatePattern: string,
        longDatePattern: string,
        shortTimePattern: string,
        longTimePattern: string,
        firstDayOfWeek: number): Culture
}
`)
}

var cultureNames map[string]string = make(map[string]string)
var translations map[string]map[string]string = make(map[string]map[string]string)
var trMutex = &sync.RWMutex{}

var I18N = []core.NativeFunction{
	core.NativeFunction{
		Name:      "i18n.setDefaultCulture",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			name := args[0].ToString()
			c, err := i18n.GetCulture(name)
			if err != nil {
				return core.NullValue, err
			}

			i18n.DefaultCulture = c
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "->i18n.DefaultCulture",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			c := core.NewObject(culture{i18n.DefaultCulture})
			return c, nil
		},
	},
	core.NativeFunction{
		Name:      "i18n.getCultureNames",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			m := make(map[string]core.Value, len(translations))

			for k, v := range cultureNames {
				m[k] = core.NewString(v)
			}

			return core.NewMapValues(m), nil
		},
	},
	core.NativeFunction{
		Name:      "i18n.addTranslation",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			ln := len(args)
			if ln < 3 || ln > 4 {
				return core.NullValue, fmt.Errorf("expected 3 or 4 arguments, got %d", ln)
			}

			if err := ValidateOptionalArgs(args, core.String, core.String, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			culture := args[0].ToString()
			key := args[1].ToString()
			translation := args[2].ToString()

			var tenant string
			if ln == 4 {
				if !vm.HasPermission("trusted") {
					// you need to be trusted to add resources to another tenant
					return core.NullValue, ErrUnauthorized
				}
				tenant = args[3].ToString()
			} else {
				tenant = GetContext(vm).Tenant
			}

			tKey := getTenantKey(tenant, key)

			trMutex.Lock()

			resources := translations[culture]
			if resources == nil {
				resources = make(map[string]string)
				translations[culture] = resources
			}
			resources[tKey] = translation

			trMutex.Unlock()

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "i18n.addResources",
		Arguments: 3,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateArgs(args, core.String, core.String, core.Array); err != nil {
				return core.NullValue, err
			}

			culture := args[0].ToString()
			name := args[1].ToString()
			values := args[2].ToArray()

			trMutex.Lock()
			defer trMutex.Unlock()

			cultureNames[culture] = name

			resources := translations[culture]
			if resources == nil {
				resources = make(map[string]string)
				translations[culture] = resources
			}

			for i, v := range values {
				if v.Type != core.Array {
					return core.NullValue, fmt.Errorf("invalid entry at index %d, expected an array but got %s", i, v.TypeName())
				}
				va := v.ToArray()
				// each element of the array is another array with two values: the key and the translation
				if len(va) != 2 {
					return core.NullValue, fmt.Errorf("invalid entry at index %d, the sice of the array must be two but is %d", i, len(va))
				}
				vk := va[0]
				if vk.Type != core.String {
					return core.NullValue, fmt.Errorf("invalid entry at index %d, expected key to be a string but got %s", i, vk.TypeName())
				}
				vv := va[1]
				if vv.Type != core.String {
					return core.NullValue, fmt.Errorf("invalid entry at index %d, expected value to be a string but got %s", i, vv.TypeName())
				}
				resources[vk.ToString()] = vv.ToString()
			}

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "T",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			l := len(args)
			if l == 0 {
				return core.NullValue, fmt.Errorf("expected at least 1 argument, got %d", len(args))
			}

			a := args[0]
			if a.Type == core.Null {
				return core.NullValue, nil
			}

			ctx := GetContext(vm)
			tenant := ctx.Tenant
			key := a.ToString()

			culture := ctx.UserCulture
			if culture == "" {
				culture = ctx.GetCulture().culture.Name
			}

			var value string

			trMutex.RLock()

			tr := translations[culture]
			if tr != nil {
				// try a tenant customized version first
				tKey := getTenantKey(tenant, key)
				if tKey != key {
					if k, ok := tr[tKey]; ok {
						value = k
					}
				}
				// try the default version
				if value == "" {
					if k, ok := tr[key]; ok {
						value = k
					}
				}
			}

			trMutex.RUnlock()

			if value == "" {
				// if there is no translation, remove the context from the key
				if len(key) > 2 {
					// if the transalations symbol has context text remove it
					if key[0] == '@' && key[1] != '@' {
						i := strings.IndexRune(key[1:], '@')
						if i != -1 {
							key = key[i+2:]
						}
					} else {
						key = strings.TrimPrefix(key, "@@")
					}
				}
				value = key
			}

			if l > 1 {
				params := make([]interface{}, l-1)
				for i, vp := range args[1:] {
					if vp.Type == core.String {
						// need to escape the % to prevent interfering with fmt
						vp = core.NewString(strings.Replace(vp.ToString(), "%", "%%", -1))
					}
					params[i] = vp.Export(0)
				}
				value = fmt.Sprintf(value, params...)
			}
			return core.NewString(value), nil
		},
	},
	core.NativeFunction{
		Name:      "i18n.translate",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {

			// translate returns empty if there is no translation

			if err := ValidateArgs(args, core.String, core.String); err != nil {
				return core.NullValue, err
			}

			culture := args[0].ToString()
			tenant := GetContext(vm).Tenant
			key := args[1].ToString()

			trMutex.RLock()
			defer trMutex.RUnlock()

			tr := translations[culture]
			if tr != nil {
				// try a tenant customized version first
				tKey := getTenantKey(tenant, key)
				if tKey != key {
					if k, ok := tr[tKey]; ok {
						return core.NewString(k), nil
					}
				}
				// try the default version
				if k, ok := tr[key]; ok {
					return core.NewString(k), nil
				}
			}

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "i18n.format",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			a := args[0]
			if a.Type != core.String {
				return core.NullValue, fmt.Errorf("expected argument 1 to be a string, got %v", a.TypeName())
			}

			c := GetContext(vm).GetCulture()

			b := args[1].Export(0)
			s := i18n.Format(a.ToString(), b, c.culture)
			return core.NewString(s), nil
		},
	},
	core.NativeFunction{
		Name:      "i18n.getCulture",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateOptionalArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			name := args[0].ToString()
			c, err := i18n.GetCulture(name)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(culture{c}), nil
		},
	},
	core.NativeFunction{
		Name:      "i18n.addCulture",
		Arguments: 15,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			for i, a := range args {
				switch i {
				case 14:
					if a.Type != core.Int {
						return core.NullValue, fmt.Errorf("expected arg %d to be int, got %s", i, a.TypeName())
					}

				default:
					if a.Type != core.String {
						return core.NullValue, fmt.Errorf("expected arg %d to be string, got %s", i, a.TypeName())
					}
				}
			}

			dec := args[1].ToString()
			if len(dec) != 1 {
				return core.NullValue, fmt.Errorf("expected arg 2 to be rune, got %s", args[1].TypeName())
			}

			ths := args[2].ToString()
			if len(dec) != 1 {
				return core.NullValue, fmt.Errorf("expected arg 3 to be rune, got %s", args[2].TypeName())
			}

			c := i18n.Culture{
				Name:                args[0].ToString(),
				DecimalSeparator:    rune(dec[0]),
				ThousandSeparator:   rune(ths[0]),
				Currency:            args[3].ToString(),
				CurrencySymbol:      args[4].ToString(),
				CurrencyPattern:     args[5].ToString(),
				CurrencyPattern2:    args[6].ToString(),
				FloatPattern:        args[7].ToString(),
				DateTimePattern:     args[8].ToString(),
				LongDateTimePattern: args[9].ToString(),
				ShortDatePattern:    args[10].ToString(),
				LongDatePattern:     args[11].ToString(),
				ShortTimePattern:    args[12].ToString(),
				LongTimePattern:     args[13].ToString(),
				FirstDayOfWeek:      time.Weekday(args[14].ToInt()),
			}

			i18n.AddCulture(c)

			return core.NewObject(culture{c}), nil
		},
	},
}

func getTenantKey(tenant, key string) string {
	if tenant != "" {
		return tenant + "::" + key
	}
	return key
}

type culture struct {
	culture i18n.Culture
}

func (c culture) Type() string {
	return "i18n.Culture"
}

func (c culture) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "name":
		return core.NewString(c.culture.Name), nil
	case "currencySymbol":
		return core.NewString(c.culture.CurrencySymbol), nil
	case "currency":
		return core.NewString(c.culture.Currency), nil
	case "currencyPattern":
		return core.NewString(c.culture.CurrencyPattern), nil
	case "decimalSeparator":
		return core.NewString(string(c.culture.DecimalSeparator)), nil
	case "thousandSeparator":
		return core.NewString(string(c.culture.ThousandSeparator)), nil
	case "shortDatePattern":
		return core.NewString(c.culture.ShortDatePattern), nil
	case "longDatePattern":
		return core.NewString(c.culture.LongDatePattern), nil
	case "dateTimePattern":
		return core.NewString(c.culture.DateTimePattern), nil
	case "shortTimePattern":
		return core.NewString(c.culture.DateTimePattern), nil
	case "longTimePattern":
		return core.NewString(c.culture.DateTimePattern), nil
	case "firstDayOfWeek":
		return core.NewInt(int(c.culture.FirstDayOfWeek)), nil
	default:
		return core.UndefinedValue, nil
	}
}

func (c culture) GetMethod(name string) core.NativeMethod {
	switch name {
	case "format":
		return c.format
	}
	return nil
}

func (c culture) format(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 2 {
		return core.NullValue, fmt.Errorf("expected 2 arguments, got %d", len(args))
	}

	a := args[0]
	if a.Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument 1 to be a string, got %v", a.TypeName())
	}

	b := args[1].Export(0)

	s := i18n.Format(a.ToString(), b, c.culture)
	return core.NewString(s), nil
}
