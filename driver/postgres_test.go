package driver

import (
	"strconv"
	"testing"
)

func TestPostgresDriver_postgres_test(t *testing.T) {
	driver := PostgresDriver{}
	translator := driver.Translator()
	for i := range 10 {
		if translator.Translate("foo") != "$"+strconv.Itoa(i+1) {
			t.Fatal("failed to translate")
		}
	}
	translator = driver.Translator()
	for i := range 10 {
		if translator.Translate("bar") != "$"+strconv.Itoa(i+1) {
			t.Fatal("failed to translate")
		}
	}
}
