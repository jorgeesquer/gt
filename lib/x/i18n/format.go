package i18n

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

func Format(format string, v interface{}, c Culture) string {
	if v == nil {
		return ""
	}

	switch format {
	case "c":
		format = c.CurrencyPattern

	case "C":
		format = c.CurrencyPattern2

	case "f":
		format = c.FloatPattern
	}

	switch t := v.(type) {
	case int:
		if format == "" {
			format = "0."
		}
		return formatNum(format, float64(t), c.ThousandSeparator, c.DecimalSeparator)
	case int32:
		if format == "" {
			format = "0."
		}
		return formatNum(format, float64(t), c.ThousandSeparator, c.DecimalSeparator)
	case int64:
		if format == "" {
			format = "0."
		}
		return formatNum(format, float64(t), c.ThousandSeparator, c.DecimalSeparator)
	case float64:
		if format == "" {
			format = "0:00"
		}
		return formatNum(format, t, c.ThousandSeparator, c.DecimalSeparator)
	default:
		if format == "" {
			format = "%v"
		}
		return fmt.Sprintf(format, v)
	}
}

func formatNum(format string, num float64, thousandsSep, decimalSep rune) string {
	if decimalSep == 0 {
		decimalSep = '.'
	}

	left, right, start, end, thousands := parseNumPattern(format)

	s := strconv.FormatFloat(num, 'f', right, 64)
	i := strings.IndexRune(s, '.')
	if i == -1 {
		i = len(s)
	}

	buf := new(bytes.Buffer)

	if start > 0 {
		buf.WriteString(format[:start])
	}

	if !thousands {
		thousandsSep = 0
	}

	formatLeft(s[:i], left, thousandsSep, buf)

	if right > 0 {
		buf.WriteRune(decimalSep)
		buf.WriteString(s[i+1:])
	}

	if len(format) > end {
		buf.WriteString(format[end+1:])
	}

	return buf.String()
}

func formatLeft(num string, length int, thousands rune, buf *bytes.Buffer) {
	if num[0] == '-' {
		buf.WriteRune('-')
		num = num[1:]
	}

	// num of zeros to prepend
	l := len(num)
	pad := length - l
	if pad < 0 {
		pad = 0
	}
	l += pad

	t := l % 3

	for i := 0; i < l; i++ {
		// left padd with zeros
		if i < pad {
			buf.WriteRune('0')
			continue
		}

		// write thousands separator
		if thousands != 0 {
			if i > 0 && i%3-t == 0 {
				buf.WriteRune(thousands)
			}
		}

		buf.WriteByte(num[i-pad])
	}
}

// the format is 0:00 with thousands separator or 0.00 without.
// Check the tests to see examples.
func parseNumPattern(format string) (left, right, start, end int, thousands bool) {
	status := 0 // 0=init, 1=int, 2=decimals
	var i int
	for i, r := range format {
		switch r {
		case '0':
			if status > 0 {
				end = i
			}
			switch status {
			case 0:
				status = 1
				start = i
				left++
			case 1:
				left++
			case 2:
				right++
			}
		case ':':
			if status > 0 {
				end = i
			}
			thousands = true
			status = 2
		case '.':
			if status > 0 {
				end = i
			}
			status = 2
		default:
			if status > 0 {
				return
			}
		}
	}
	if end == 0 {
		end = i
	}
	return
}
