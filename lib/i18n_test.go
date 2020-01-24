package lib

import (
	"testing"
)

func TestCreateCulture(t *testing.T) {
	v := runTest(t, `
		function main() {
		    let c = i18n.addCulture(
					"es-XX",
					":",
					";",
					"EUR",
					"RAND",
					"0:00RAND",
					"0:0000RAND",
					"0:00",
					"dd-MM-yyyy HH:mm",
					"dddd, dd-MM-yyyy HH:mm",
					"dd-MM-yyyy",
					"dddd, dd MMM yyyy",
					"HH:mm",
					"HH:mm:ss",
					time.Monday,
				)

			runtime.context.culture = c
			return i18n.format("c", 1000)
		}
	`)

	if v.ToString() != "1;000:00RAND" {
		t.Fatal(v)
	}
}
