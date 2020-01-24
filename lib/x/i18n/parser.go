package i18n

import (
	"strconv"
	"strings"
)

func ParseInt(v string, c Culture) (int, error) {
	v = strings.Replace(v, string(c.ThousandSeparator), "", -1)
	return strconv.Atoi(v)
}

func ParseFloat(v string, c Culture) (float64, error) {
	v = strings.Replace(v, string(c.ThousandSeparator), "", -1)

	if c.DecimalSeparator != '.' {
		v = strings.Replace(v, string(c.DecimalSeparator), ".", 1)
	}

	return strconv.ParseFloat(v, 64)
}

func ParseCurrency(v string, c Culture) (float64, error) {
	v = strings.Replace(v, c.CurrencySymbol, "", 1)
	return ParseFloat(v, c)
}
