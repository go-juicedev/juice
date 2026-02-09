package driver

import "testing"

func TestMySQLDriver_mysql_test(t *testing.T) {
	driver := MySQLDriver{}
	translator := driver.Translator()
	if translator.Translate("foo") != "?" {
		t.Fatal("failed to translate")
	}
}
