package i18n

import (
	"testing"
)

func TestFormat(t *testing.T) {
	data := []struct {
		culture string
		pattern string
		value   interface{}
		result  string
	}{
		{"es-ES", "0:", 1, "1"},
		{"es-ES", "0:", 100, "100"},
		{"es-ES", "0:", 1000, "1.000"},
		{"es-ES", "0:", 10000000, "10.000.000"},
		{"es-ES", "00:", 1, "01"},
		{"es-ES", "000:", 1, "001"},
		{"es-ES", "0000:", 1, "0001"},
		{"es-ES", "c", 1, "1,00€"},
		{"es-ES", "c", 1.33, "1,33€"},
		{"es-ES", "C", 1.2345, "1,2345€"},
		{"es-ES", "c", 10, "10,00€"},
		{"es-ES", "c", 1000, "1.000,00€"},
		{"es-ES", "c", 10000, "10.000,00€"},
		{"es-ES", "c", 1000000, "1.000.000,00€"},
		{"en-US", "c", 10000000, "$10,000,000.00"},
		{"es-ES", "0.0", 1, "1,0"},
		{"es-ES", "0.0", 1000, "1000,0"},
		{"es-ES", "0:0", 1000, "1.000,0"},
		{"es-ES", "0:00", 1000.2345, "1.000,23"},
		{"es-ES", "0:00", 1000.2, "1.000,20"},
		{"es-ES", "f", 1000.2, "1.000,20"},
		{"en-US", "f", 1000.2, "1,000.20"},
		{"en-US", "c", 1000.2, "$1,000.20"},
		{"en-US", "c", 1000.2, "$1,000.20"},
	}
	for i, d := range data {
		c, err := GetCulture(d.culture)
		if err != nil {
			t.Fatal(err)
		}
		v := Format(d.pattern, d.value, c)
		if v != d.result {
			t.Fatalf("test %d: expected %s, got %s", i, d.result, v)
		}
	}
}
