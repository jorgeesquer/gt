package i18n

import "testing"

func TestParseNumFormat(t *testing.T) {
	data := []struct {
		pattern   string
		left      int
		right     int
		start     int
		end       int
		thousands bool
	}{
		{"0.", 1, 0, 0, 1, false},
		{"0.00", 1, 2, 0, 3, false},
		{"0:00", 1, 2, 0, 3, true},
		{"0000.000", 4, 3, 0, 7, false},
		{"$0:00 USD", 1, 2, 1, 4, true},
		{"0:00€", 1, 2, 0, 3, true},
	}
	for i, d := range data {
		left, right, start, end, thousands := parseNumPattern(d.pattern)
		if left != d.left {
			t.Fatalf("test %d: expected left %d, got %d", i, d.left, left)
		}
		if right != d.right {
			t.Fatalf("test %d: expected right %d, got %d", i, d.right, right)
		}
		if start != d.start {
			t.Fatalf("test %d: expected start %d, got %d", i, d.start, start)
		}
		if end != d.end {
			t.Fatalf("test %d: expected end %d, got %d", i, d.end, end)
		}
		if thousands != d.thousands {
			t.Fatalf("test %d: expected left %t, got %t", i, d.thousands, thousands)
		}
	}
}
