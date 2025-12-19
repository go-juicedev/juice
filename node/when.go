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

package node

// WhenNode is an alias for ConditionNode, representing a conditional branch
// within a <choose> statement. It evaluates a condition and executes its
// content if the condition is true and it's the first matching condition
// in the choose block.
//
// Behavior:
//   - Evaluates condition using same rules as ConditionNode
//   - Only executes if it's the first true condition in choose
//   - Subsequent true conditions are ignored
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
//
// Supported conditions:
//   - Boolean expressions
//   - Numeric comparisons
//   - String comparisons
//   - Null checks
//   - Property access
//
// Note: Unlike a standalone ConditionNode, WhenNode's execution
// is controlled by its parent ChooseNode and follows choose-when
// semantics similar to switch-case statements.
//
// See ConditionNode for detailed condition evaluation rules.
type WhenNode = ConditionNode

var _ Node = (*WhenNode)(nil)
