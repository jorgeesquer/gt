// Package i18n implements internacionalization functions
package i18n

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// https://msdn.microsoft.com/es-es/library/az4se3k1(v=vs.110).aspx
const (
	ShortTimePattern    = "t" // HH:mm
	LongTimePattern     = "T" // HH:mm:ss
	ShortDatePattern    = "d"
	LongDatePattern     = "D"
	DateTimePattern     = "g"
	LongDateTimePattern = "G"
	CurrencyPattern     = "c"
	FloatPattern        = "f"
	IntPattern          = "i"
)

var DefaultCulture Culture

var mutex = &sync.RWMutex{}
var cultures map[string]Culture

func AddCulture(c Culture) {
	mutex.Lock()
	cultures[c.Name] = c
	mutex.Unlock()
}

func ValidCulture(name string) bool {
	mutex.RLock()
	_, ok := cultures[name]
	mutex.RUnlock()
	return ok
}

func GetCulture(name string) (Culture, error) {
	if name == "" {
		name = "en-US"
	}

	mutex.RLock()
	c, ok := cultures[name]
	mutex.RUnlock()

	if !ok {
		return Culture{}, fmt.Errorf("invalid culture: %s", name)
	}
	return c, nil
}

type Culture struct {
	Name              string
	DecimalSeparator  rune
	ThousandSeparator rune
	Currency          string
	CurrencySymbol    string
	CurrencyPattern   string
	CurrencyPattern2  string
	FloatPattern      string

	DateTimePattern     string
	LongDateTimePattern string
	ShortDatePattern    string
	LongDatePattern     string
	ShortTimePattern    string
	LongTimePattern     string
	FirstDayOfWeek      time.Weekday
}

// https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes
// he name is a combination of an ISO 639 two-letter lowercase culture code associated with a language and
// an ISO 3166 two-letter uppercase subculture code associated with a country or region.
// https://azuliadesigns.com/list-net-culture-country-codes

func init() {
	cultures = map[string]Culture{
		"es-ES": {
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
		},
		"pt-PT": {
			"pt-PT",
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
		},
		"it-IT": {
			"it-IT",
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
		},
		"en-US": {
			"en-US",
			'.',
			',',
			"USD",
			"$",
			"$0:00",
			"$0:0000",
			"0:00",

			"MM-dd-yyyy HH:mm",
			"dddd, MM-dd-yyyy HH:mm",
			"MM-dd-yyyy",
			"dddd, MMM dd yyyy",
			"HH:mm",
			"HH:mm:ss",
			time.Sunday,
		},
		"es-DO": {
			"es-DO",
			'.',
			',',
			"USD",
			"$",
			"$0:00",
			"$0:0000",
			"0:00",

			"MM-dd-yyyy HH:mm",
			"dddd, MM-dd-yyyy HH:mm",
			"MM-dd-yyyy",
			"dddd, MMM dd yyyy",
			"HH:mm",
			"HH:mm:ss",
			time.Sunday,
		},
		"en-ZA": {
			"en-ZA",
			'.',
			',',
			"RAND",
			"R",
			"R0:00",
			"R0:0000",
			"0:00",

			"MM-dd-yyyy HH:mm",
			"dddd, MM-dd-yyyy HH:mm",
			"MM-dd-yyyy",
			"dddd, MMM dd yyyy",
			"HH:mm",
			"HH:mm:ss",
			time.Sunday,
		},
	}

	DefaultCulture = cultures["es-ES"]
}

// Mon Jan 2 15:04:05 -0700 MST 2006
// https://sclsoftware.com/santi/kbase/post/276
// https://msdn.microsoft.com/es-es/library/az4se3k1(v=vs.110).aspx
// https://msdn.microsoft.com/es-es/library/8kb3ddd4(v=vs.110).aspx

func CSharpStyleToGo(csharpStyle string) string {
	csharpStyle = strings.Replace(csharpStyle, "yyyy", "2006", -1)
	csharpStyle = strings.Replace(csharpStyle, "yy", "06", -1)
	csharpStyle = strings.Replace(csharpStyle, "MMM", "Jan", -1)
	csharpStyle = strings.Replace(csharpStyle, "MM", "01", -1)
	csharpStyle = strings.Replace(csharpStyle, "M", "1", -1)
	csharpStyle = strings.Replace(csharpStyle, "dddd", "Monday", -1)
	csharpStyle = strings.Replace(csharpStyle, "ddd", "Mon", -1)
	csharpStyle = strings.Replace(csharpStyle, "dd", "02", -1)
	csharpStyle = strings.Replace(csharpStyle, "d", "2", -1)
	csharpStyle = strings.Replace(csharpStyle, "HH", "15", -1)
	csharpStyle = strings.Replace(csharpStyle, "hh", "03", -1)
	csharpStyle = strings.Replace(csharpStyle, "h", "3", -1)
	csharpStyle = strings.Replace(csharpStyle, "mm", "04", -1)
	csharpStyle = strings.Replace(csharpStyle, "m", "4", -1)
	csharpStyle = strings.Replace(csharpStyle, "ss", "05", -1)
	csharpStyle = strings.Replace(csharpStyle, "s", "5", -1)
	return csharpStyle
}
