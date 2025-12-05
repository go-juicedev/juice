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

package sql

// Action defines a sql Action.
type Action string

const (
	// Select is an Action for query
	Select Action = "select"

	// Insert is an Action for insert
	Insert Action = "insert"

	// Update is an Action for update
	Update Action = "update"

	// Delete is an Action for delete
	Delete Action = "delete"
)

func (a Action) String() string {
	return string(a)
}

func (a Action) ForRead() bool {
	return a == Select
}

func (a Action) ForWrite() bool {
	return a == Insert || a == Update || a == Delete
}
