package driver

import "testing"

func TestSQLiteDriver_sqlite_test(t *testing.T) {
	driver := SQLiteDriver{}
	translator := driver.Translator()
	if translator.Translate("foo") != "?" {
		t.Fatal("failed to translate")
	}
}
