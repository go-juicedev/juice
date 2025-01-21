/*
Copyright 2025 eatmoreapple

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

/*
Package juice is a lightweight and efficient SQL mapping framework for Go.

It provides a simple way to execute SQL queries and map results to Go structs,
with support for both XML-based SQL configurations and raw SQL statements.

Basic Usage:

	cfg, err := juice.NewXMLConfiguration("config.xml")
	if err != nil {
		// handle error
		panic(err)
	}
	engine, err := juice.New(cfg)
	if err != nil {
		// handle error
		panic(err)
	}
	defer engine.Close()

	rows, err := engine.Raw(`select "hello world"`).Select(context.Background(), nil)
	if err != nil {
		// handle error
		panic(err)
	}
	defer rows.Close()

	result, err := juice.Bind[string](rows)
	if err != nil {
		// handle error
		panic(err)
	}
	fmt.Println(result)

Features:

  - XML-based SQL configuration
  - Raw SQL execution
  - Result mapping to structs
  - Transaction support
  - Generic result binding
  - Parameter binding with #{} syntax
  - Middleware support

For more information and examples, visit: https://github.com/go-juicedev/juice
*/
package juice
