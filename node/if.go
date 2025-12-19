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

// IfNode is an alias for ConditionNode, representing a conditional SQL fragment.
// It evaluates a condition and determines whether its content should be included in the final SQL.
//
// The condition can be based on various types:
//   - Boolean: direct condition
//   - Numbers: non-zero values are true
//   - Strings: non-empty strings are true
//
// Example usage:
//
//	<if test="ID > 0">
//	    AND ID = #{ID}
//	</if>
//
// See ConditionNode for detailed behavior of condition evaluation.
type IfNode = ConditionNode

var _ Node = (*IfNode)(nil)
