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

package juice

// action defines a sql action.
type action string

const (
	// Select is an action for query
	Select action = "select"

	// Insert is an action for insert
	Insert action = "insert"

	// Update is an action for update
	Update action = "update"

	// Delete is an action for delete
	Delete action = "delete"
)

func (a action) String() string {
	return string(a)
}

func (a action) ForRead() bool {
	return a == Select
}

func (a action) ForWrite() bool {
	return a == Insert || a == Update || a == Delete
}
