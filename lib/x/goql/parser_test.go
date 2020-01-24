package goql

import (
	"strings"
	"testing"
)

func TestParseUpdateAND(t *testing.T) {
	_, err := ParseQuery("UPDATE type SET weekDays = 0 AND holiday = ? WHERE id = ?")
	if err == nil {
		t.Fatal("Expetected to fail")
	}

	if !strings.Contains(err.Error(), "Unexpected 'AND'") {
		t.Fatal(err)
	}
}

func TestParseUpdateValues2(t *testing.T) {
	q, err := ParseQuery("UPDATE type SET weekDays = (1 + 2)")
	if err != nil {
		t.Fatal(err)
	}

	s, _, err := toSQL(false, q, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}

	if s != "UPDATE type SET weekDays = (1 + 2)" {
		t.Fatal(s)
	}
}
