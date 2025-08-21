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

// OracleDriver is a driver of Oracle.
type OracleDriver struct{}

// Translator is a function to translate a matched string.
func (o OracleDriver) Translator() Translator {
	var i int
	return TranslateFunc(func(matched string) string {
		i++
		return ":" + strconv.Itoa(i)
	})
}

func (o OracleDriver) Name() string {
	return "oracle"
}

func (o OracleDriver) String() string {
	return o.Name()
}

func init() {
	Register("oracle", &OracleDriver{})
}
