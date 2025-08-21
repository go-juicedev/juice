/*
Copyright 2024 eatmoreapple

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import "strconv"

// PostgresDriver is a driver of PostgreSQL.
type PostgresDriver struct{}

// Translator is a function to translate a matched string.
func (d PostgresDriver) Translator() Translator {
	var i int
	return TranslateFunc(func(matched string) string {
		i++
		return "$" + strconv.Itoa(i)
	})
}

func (d PostgresDriver) Name() string {
	return "postgres"
}

func (d PostgresDriver) String() string {
	return d.Name()
}

func init() {
	Register("postgres", &PostgresDriver{})
}
