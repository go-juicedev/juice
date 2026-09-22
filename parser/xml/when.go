/*
Copyright 2023 eatmoreapple

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

package xml

import "github.com/go-juicedev/juice/node"

// WhenNode represents an XML <when> branch inside <choose>.
// It renders its children when its test expression evaluates to true.
// ChooseNode decides which matching branch runs.
//
// Example XML:
//
//	<choose>
//	  <when test='type == "PREMIUM"'>
//	    AND membership_level = 'PREMIUM'
//	  </when>
//	  <when test='type == "BASIC"'>
//	    AND membership_level IN ('BASIC', 'STANDARD')
//	  </when>
//	</choose>
type WhenNode struct {
	ConditionNode
}

// NewWhenNode compiles test into a <when> branch.
// Child nodes and bind nodes are assigned by the caller.
func NewWhenNode(test string) (*WhenNode, error) {
	condition, err := NewConditionNode(test)
	if err != nil {
		return nil, err
	}
	return &WhenNode{ConditionNode: *condition}, nil
}

var _ node.Node = (*WhenNode)(nil)
